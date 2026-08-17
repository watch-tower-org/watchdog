package uptime

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
	"github.com/watch-tower-org/watchtower/backend/internal/pagination"
)

const (
	defaultWorkers          = 2
	defaultDispatchInterval = 5 * time.Second
	defaultTimeout          = 10 * time.Second
	defaultQueueSize        = 1000
	checkHistoryLimit       = 1000
	defaultRecentChecks     = 10
	shutdownTimeout         = 10 * time.Second
)

// ErrNotFound is returned when a monitored service does not exist.
var ErrNotFound = errors.New("monitored service not found")

// Controller owns both the CRUD layer and the background scheduler/worker pool
// that poll monitored services on their own schedules.
type Controller struct {
	db               *bun.DB
	workers          int
	dispatchInterval time.Duration
	defaultTimeout   time.Duration
	jobs             chan int64
	stop             chan struct{}
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

func NewController(db *bun.DB, cfg config.UptimeConfig) *Controller {
	workers := cfg.Workers
	if workers < 1 {
		workers = defaultWorkers
	}
	interval := cfg.DispatchInterval
	if interval < 1 {
		interval = defaultDispatchInterval
	}
	timeout := cfg.DefaultTimeout
	if timeout < 1 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Controller{
		db:               db,
		workers:          workers,
		dispatchInterval: interval,
		defaultTimeout:   timeout,
		jobs:             make(chan int64, defaultQueueSize),
		stop:             make(chan struct{}),
		ctx:              ctx,
		cancel:           cancel,
	}
}

func (c *Controller) GetByID(ctx context.Context, id int64) (*model.MonitoredService, error) {
	var svc model.MonitoredService
	err := c.db.NewSelect().
		Model(&svc).
		Relation("RecipientList").
		Where("ms.id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		logger.Ctx(ctx).Error().Msgf("failed to retrieve monitored service id=%d: %v", id, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	return &svc, nil
}

func (c *Controller) List(ctx context.Context, search string, page, pageSize int) ([]model.MonitoredService, *model.PageInfo, error) {
	page, pageSize = pagination.Normalize(page, pageSize)

	q := c.db.NewSelect().Model((*model.MonitoredService)(nil)).Relation("RecipientList")
	countQ := c.db.NewSelect().Model((*model.MonitoredService)(nil))

	if search != "" {
		term := "%" + search + "%"
		q = q.Where("(name ILIKE ? OR url ILIKE ?)", term, term)
		countQ = countQ.Where("(name ILIKE ? OR url ILIKE ?)", term, term)
	}

	var count int
	count, err := countQ.Count(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to count monitored services: %v", err)
		return nil, nil, errors.New("An unexpected error occurred. Please try again.")
	}

	var services []model.MonitoredService
	err = q.Order("name ASC").Limit(pageSize).Offset((page-1)*pageSize).Scan(ctx, &services)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to get monitored services: %v", err)
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

	if len(services) == 0 {
		pageInfo.CurrentPage = 0
		return []model.MonitoredService{}, pageInfo, nil
	}

	return services, pageInfo, nil
}

func (c *Controller) RecentChecks(ctx context.Context, serviceID int64, limit int) ([]model.UptimeCheck, error) {
	if limit < 1 {
		limit = defaultRecentChecks
	}
	var checks []model.UptimeCheck
	err := c.db.NewSelect().
		Model(&checks).
		Where("service_id = ?", serviceID).
		Order("checked_at DESC", "id DESC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to get recent checks for service id=%d: %v", serviceID, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}
	return checks, nil
}

func (c *Controller) Create(ctx context.Context, req *model.CreateMonitoredServiceRequest) (*model.MonitoredService, error) {
	interval := 60
	if req.IntervalSeconds != nil {
		interval = *req.IntervalSeconds
	}
	timeout := 10
	if req.TimeoutSeconds != nil {
		timeout = *req.TimeoutSeconds
	}
	failures := 1
	if req.FailuresBeforeAlert != nil {
		failures = *req.FailuresBeforeAlert
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}

	if err := validateInput(req.Name, req.URL, interval, timeout, failures); err != nil {
		return nil, err
	}

	svc := &model.MonitoredService{
		Name:                req.Name,
		URL:                 req.URL,
		IntervalSeconds:     interval,
		TimeoutSeconds:      timeout,
		FailuresBeforeAlert: failures,
		RecipientListID:     req.RecipientListID,
		IsActive:            active,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if _, err := c.db.NewInsert().Model(svc).Returning("*").Exec(ctx); err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to create monitored service: %v", err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	return svc, nil
}

func (c *Controller) Update(ctx context.Context, id int64, req *model.UpdateMonitoredServiceRequest) (*model.MonitoredService, error) {
	svc, err := c.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		svc.Name = *req.Name
	}
	if req.URL != nil {
		svc.URL = *req.URL
	}
	if req.IntervalSeconds != nil {
		svc.IntervalSeconds = *req.IntervalSeconds
	}
	if req.TimeoutSeconds != nil {
		svc.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.FailuresBeforeAlert != nil {
		svc.FailuresBeforeAlert = *req.FailuresBeforeAlert
	}
	if req.RecipientListID != nil {
		// Non-nil outer pointer means the field was present in the payload:
		// inner nil clears the list, inner value sets it.
		svc.RecipientListID = *req.RecipientListID
	}
	if req.IsActive != nil {
		svc.IsActive = *req.IsActive
	}

	if err := validateInput(svc.Name, svc.URL, svc.IntervalSeconds, svc.TimeoutSeconds, svc.FailuresBeforeAlert); err != nil {
		return nil, err
	}

	svc.UpdatedAt = time.Now()

	if _, err := c.db.NewUpdate().
		Model(svc).
		Where("id = ?", svc.ID).
		Exec(ctx); err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to update monitored service id=%d: %v", id, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	return svc, nil
}

func (c *Controller) Delete(ctx context.Context, id int64) error {
	_, err := c.db.NewDelete().
		Model((*model.MonitoredService)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to delete monitored service id=%d: %v", id, err)
		return errors.New("An unexpected error occurred. Please try again.")
	}

	if _, err := c.db.NewDelete().
		Model((*model.UptimeCheck)(nil)).
		Where("service_id = ?", id).
		Exec(ctx); err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to delete checks for monitored service id=%d: %v", id, err)
		return errors.New("An unexpected error occurred. Please try again.")
	}

	return nil
}

// CheckNow runs one check synchronously (same code path as a worker), records
// it, runs the state machine, and returns the updated service plus the result.
func (c *Controller) CheckNow(ctx context.Context, id int64) (*CheckNowResult, error) {
	svc, err := c.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	timeout := c.timeoutFor(svc)
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := runCheck(svc.URL, timeout)
	if err := c.applyResult(checkCtx, svc, result, true); err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to record manual check for service id=%d: %v", id, err)
		return nil, errors.New("An unexpected error occurred. Please try again.")
	}

	updated, err := c.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &CheckNowResult{Service: updated, Result: *result}, nil
}

func (c *Controller) timeoutFor(svc *model.MonitoredService) time.Duration {
	if svc.TimeoutSeconds >= 1 {
		return time.Duration(svc.TimeoutSeconds) * time.Second
	}
	return c.defaultTimeout
}

func validateInput(name, rawURL string, intervalSeconds, timeoutSeconds, failuresBeforeAlert int) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("Name is required.")
	}
	if strings.TrimSpace(rawURL) == "" {
		return errors.New("URL is required.")
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("URL must be a valid http(s) URL.")
	}
	if intervalSeconds < 10 {
		return errors.New("Check interval must be at least 10 seconds.")
	}
	if timeoutSeconds < 1 {
		return errors.New("Timeout must be at least 1 second.")
	}
	if timeoutSeconds >= intervalSeconds {
		return errors.New("Timeout must be less than the check interval.")
	}
	if failuresBeforeAlert < 1 {
		return errors.New("Failures before alert must be at least 1.")
	}
	return nil
}
