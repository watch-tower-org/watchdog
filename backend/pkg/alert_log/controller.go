package alert_log

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

func (c *Controller) List(ctx context.Context, req *model.ListAlertLogsRequest) ([]model.AlertLog, *model.PageInfo, error) {
	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}

	q := c.db.NewSelect().Model((*model.AlertLog)(nil)).Relation("Issue").Relation("Rule")
	countQ := c.db.NewSelect().Model((*model.AlertLog)(nil))

	if req.IssueID > 0 {
		q = q.Where("issue_id = ?", req.IssueID)
		countQ = countQ.Where("issue_id = ?", req.IssueID)
	}
	if req.RuleID > 0 {
		q = q.Where("rule_id = ?", req.RuleID)
		countQ = countQ.Where("rule_id = ?", req.RuleID)
	}

	var count int
	count, err := countQ.Count(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to count alert logs: %v", err)
		return nil, nil, errors.New("An unexpected error occurred. Please try again.")
	}

	var logs []model.AlertLog
	err = q.Order("sent_at DESC").Limit(pageSize).Offset((page-1)*pageSize).Scan(ctx, &logs)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to list alert logs: %v", err)
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

	if len(logs) == 0 {
		pageInfo.CurrentPage = 0
		return []model.AlertLog{}, pageInfo, nil
	}

	return logs, pageInfo, nil
}

func (c *Controller) GetByID(ctx context.Context, id int64) (*model.AlertLog, error) {
	log := &model.AlertLog{ID: id}
	err := c.db.NewSelect().
		Model(log).
		Relation("Issue").
		Relation("Rule").
		WherePK().
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("Alert log entry not found.")
		}
		logger.Ctx(ctx).Error().Msgf("failed to get alert log id=%d: %v", id, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	return log, nil
}
