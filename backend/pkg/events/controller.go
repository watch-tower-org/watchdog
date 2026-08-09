package events

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

const defaultPageSize = 10

type Controller struct {
	db *bun.DB
}

func NewController(db *bun.DB) *Controller {
	return &Controller{db: db}
}

func (c *Controller) List(ctx context.Context, req *model.ListEventsRequest) ([]model.Event, *model.PageInfo, error) {
	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}

	q := c.db.NewSelect().Model((*model.Event)(nil))
	countQ := c.db.NewSelect().Model((*model.Event)(nil))

	if req.IssueID > 0 {
		q = q.Where("issue_id = ?", req.IssueID)
		countQ = countQ.Where("issue_id = ?", req.IssueID)
	}
	if req.Project != "" {
		q = q.Where("project = ?", req.Project)
		countQ = countQ.Where("project = ?", req.Project)
	}

	var count int
	count, err := countQ.Count(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to count events: %v", err)
		return nil, nil, errors.New("An unexpected error occurred. Please try again.")
	}

	var events []model.Event
	err = q.Order("timestamp DESC").Limit(pageSize).Offset((page-1)*pageSize).Scan(ctx, &events)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to list events: %v", err)
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

	if len(events) == 0 {
		pageInfo.CurrentPage = 0
		return []model.Event{}, pageInfo, nil
	}

	return events, pageInfo, nil
}

func (c *Controller) GetByID(ctx context.Context, id int64) (*model.Event, error) {
	var event model.Event
	err := c.db.NewSelect().
		Model(&event).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("Event not found.")
		}
		logger.Ctx(ctx).Error().Msgf("failed to get event id=%d: %v", id, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	return &event, nil
}
