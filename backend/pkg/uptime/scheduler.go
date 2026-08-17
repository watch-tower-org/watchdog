package uptime

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

// Start launches the dispatcher goroutine and the check worker pool. Safe to
// call once.
func (c *Controller) Start() {
	c.wg.Add(1)
	go c.dispatcher()
	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go c.worker()
	}
	logger.Info().Msgf("uptime monitor started (%d workers, dispatch %s)", c.workers, c.dispatchInterval)
}

// Shutdown stops the dispatcher and workers, waiting for in-flight checks with
// a timeout.
func (c *Controller) Shutdown() {
	c.cancel()
	close(c.stop)
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(shutdownTimeout):
		logger.Warn().Msg("uptime monitor shutdown timed out")
	}
}

// dispatcher scans active services every dispatchInterval and pushes due
// services to the jobs channel.
func (c *Controller) dispatcher() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.dispatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.dispatch()
		}
	}
}

func (c *Controller) dispatch() {
	ctx, cancel := context.WithTimeout(c.ctx, c.dispatchInterval)
	defer cancel()

	var services []model.MonitoredService
	err := c.db.NewSelect().
		Model(&services).
		Where("is_active = true").
		Scan(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("uptime dispatch scan failed: %v", err)
		return
	}

	now := time.Now()
	for i := range services {
		s := &services[i]
		if !isDue(s.LastCheckedAt, s.IntervalSeconds, now) {
			continue
		}

		select {
		case c.jobs <- s.ID:
		default:
			logger.Warn().Msgf("uptime queue full, skipped check for service %d", s.ID)
		}
	}
}

// worker consumes service IDs and runs one check each.
func (c *Controller) worker() {
	defer c.wg.Done()
	for {
		select {
		case <-c.stop:
			return
		case id, ok := <-c.jobs:
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(c.ctx, c.defaultTimeout)
			if err := c.checkService(ctx, id); err != nil {
				logger.Ctx(ctx).Error().Msgf("uptime check failed for service %d: %v", id, err)
			}
			cancel()
		}
	}
}

// checkService loads a service, runs the HTTP check with its configured
// timeout, and records the result through the state machine.
func (c *Controller) checkService(ctx context.Context, serviceID int64) error {
	var svc model.MonitoredService
	if err := c.db.NewSelect().
		Model(&svc).
		Where("id = ?", serviceID).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if !svc.IsActive {
		return nil
	}

	timeout := c.timeoutFor(&svc)
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := runCheck(svc.URL, timeout)
	if result.Status == model.ServiceStatusDown {
		if result.Error != nil {
			logger.Ctx(ctx).Warn().Msgf("uptime service %q (%s) DOWN: %s", svc.Name, svc.URL, *result.Error)
		} else {
			logger.Ctx(ctx).Warn().Msgf("uptime service %q (%s) DOWN: HTTP %d in %dms", svc.Name, svc.URL, result.StatusCode, result.ResponseTimeMs)
		}
	}
	return c.applyResult(checkCtx, &svc, result)
}

// isDue reports whether a service is due for a check, given its last check and
// interval. A never-checked service is always due. Pure function (testable).
func isDue(lastChecked *time.Time, intervalSeconds int, now time.Time) bool {
	if lastChecked == nil || intervalSeconds < 1 {
		return true
	}
	return !now.Before(lastChecked.Add(time.Duration(intervalSeconds) * time.Second))
}
