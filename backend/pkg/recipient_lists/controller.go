package recipient_lists

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/cache"
	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/internal/pagination"
)

type Controller struct {
	db    *bun.DB
	cache *cache.MapCache[int64, model.RecipientList]
}

func NewController(db *bun.DB, cache *cache.MapCache[int64, model.RecipientList]) *Controller {
	return &Controller{db: db, cache: cache}
}

func (c *Controller) GetByID(ctx context.Context, id int64) (*model.RecipientList, error) {
	if rl, ok := c.cache.Get(id); ok {
		return rl, nil
	}

	var rl model.RecipientList
	err := c.db.NewSelect().
		Model(&rl).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to retrieve recipient list id=%d: %v", id, err)
		return nil, errors.New("Failed to retrieve recipient list.")
	}

	c.cache.Add(id, &rl)

	return &rl, nil
}

func (c *Controller) List(ctx context.Context, search string, page, pageSize int) ([]model.RecipientList, *model.PageInfo, error) {
	page, pageSize = pagination.Normalize(page, pageSize)

	q := c.db.NewSelect().Model((*model.RecipientList)(nil))
	countQ := c.db.NewSelect().Model((*model.RecipientList)(nil))

	if search != "" {
		term := "%" + search + "%"
		q = q.Where("name ILIKE ?", term)
		countQ = countQ.Where("name ILIKE ?", term)
	}

	var count int
	count, err := countQ.Count(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to count recipient lists: %v", err)
		return nil, nil, errors.New("An unexpected error occurred. Please try again.")
	}

	var lists []model.RecipientList
	err = q.Order("id ASC").Limit(pageSize).Offset((page-1)*pageSize).Scan(ctx, &lists)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to get recipient lists: %v", err)
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

	if len(lists) == 0 {
		pageInfo.CurrentPage = 0
		return []model.RecipientList{}, pageInfo, nil
	}

	return lists, pageInfo, nil
}

func (c *Controller) Create(ctx context.Context, req *model.CreateRecipientListRequest) (*model.RecipientList, error) {
	rl := &model.RecipientList{
		Name:      req.Name,
		Emails:    req.Emails,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := c.db.NewInsert().Model(rl).Returning("*").Exec(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to create recipient list: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	c.cache.Invalidate()

	return rl, nil
}

func (c *Controller) Update(ctx context.Context, id int64, req *model.UpdateRecipientListRequest) (*model.RecipientList, error) {
	rl, err := c.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		rl.Name = *req.Name
	}
	if req.Emails != nil {
		rl.Emails = *req.Emails
	}

	rl.UpdatedAt = time.Now()

	_, err = c.db.NewUpdate().
		Model(rl).
		Where("id = ?", rl.ID).
		Exec(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to update recipient list id=%d: %v", id, err)
		return nil, errors.New("Failed to update recipient list.")
	}

	c.cache.Invalidate()

	return rl, nil
}

func (c *Controller) Delete(ctx context.Context, id int64) error {
	_, err := c.db.NewDelete().
		Model((*model.RecipientList)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to delete recipient list id=%d: %v", id, err)
		return errors.New("Failed to delete recipient list.")
	}

	c.cache.Invalidate()

	return nil
}
