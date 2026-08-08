package issues

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/internal/model"
)

const defaultPageSize = 10

type Controller struct {
	db *bun.DB
}

func NewController(db *bun.DB) *Controller {
	return &Controller{db: db}
}

func (c *Controller) List(ctx context.Context, req *model.ListIssuesRequest) ([]model.Issue, *model.PageInfo, error) {
	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}

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
