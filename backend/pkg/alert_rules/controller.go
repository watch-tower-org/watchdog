package alert_rules

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

func (c *Controller) List(ctx context.Context, req *model.ListAlertRulesRequest) ([]model.AlertRule, *model.PageInfo, error) {
	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}

	q := c.db.NewSelect().Model((*model.AlertRule)(nil)).Relation("RecipientList")
	countQ := c.db.NewSelect().Model((*model.AlertRule)(nil))

	if req.Project != "" {
		q = q.Where("project = ?", req.Project)
		countQ = countQ.Where("project = ?", req.Project)
	}

	var count int
	count, err := countQ.Count(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to count alert rules: %v", err)
		return nil, nil, errors.New("An unexpected error occurred. Please try again.")
	}

	var rules []model.AlertRule
	err = q.Order("id DESC").Limit(pageSize).Offset((page-1)*pageSize).Scan(ctx, &rules)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to list alert rules: %v", err)
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

	if len(rules) == 0 {
		pageInfo.CurrentPage = 0
		return []model.AlertRule{}, pageInfo, nil
	}

	return rules, pageInfo, nil
}

func (c *Controller) GetByID(ctx context.Context, id int64) (*model.AlertRule, error) {
	rule := &model.AlertRule{ID: id}
	err := c.db.NewSelect().
		Model(rule).
		Relation("RecipientList").
		WherePK().
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("Alert rule not found.")
		}
		logger.Ctx(ctx).Error().Msgf("failed to get alert rule id=%d: %v", id, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	return rule, nil
}

func (c *Controller) recipientListExists(ctx context.Context, id int64) (bool, error) {
	exists, err := c.db.NewSelect().
		Model((*model.RecipientList)(nil)).
		Where("id = ?", id).
		Exists(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to check recipient list id=%d: %v", id, err)
		return false, errors.New("An unexpected error occurred. Please try again.")
	}
	return exists, nil
}

func (c *Controller) Create(ctx context.Context, req *model.CreateAlertRuleRequest) (*model.AlertRule, error) {
	exists, err := c.recipientListExists(ctx, req.RecipientListID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("Recipient list does not exist.")
	}

	rule := &model.AlertRule{
		Name:            req.Name,
		TriggerType:     req.TriggerType,
		Project:         req.Project,
		Tag:             req.Tag,
		Threshold:       req.Threshold,
		WindowMinutes:   req.WindowMinutes,
		ThrottleWindow:  req.ThrottleWindow,
		RecipientListID: req.RecipientListID,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}

	_, err = c.db.NewInsert().Model(rule).Returning("*").Exec(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to create alert rule: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	return c.GetByID(ctx, rule.ID)
}

func (c *Controller) Update(ctx context.Context, id int64, req *model.UpdateAlertRuleRequest) (*model.AlertRule, error) {
	rule, err := c.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.TriggerType != nil {
		rule.TriggerType = *req.TriggerType
	}
	if req.Project != nil {
		rule.Project = *req.Project
	}
	if req.Tag != nil {
		rule.Tag = *req.Tag
	}
	if req.Threshold != nil {
		rule.Threshold = *req.Threshold
	}
	if req.WindowMinutes != nil {
		rule.WindowMinutes = *req.WindowMinutes
	}
	if req.ThrottleWindow != nil {
		rule.ThrottleWindow = *req.ThrottleWindow
	}
	if req.RecipientListID != nil {
		exists, err := c.recipientListExists(ctx, *req.RecipientListID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("Recipient list does not exist.")
		}
		rule.RecipientListID = *req.RecipientListID
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}

	rule.UpdatedAt = time.Now()

	_, err = c.db.NewUpdate().
		Model(rule).
		Where("id = ?", rule.ID).
		Exec(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to update alert rule id=%d: %v", id, err)
		return nil, errors.New("Failed to update alert rule.")
	}

	return c.GetByID(ctx, rule.ID)
}

func (c *Controller) Delete(ctx context.Context, id int64) error {
	res, err := c.db.NewDelete().
		Model((*model.AlertRule)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to delete alert rule id=%d: %v", id, err)
		return errors.New("Failed to delete alert rule.")
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return errors.New("Alert rule not found.")
	}
	return nil
}
