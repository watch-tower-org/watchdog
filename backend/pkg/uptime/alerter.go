package uptime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchtower/backend/internal/logger"
	"github.com/watch-tower-org/watchtower/backend/internal/mailer"
	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

type alertEmail struct {
	kind      string // "down" | "up"
	service   *model.MonitoredService
	result    *CheckResult
	downSince *time.Time
	failures  int
}

// applyResult writes the check history row and runs the state machine on the
// service row. State transitions are serialized per service via a row lock
// (SELECT ... FOR UPDATE) so concurrent workers cannot double-alert.
// When force is true (manual "Check now"), results are recorded even for
// inactive services.
func (c *Controller) applyResult(ctx context.Context, svc *model.MonitoredService, result *CheckResult, force ...bool) error {
	allowInactive := len(force) > 0 && force[0]
	now := time.Now()

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var current model.MonitoredService
	err = tx.NewSelect().
		Model(&current).
		Where("id = ?", svc.ID).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tx.Commit()
		}
		return fmt.Errorf("lock service: %w", err)
	}
	if !current.IsActive && !allowInactive {
		return tx.Commit()
	}

	// Skip a check that raced with a newer one already recorded.
	if svc.LastCheckedAt != nil && current.LastCheckedAt != nil && current.LastCheckedAt.After(*svc.LastCheckedAt) {
		return tx.Commit()
	}

	check := &model.UptimeCheck{
		ServiceID:      current.ID,
		CheckedAt:      now,
		Status:         result.Status,
		StatusCode:     statusCodeArg(result.StatusCode),
		ResponseTimeMs: responseTimeArg(result.ResponseTimeMs),
		Error:          result.Error,
	}
	if _, err := tx.NewInsert().Model(check).Exec(ctx); err != nil {
		return fmt.Errorf("insert uptime check: %w", err)
	}

	var alert *alertEmail
	prevStatus := current.LastStatus
	upd := computeTransition(prevStatus, current.ConsecutiveFailures, current.FailuresBeforeAlert, current.LastDownAt, result.Status, now)

	current.LastCheckedAt = &now
	current.LastStatus = &result.Status
	current.UpdatedAt = now
	current.ConsecutiveFailures = upd.consecutiveFailures
	if upd.lastUpAt != nil {
		current.LastUpAt = upd.lastUpAt
	}
	if upd.lastDownAt != nil {
		current.LastDownAt = upd.lastDownAt
	}
	switch upd.alert {
	case alertDown:
		alert = &alertEmail{
			kind:      "down",
			service:   &current,
			result:    result,
			downSince: upd.downSince,
			failures:  upd.consecutiveFailures,
		}
	case alertUp:
		alert = &alertEmail{
			kind:      "up",
			service:   &current,
			result:    result,
			downSince: upd.downSince,
		}
	}

	if _, err := tx.NewUpdate().
		Model(&current).
		Where("id = ?", current.ID).
		Exec(ctx); err != nil {
		return fmt.Errorf("update service: %w", err)
	}

	if err := c.pruneChecks(ctx, tx, current.ID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit state transition: %w", err)
	}

	if alert != nil {
		c.sendAlert(ctx, alert)
	}

	return nil
}

// pruneChecks keeps only the latest checkHistoryLimit rows for a service,
// deleting older history after each insert.
func (c *Controller) pruneChecks(ctx context.Context, tx bun.Tx, serviceID int64) error {
	if _, err := tx.NewRaw(
		`DELETE FROM uptime_checks WHERE service_id = ? AND id NOT IN (SELECT id FROM uptime_checks WHERE service_id = ? ORDER BY id DESC LIMIT ?)`,
		serviceID, serviceID, checkHistoryLimit,
	).Exec(ctx); err != nil {
		return fmt.Errorf("prune uptime checks: %w", err)
	}
	return nil
}

// isDown reports whether the stored status is "down".
func isDown(status *model.ServiceStatus) bool {
	return status != nil && *status == model.ServiceStatusDown
}

type alertKind string

const (
	alertNone alertKind = ""
	alertDown alertKind = "down"
	alertUp   alertKind = "up"
)

// stateUpdate is the result of computing a state transition for one check.
type stateUpdate struct {
	consecutiveFailures int
	lastUpAt            *time.Time
	lastDownAt          *time.Time
	alert               alertKind
	downSince           *time.Time // last_down_at to reference in the email
}

// computeTransition is the pure state machine: given the persisted service
// state before a check and the new check result, decide the new
// consecutive_failures, last_up_at/last_down_at, and whether an alert
// (DOWN/recovery) fires. Alerts are edge-triggered on transitions — no
// re-alert while already down.
func computeTransition(prevStatus *model.ServiceStatus, consecutiveFailures, failuresBeforeAlert int, lastDownAt *time.Time, result model.ServiceStatus, now time.Time) stateUpdate {
	upd := stateUpdate{}

	if result == model.ServiceStatusUp {
		upd.consecutiveFailures = 0
		// Recovery: email only if the service was previously alerted down.
		if isDown(prevStatus) && lastDownAt != nil {
			upd.lastUpAt = &now
			upd.alert = alertUp
			upd.downSince = lastDownAt
		}
		return upd
	}

	// Down.
	upd.consecutiveFailures = consecutiveFailures + 1
	if upd.consecutiveFailures >= failuresBeforeAlert && !isDown(prevStatus) {
		upd.lastDownAt = &now
		upd.downSince = &now
		upd.alert = alertDown
	}
	return upd
}

func statusCodeArg(code int) *int {
	if code >= 200 {
		return &code
	}
	return nil
}

func responseTimeArg(ms int) *int {
	if ms >= 0 {
		return &ms
	}
	return nil
}

// sendAlert emails the DOWN/UP notification to the service's recipient list.
// Missing SMTP config or recipient list simply skips the email (checks are
// still recorded); failures are logged, never fatal.
func (c *Controller) sendAlert(ctx context.Context, a *alertEmail) {
	email, err := c.loadEmailSettings(ctx)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("uptime email skipped: %v", err)
		return
	}
	if email.SMTPHost == "" || email.SMTPFromEmail == "" {
		logger.Ctx(ctx).Info().Msg("uptime email skipped: smtp settings not configured")
		return
	}

	if a.service.RecipientListID == nil {
		logger.Ctx(ctx).Info().Msgf("uptime email skipped: service %d has no recipient list", a.service.ID)
		return
	}

	list, err := c.loadRecipientList(ctx, *a.service.RecipientListID)
	if err != nil {
		logger.Ctx(ctx).Error().Msgf("uptime email failed: %v", err)
		return
	}
	if len(list.Emails) == 0 {
		logger.Ctx(ctx).Info().Msgf("uptime email skipped: recipient list %d has no emails", list.ID)
		return
	}

	subject, body := buildEmailBody(a)
	for _, to := range list.Emails {
		if err := mailer.Send(
			email.SMTPHost,
			email.SMTPPort,
			email.SMTPUsername,
			email.SMTPPassword,
			email.SMTPFromName,
			email.SMTPFromEmail,
			to,
			subject,
			body,
		); err != nil {
			logger.Ctx(ctx).Error().Msgf("smtp send to %s failed: %v", to, err)
		}
	}
	logger.Ctx(ctx).Info().Msgf("uptime %s email sent for service %d to %d recipients", a.kind, a.service.ID, len(list.Emails))
}

func (c *Controller) loadEmailSettings(ctx context.Context) (*model.EmailSettings, error) {
	var email model.EmailSettings
	err := c.db.NewSelect().
		Model(&email).
		Order("id ASC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("load email settings: %w", err)
	}
	return &email, nil
}

func (c *Controller) loadRecipientList(ctx context.Context, id int64) (*model.RecipientList, error) {
	var list model.RecipientList
	err := c.db.NewSelect().
		Model(&list).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("load recipient list: %w", err)
	}
	return &list, nil
}

func buildEmailBody(a *alertEmail) (string, string) {
	svc := a.service
	checkedAt := time.Now()
	if a.result != nil && svc.LastCheckedAt != nil {
		checkedAt = *svc.LastCheckedAt
	}
	codeOrErr := ""
	if a.result != nil {
		if a.result.Error != nil {
			codeOrErr = *a.result.Error
		} else if a.result.StatusCode > 0 {
			codeOrErr = fmt.Sprintf("%d", a.result.StatusCode)
		}
	}

	switch a.kind {
	case "down":
		subject := fmt.Sprintf("[WatchTower] DOWN: %s", svc.Name)
		body := fmt.Sprintf(
			"WatchTower alert: service is DOWN\n\n"+
				"Service:    %s\n"+
				"URL:        %s\n"+
				"Status:     DOWN\n"+
				"Checked at: %s\n"+
				"Code/error: %s\n"+
				"Failures:   %d\n"+
				"Down since: %s\n"+
				"Interval:   %ds\n",
			svc.Name,
			svc.URL,
			checkedAt.Format(time.RFC3339),
			codeOrErr,
			a.failures,
			a.downSince.Format(time.RFC3339),
			svc.IntervalSeconds,
		)
		return subject, body

	case "up":
		downtime := "unknown"
		if a.downSince != nil {
			downtime = checkedAt.Sub(*a.downSince).Round(time.Second).String()
		}
		subject := fmt.Sprintf("[WatchTower] UP: %s", svc.Name)
		body := fmt.Sprintf(
			"Service:    %s\n"+
				"URL:        %s\n"+
				"Status:     UP (recovered)\n"+
				"Checked at: %s\n"+
				"Down since: %s\n"+
				"Downtime:   %s\n",
			svc.Name,
			svc.URL,
			checkedAt.Format(time.RFC3339),
			a.downSince.Format(time.RFC3339),
			downtime,
		)
		return subject, body
	}

	return "", ""
}
