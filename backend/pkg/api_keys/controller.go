package api_keys

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/internal/model"
)

const (
	defaultPageSize = 10
	keyPrefix       = "wt_"
)

type Controller struct {
	db *bun.DB
}

func NewController(db *bun.DB) *Controller {
	return &Controller{db: db}
}

// GenerateKey creates a random API key with the wt_ prefix.
func (c *Controller) GenerateKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		logger.Error().Msgf("GenerateKey: rand failed: %v", err)
		return "", errors.New("An unexpected error occurred. Please try again.")
	}
	return keyPrefix + hex.EncodeToString(buf), nil
}

func HashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// MaskKey returns the display form wt_...abcd (prefix + last 4 chars).
func MaskKey(key string) string {
	if len(key) <= 4 {
		return key
	}
	return keyPrefix + "..." + key[len(key)-4:]
}

func (c *Controller) List(ctx context.Context, page, pageSize int) ([]model.ApiKey, *model.PageInfo, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}

	q := c.db.NewSelect().Model((*model.ApiKey)(nil))
	countQ := c.db.NewSelect().Model((*model.ApiKey)(nil))

	var count int
	count, err := countQ.Count(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to count api keys: %v", err)
		return nil, nil, errors.New("An unexpected error occurred. Please try again.")
	}

	var keys []model.ApiKey
	err = q.Order("id ASC").Limit(pageSize).Offset((page-1)*pageSize).Scan(ctx, &keys)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to get api keys: %v", err)
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

	if len(keys) == 0 {
		pageInfo.CurrentPage = 0
		return []model.ApiKey{}, pageInfo, nil
	}

	return keys, pageInfo, nil
}

func (c *Controller) Create(ctx context.Context, req *model.CreateApiKeyRequest) (*model.CreateApiKeyResponse, error) {
	key, err := c.GenerateKey()
	if err != nil {
		return nil, err
	}

	ak := &model.ApiKey{
		Name:      req.Name,
		KeyHash:   HashKey(key),
		Masked:    MaskKey(key),
		Project:   req.Project,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = c.db.NewInsert().Model(ak).Returning("*").Exec(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to create api key: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	return &model.CreateApiKeyResponse{
		ApiKey: *ak,
		Key:    key,
		KeyID:  ak.ID,
	}, nil
}

func (c *Controller) Revoke(ctx context.Context, id int64) error {
	ak, err := c.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !ak.IsActive {
		return errors.New("API key is already revoked.")
	}

	ak.IsActive = false
	ak.UpdatedAt = time.Now()

	_, err = c.db.NewUpdate().
		Model(ak).
		Where("id = ?", ak.ID).
		Exec(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to revoke api key id=%d: %v", id, err)
		return errors.New("Failed to revoke API key.")
	}

	return nil
}

func (c *Controller) GetByID(ctx context.Context, id int64) (*model.ApiKey, error) {
	var ak model.ApiKey
	err := c.db.NewSelect().
		Model(&ak).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to retrieve api key id=%d: %v", id, err)
		return nil, errors.New("Failed to retrieve API key.")
	}

	return &ak, nil
}

func (c *Controller) Update(ctx context.Context, id int64, req *model.UpdateApiKeyRequest) (*model.ApiKey, error) {
	ak, err := c.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		ak.Name = *req.Name
	}
	if req.Project != nil {
		ak.Project = *req.Project
	}

	ak.UpdatedAt = time.Now()

	_, err = c.db.NewUpdate().
		Model(ak).
		Where("id = ?", ak.ID).
		Exec(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to update api key id=%d: %v", id, err)
		return nil, errors.New("Failed to update API key.")
	}

	return ak, nil
}
