package issues

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/internal/pagination"
)

type Controller struct {
	db *bun.DB
}

func NewController(db *bun.DB) *Controller {
	return &Controller{db: db}
}

func (c *Controller) List(ctx context.Context, req *model.ListIssuesRequest) ([]model.Issue, *model.PageInfo, error) {
	page, pageSize := pagination.Normalize(req.Page, req.PageSize)

	q := c.db.NewSelect().Model((*model.Issue)(nil))
	countQ := c.db.NewSelect().Model((*model.Issue)(nil))

	if req.Project != "" {
		q = q.Where("project = ?", req.Project)
		countQ = countQ.Where("project = ?", req.Project)
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
		countQ = countQ.Where("status = ?", req.Status)
	}
	if req.Search != "" {
		term := "%" + req.Search + "%"
		q = q.Where("title ILIKE ?", term)
		countQ = countQ.Where("title ILIKE ?", term)
	}

	var count int
	count, err := countQ.Count(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to count issues: %v", err)
		return nil, nil, errors.New("An unexpected error occurred. Please try again.")
	}

	var issues []model.Issue
	err = q.Order("last_seen DESC").Limit(pageSize).Offset((page-1)*pageSize).Scan(ctx, &issues)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to list issues: %v", err)
		return nil, nil, errors.New("An unexpected error occurred. Please try again.")
	}

	totalPages := (count + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 0
	}
	pageInfo := &model.PageInfo{
		CurrentPage:     page,
		Limit:           pageSize,
		Total:           count,
		TotalPages:      totalPages,
		HasNextPage:     page < totalPages,
		HasPreviousPage: page > 1,
	}

	if len(issues) == 0 {
		pageInfo.CurrentPage = 0
		return []model.Issue{}, pageInfo, nil
	}

	return issues, pageInfo, nil
}

func (c *Controller) GetByID(ctx context.Context, id int64) (*model.Issue, error) {
	var issue model.Issue
	err := c.db.NewSelect().
		Model(&issue).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("Issue not found.")
		}
		logger.Ctx(ctx).Error().Msgf("failed to get issue id=%d: %v", id, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	return &issue, nil
}

func (c *Controller) GetWithEvents(ctx context.Context, id int64, limit int) (*model.Issue, []model.Event, error) {
	issue, err := c.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	if limit < 1 {
		limit = 20
	}

	var events []model.Event
	err = c.db.NewSelect().
		Model(&events).
		Where("issue_id = ?", id).
		Order("timestamp DESC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to get events for issue id=%d: %v", id, err)
		return nil, nil, errors.New("An unexpected error occurred. Please try again.")
	}

	return issue, events, nil
}

func (c *Controller) UpdateStatus(ctx context.Context, id int64, status model.IssueStatus) (*model.Issue, error) {
	issue, err := c.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	issue.Status = status
	issue.UpdatedAt = time.Now()

	_, err = c.db.NewUpdate().
		Model(issue).
		Where("id = ?", issue.ID).
		Exec(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to update issue status id=%d: %v", id, err)
		return nil, errors.New("Failed to update issue.")
	}
	return issue, nil
}

// Merge moves every event of the source issues into the target issue and
// deletes the sources. This is the admin's escape hatch for over-grouped
// issues (distinct bugs collapsed under one fingerprint).
func (c *Controller) Merge(ctx context.Context, sourceIDs []int64, targetID int64) (*model.Issue, error) {
	if len(sourceIDs) == 0 {
		return nil, errors.New("At least one source issue is required.")
	}
	for _, id := range sourceIDs {
		if id == targetID {
			return nil, errors.New("Target issue cannot also be a source.")
		}
	}

	target, err := c.GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}
	for _, id := range sourceIDs {
		if _, err := c.GetByID(ctx, id); err != nil {
			return nil, err
		}
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("merge: failed to begin tx: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	defer tx.Rollback()

	if _, err := tx.NewUpdate().
		Model((*model.Event)(nil)).
		Set("issue_id = ?", targetID).
		Where("issue_id IN (?)", bun.In(sourceIDs)).
		Exec(ctx); err != nil {
		logger.Ctx(ctx).Error().Msgf("merge: failed to reparent events: %v", err)
		return nil, errors.New("Failed to merge issues.")
	}

	if err := c.recomputeAggregates(ctx, tx, []int64{targetID}); err != nil {
		return nil, err
	}

	if _, err := tx.NewDelete().
		Model((*model.Issue)(nil)).
		Where("id IN (?)", bun.In(sourceIDs)).
		Exec(ctx); err != nil {
		logger.Ctx(ctx).Error().Msgf("merge: failed to delete source issues: %v", err)
		return nil, errors.New("Failed to merge issues.")
	}

	if err := tx.Commit(); err != nil {
		logger.Ctx(ctx).Error().Msgf("merge: failed to commit: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	return c.GetByID(ctx, target.ID)
}

// MoveEvents reparents events to an existing issue (targetID > 0) or into a
// newly created issue. Counts and first/last seen are recomputed for every
// affected issue. This is the escape hatch for under-grouped issues (one bug
// spread across several issues).
func (c *Controller) MoveEvents(ctx context.Context, eventIDs []int64, targetID int64, title string) (*model.Issue, error) {
	if len(eventIDs) == 0 {
		return nil, errors.New("At least one event is required.")
	}

	var sources []int64
	if err := c.db.NewSelect().
		Model((*model.Event)(nil)).
		ColumnExpr("DISTINCT issue_id").
		Where("id IN (?)", bun.In(eventIDs)).
		Scan(ctx, &sources); err != nil {
		logger.Ctx(ctx).Error().Msgf("move: failed to resolve source issues: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	if len(sources) == 0 {
		return nil, errors.New("No such events.")
	}

	target := &model.Issue{}
	if targetID > 0 {
		existing, err := c.GetByID(ctx, targetID)
		if err != nil {
			return nil, err
		}
		target = existing
	} else {
		created, err := c.createSplitIssue(ctx, eventIDs, title)
		if err != nil {
			return nil, err
		}
		target = created
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("move: failed to begin tx: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	defer tx.Rollback()

	if _, err := tx.NewUpdate().
		Model((*model.Event)(nil)).
		Set("issue_id = ?", target.ID).
		Where("id IN (?)", bun.In(eventIDs)).
		Exec(ctx); err != nil {
		logger.Ctx(ctx).Error().Msgf("move: failed to reparent events: %v", err)
		return nil, errors.New("Failed to move events.")
	}

	affected := dedupeIDs(append(sources, target.ID))
	if err := c.recomputeAggregates(ctx, tx, affected); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		logger.Ctx(ctx).Error().Msgf("move: failed to commit: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	return c.GetByID(ctx, target.ID)
}

// createSplitIssue builds a new open issue for events being split out of an
// existing one. A random fingerprint is used so future events never land in it
// automatically; the admin explicitly curates the group.
func (c *Controller) createSplitIssue(ctx context.Context, eventIDs []int64, title string) (*model.Issue, error) {
	var first model.Event
	if err := c.db.NewSelect().
		Model(&first).
		Where("id IN (?)", bun.In(eventIDs)).
		Order("timestamp ASC").
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("No such events.")
		}
		logger.Ctx(ctx).Error().Msgf("move: failed to load representative event: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	name := strings.TrimSpace(title)
	if name == "" {
		name = truncateTitle(first.Message)
	}
	if name == "" {
		name = fmt.Sprintf("Split from issue %d", first.IssueID)
	}

	fingerprint, err := randomFingerprint()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	created := &model.Issue{
		Fingerprint: fingerprint,
		Title:       name,
		Project:     first.Project,
		Tag:         first.Tag,
		Status:      model.IssueStatusOpen,
		FirstSeen:   now,
		LastSeen:    now,
		Count:       0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := c.db.NewInsert().Model(created).Returning("*").Exec(ctx); err != nil {
		logger.Ctx(ctx).Error().Msgf("move: failed to create split issue: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	return created, nil
}

// recomputeAggregates refreshes count/first_seen/last_seen for the given
// issues from their current events. Must run inside a transaction.
func (c *Controller) recomputeAggregates(ctx context.Context, tx bun.Tx, issueIDs []int64) error {
	for _, id := range issueIDs {
		var agg struct {
			Count int64
			First time.Time
			Last  time.Time
		}
		if err := tx.NewSelect().
			Model((*model.Event)(nil)).
			ColumnExpr("COUNT(*) AS count, COALESCE(MIN(timestamp), now()) AS first, COALESCE(MAX(timestamp), now()) AS last").
			Where("issue_id = ?", id).
			Scan(ctx, &agg); err != nil {
			logger.Ctx(ctx).Error().Msgf("recompute aggregate for issue %d failed: %v", id, err)
			return errors.New("An unexpected error occurred. Please try again.")
		}
		if _, err := tx.NewUpdate().
			Model((*model.Issue)(nil)).
			Set("count = ?", agg.Count).
			Set("first_seen = ?", agg.First).
			Set("last_seen = ?", agg.Last).
			Set("updated_at = ?", time.Now()).
			Where("id = ?", id).
			Exec(ctx); err != nil {
			logger.Ctx(ctx).Error().Msgf("recompute update for issue %d failed: %v", id, err)
			return errors.New("An unexpected error occurred. Please try again.")
		}
	}
	return nil
}

func randomFingerprint() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		logger.Error().Msgf("failed to generate fingerprint: %v", err)
		return "", errors.New("An unexpected error occurred. Please try again.")
	}
	return hex.EncodeToString(b), nil
}

func dedupeIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func truncateTitle(s string) string {
	if len(s) <= 200 {
		return s
	}
	return s[:200] + "..."
}
