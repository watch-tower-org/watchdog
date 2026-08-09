package ingestion

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/pkg/notifier"
)

type Controller struct {
	db       *bun.DB
	notifier *notifier.Notifier
	mode     FingerprintMode
}

// NewController creates the ingestion controller. mode controls the dedup key
// (see FingerprintMode); an empty mode uses ModeTypeAndFrames.
func NewController(db *bun.DB, n *notifier.Notifier, mode FingerprintMode) *Controller {
	return &Controller{db: db, notifier: n, mode: ParseFingerprintMode(string(mode))}
}

// Ingest deduplicates a single event into an issue and stores it. Returns the
// resulting issue/event ids and whether a new issue was created. After commit
// the notifier is asked (asynchronously) to evaluate alert rules.
func (c *Controller) Ingest(ctx context.Context, req *model.IngestEventRequest) (*model.IngestResult, error) {
	errorType := ResolveErrorType(req)
	fingerprint := ComputeFingerprintMode(req.Project, errorType, req.StackTrace, req.Message, c.mode)
	title := ExtractTitle(req)

	ts := time.Now()
	if req.Timestamp != nil {
		ts = *req.Timestamp
	}

	res, err := c.ingestTx(ctx, req, fingerprint, title, ts)
	if err != nil {
		return nil, err
	}
	res.Fingerprint = fingerprint

	if c.notifier != nil {
		c.notifier.NotifyAsync(res.IssueID, res.IsNewIssue, res.WasRegression, req.Project, req.Tag)
	}

	return res, nil
}

func (c *Controller) ingestTx(ctx context.Context, req *model.IngestEventRequest, fingerprint, title string, ts time.Time) (*model.IngestResult, error) {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to begin ingestion tx: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	defer tx.Rollback()

	issue, err := c.findOrCreateIssue(ctx, tx, req, fingerprint, title)
	if err != nil {
		return nil, err
	}

	event := &model.Event{
		IssueID:    issue.ID,
		Timestamp:  ts,
		Message:    req.Message,
		StackTrace: req.StackTrace,
		Context:    req.Context,
		Project:    req.Project,
		Tag:        req.Tag,
		CreatedAt:  ts,
	}
	if _, err := tx.NewInsert().Model(event).Returning("*").Exec(ctx); err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to insert event: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	if err := tx.Commit(); err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to commit ingestion tx: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	return &model.IngestResult{
		IssueID:       issue.ID,
		EventID:       event.ID,
		IsNewIssue:    issue.IsNew,
		WasRegression: issue.WasRegression,
	}, nil
}

type issueWithNew struct {
	model.Issue
	IsNew         bool
	WasRegression bool
}

// findOrCreateIssue looks up the issue by fingerprint. On a hit it bumps count,
// refreshes last_seen and reopens the issue if it was resolved (regression). On
// a miss it inserts a new open issue, handling the concurrent-create race via
// ON CONFLICT DO NOTHING.
func (c *Controller) findOrCreateIssue(ctx context.Context, tx bun.Tx, req *model.IngestEventRequest, fingerprint, title string) (*issueWithNew, error) {
	var issue model.Issue
	err := tx.NewSelect().
		Model(&issue).
		Where("fingerprint = ?", fingerprint).
		Limit(1).
		Scan(ctx)

	if err == nil {
		issue.Count++
		issue.LastSeen = time.Now()
		wasRegression := false
		if issue.Status == model.IssueStatusResolved {
			issue.Status = model.IssueStatusOpen
			wasRegression = true
		}
		issue.UpdatedAt = time.Now()

		if _, err := tx.NewUpdate().
			Model(&issue).
			Where("id = ?", issue.ID).
			Set("count = count + 1").
			Set("last_seen = ?", issue.LastSeen).
			Set("status = ?", issue.Status).
			Set("updated_at = ?", issue.UpdatedAt).
			Exec(ctx); err != nil {
			logger.Ctx(ctx).Error().Msgf("failed to update issue id=%d: %v", issue.ID, err)
			return nil, errors.New("An unexpected error occurred. Please try again.")
		}
		return &issueWithNew{Issue: issue, WasRegression: wasRegression}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		logger.Ctx(ctx).Error().Msgf("failed to select issue by fingerprint: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	newIssue := &model.Issue{
		Fingerprint: fingerprint,
		Title:       title,
		Project:     req.Project,
		Tag:         req.Tag,
		Status:      model.IssueStatusOpen,
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
		Count:       1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	res, err := tx.NewInsert().
		Model(newIssue).
		On("CONFLICT (fingerprint) DO NOTHING").
		Returning("id, created_at").
		Exec(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to insert issue: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	affected, _ := res.RowsAffected()
	if affected == 1 {
		return &issueWithNew{Issue: *newIssue, IsNew: true}, nil
	}

	// Lost the create race — another request inserted the issue. Reload it.
	var existing model.Issue
	if err := tx.NewSelect().
		Model(&existing).
		Where("fingerprint = ?", fingerprint).
		Limit(1).
		Scan(ctx); err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to reload issue after conflict: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	existing.Count++
	existing.LastSeen = time.Now()
	wasRegression := false
	if existing.Status == model.IssueStatusResolved {
		existing.Status = model.IssueStatusOpen
		wasRegression = true
	}
	existing.UpdatedAt = time.Now()

	if _, err := tx.NewUpdate().
		Model(&existing).
		Where("id = ?", existing.ID).
		Set("count = count + 1").
		Set("last_seen = ?", existing.LastSeen).
		Set("status = ?", existing.Status).
		Set("updated_at = ?", existing.UpdatedAt).
		Exec(ctx); err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to update issue id=%d after conflict: %v", existing.ID, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	return &issueWithNew{Issue: existing, WasRegression: wasRegression}, nil
}

// IngestBatch ingests multiple events, returning per-event results. Each event
// is committed independently so a single bad event in a batch can't roll back
// the others; batches are expected to be small (SDK flush size).
func (c *Controller) IngestBatch(ctx context.Context, reqs []*model.IngestEventRequest) ([]*model.IngestResult, error) {
	results := make([]*model.IngestResult, 0, len(reqs))
	for _, req := range reqs {
		res, err := c.Ingest(ctx, req)
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}
	return results, nil
}
