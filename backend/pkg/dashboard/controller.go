package dashboard

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/logger"
)

type Controller struct {
	db *bun.DB
}

func NewController(db *bun.DB) *Controller {
	return &Controller{db: db}
}

type Summary struct {
	RecipientLists int64 `json:"recipient_lists"`
	ApiKeys        int64 `json:"api_keys"`
	Issues         int64 `json:"issues"`
	Events         int64 `json:"events"`
	Alerts         int64 `json:"alerts"`
}

// TrendBucket is one time bucket in the dashboard trend series.
type TrendBucket struct {
	TS        time.Time `json:"ts"`
	NewIssues int64     `json:"new_issues"`
	Events    int64     `json:"events"`
	Alerts    int64     `json:"alerts"`
}

// TrendItem is a ranked breakdown entry (e.g. top project, top tag, status).
type TrendItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// Trends is the full dashboard analytics payload for a time range.
type Trends struct {
	Range          string        `json:"range"`
	Buckets        []TrendBucket `json:"buckets"`
	IssuesByStatus []TrendItem   `json:"issues_by_status"`
	TopProjects    []TrendItem   `json:"top_projects"`
	TopTags        []TrendItem   `json:"top_tags"`
}

// bucketCount is the scanned shape of a GROUP BY time-truncated query.
type bucketCount struct {
	Bucket time.Time `bun:"bucket"`
	Count  int64     `bun:"count"`
}

// rangeSpec describes a supported trend range: the lookback window, the time
// bucket step, and the SQL date_trunc unit used to align buckets.
type rangeSpec struct {
	window time.Duration
	step   time.Duration
	trunc  string
}

var rangeSpecs = map[string]rangeSpec{
	"24h": {window: 24 * time.Hour, step: time.Hour, trunc: "hour"},
	"7d":  {window: 7 * 24 * time.Hour, step: 24 * time.Hour, trunc: "day"},
	"30d": {window: 30 * 24 * time.Hour, step: 24 * time.Hour, trunc: "day"},
}

func (c *Controller) GetSummary(ctx context.Context) (*Summary, error) {
	summary := &Summary{}

	counts := []struct {
		table string
		dst   *int64
	}{
		{"recipient_lists", &summary.RecipientLists},
		{"api_keys", &summary.ApiKeys},
		{"issues", &summary.Issues},
		{"events", &summary.Events},
		{"alert_log", &summary.Alerts},
	}

	for _, cnt := range counts {
		err := c.db.NewRaw("SELECT COUNT(*) FROM "+cnt.table).Scan(ctx, cnt.dst)
		if err != nil {
			logger.Ctx(ctx).Error().Msgf("failed to count %s: %v", cnt.table, err)
			return nil, errors.New("An unexpected error occurred. Please try again.")
		}
	}

	return summary, nil
}

// GetTrends builds the analytics payload for the dashboard: a dense time series
// of new issues / events / alerts, plus status, project, and tag breakdowns.
// All times are bucketed in UTC so the series is stable regardless of the
// database session timezone.
func (c *Controller) GetTrends(ctx context.Context, rangeKey string) (*Trends, error) {
	spec, ok := rangeSpecs[rangeKey]
	if !ok {
		return nil, fmt.Errorf("invalid range %q", rangeKey)
	}

	now := time.Now().UTC()
	// Align the window start to the bucket granularity (top of hour / UTC
	// midnight) so the Go-generated bucket keys match date_trunc(...) output
	// exactly; otherwise real data would never land in a bucket.
	start := now.Add(-spec.window).Truncate(spec.step)
	trends := &Trends{Range: rangeKey}

	events, err := c.countBuckets(ctx, "events", "created_at", spec.trunc, start)
	if err != nil {
		return nil, err
	}
	issues, err := c.countBuckets(ctx, "issues", "first_seen", spec.trunc, start)
	if err != nil {
		return nil, err
	}
	alerts, err := c.countBuckets(ctx, "alert_log", "sent_at", spec.trunc, start)
	if err != nil {
		return nil, err
	}

	trends.Buckets = buildSeries(start, spec.step, now, events, issues, alerts)

	// Slices are pre-allocated so they marshal as [] instead of null when the
	// database is empty; the frontend expects arrays for these fields.
	statusCounts := make([]TrendItem, 0)
	if err := c.db.NewRaw(`SELECT status AS name, count(*) AS count FROM issues GROUP BY status ORDER BY count DESC`).Scan(ctx, &statusCounts); err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to load issues by status: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	trends.IssuesByStatus = statusCounts

	topProjects := make([]TrendItem, 0)
	if err := c.db.NewRaw(`SELECT project AS name, count(*) AS count FROM issues WHERE status = 'open' GROUP BY project ORDER BY count DESC LIMIT 5`).Scan(ctx, &topProjects); err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to load top projects: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	trends.TopProjects = topProjects

	topTags := make([]TrendItem, 0)
	if err := c.db.NewRaw(`SELECT tag AS name, count(*) AS count FROM issues WHERE status = 'open' AND tag <> '' GROUP BY tag ORDER BY count DESC LIMIT 5`).Scan(ctx, &topTags); err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to load top tags: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	trends.TopTags = topTags

	return trends, nil
}

// countBuckets runs a time-truncated GROUP BY over the given table/column and
// returns the per-bucket counts. Columns are timestamptz; they are converted to
// UTC before truncation so bucket boundaries match buildSeries.
func (c *Controller) countBuckets(ctx context.Context, table, column, trunc string, start time.Time) (map[time.Time]int64, error) {
	var rows []bucketCount
	query := fmt.Sprintf(
		"SELECT date_trunc('%s', %s AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' AS bucket, count(*) AS count "+
			"FROM %s WHERE %s >= ? GROUP BY bucket ORDER BY bucket",
		trunc, column, table, column,
	)
	if err := c.db.NewRaw(query, start).Scan(ctx, &rows); err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to load %s trend series: %v", table, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	counts := make(map[time.Time]int64, len(rows))
	for _, r := range rows {
		counts[r.Bucket.UTC()] = r.Count
	}
	return counts, nil
}

// buildSeries merges the per-table bucket maps into a dense, ordered series
// (every bucket from start to now, zero-filled) so the frontend gets a
// continuous line with no gaps.
func buildSeries(start time.Time, step time.Duration, now time.Time, events, issues, alerts map[time.Time]int64) []TrendBucket {
	var out []TrendBucket
	for t := start.UTC(); !t.After(now); t = t.Add(step) {
		b := TrendBucket{TS: t}
		if v, ok := issues[t]; ok {
			b.NewIssues = v
		}
		if v, ok := events[t]; ok {
			b.Events = v
		}
		if v, ok := alerts[t]; ok {
			b.Alerts = v
		}
		out = append(out, b)
	}
	return out
}
