package notifier

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/internal/mailer"
	"github.com/watch-tower-org/watchdog/backend/internal/model"
)

// matchesRule reports whether a rule's project/tag matcher applies to an event.
// An empty matcher field acts as a wildcard.
func matchesRule(rule *model.AlertRule, project, tag string) bool {
	if rule.Project != "" && rule.Project != project {
		return false
	}
	if rule.Tag != "" && rule.Tag != tag {
		return false
	}
	return true
}

// triggerFired reports whether a rule's trigger fired for the given event.
// spike uses the windowed occurrence count passed in.
func triggerFired(trigger model.AlertTriggerType, isNewIssue, wasRegression bool, windowedCount, threshold int) bool {
	switch trigger {
	case model.AlertTriggerNewIssue:
		return isNewIssue
	case model.AlertTriggerRegression:
		return wasRegression
	case model.AlertTriggerSpike:
		return threshold > 0 && windowedCount >= threshold
	default:
		return false
	}
}

// isThrottled reports whether an alert was already sent for this rule+issue
// within the throttle window. lastSent == nil means never sent.
func isThrottled(lastSent *time.Time, throttleMinutes int, now time.Time) bool {
	if lastSent == nil || throttleMinutes < 1 {
		return false
	}
	return now.Before(lastSent.Add(time.Duration(throttleMinutes) * time.Minute))
}

func (n *Notifier) evaluate(ctx context.Context, j job) error {
	var issue model.Issue
	if err := n.db.NewSelect().Model(&issue).Where("id = ?", j.IssueID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("load issue: %w", err)
	}

	var rules []model.AlertRule
	err := n.db.NewSelect().
		Model(&rules).
		Where("is_active = true").
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("load rules: %w", err)
	}

	alertSettings, err := n.loadAlertSettings(ctx)
	if err != nil {
		return err
	}

	for i := range rules {
		rule := &rules[i]
		if !matchesRule(rule, j.Project, j.Tag) {
			continue
		}

		var windowedCount int
		if rule.TriggerType == model.AlertTriggerSpike && rule.Threshold > 0 && rule.WindowMinutes > 0 {
			since := time.Now().Add(-time.Duration(rule.WindowMinutes) * time.Minute)
			windowedCount, err = n.db.NewSelect().
				Model((*model.Event)(nil)).
				Where("issue_id = ?", issue.ID).
				Where("timestamp > ?", since).
				Count(ctx)
			if err != nil {
				logger.Ctx(ctx).Error().Msgf("windowed count failed for issue %d: %v", issue.ID, err)
				continue
			}
		}

		if !triggerFired(rule.TriggerType, j.IsNewIssue, j.WasRegression, windowedCount, rule.Threshold) {
			continue
		}

		throttleMinutes := rule.ThrottleWindow
		if throttleMinutes < 1 {
			throttleMinutes = alertSettings.ThrottleWindow
		}

		lastSent, err := n.lastSentAt(ctx, issue.ID, rule.ID)
		if err != nil {
			logger.Ctx(ctx).Error().Msgf("lastSent lookup failed for issue %d rule %d: %v", issue.ID, rule.ID, err)
			continue
		}
		if isThrottled(lastSent, throttleMinutes, time.Now()) {
			continue
		}

		if err := n.send(ctx, &issue, rule, throttleMinutes); err != nil {
			logger.Ctx(ctx).Error().Msgf("alert send failed for issue %d rule %d: %v", issue.ID, rule.ID, err)
			continue
		}
	}

	return nil
}

func (n *Notifier) loadAlertSettings(ctx context.Context) (*model.AlertSettings, error) {
	var s model.AlertSettings
	err := n.db.NewSelect().
		Model(&s).
		Limit(1).
		OrderBy("id", bun.OrderAsc).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("load alert settings: %w", err)
	}
	return &s, nil
}

func (n *Notifier) lastSentAt(ctx context.Context, issueID, ruleID int64) (*time.Time, error) {
	var log model.AlertLog
	err := n.db.NewSelect().
		Model(&log).
		Where("issue_id = ?", issueID).
		Where("rule_id = ?", ruleID).
		Order("sent_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &log.SentAt, nil
}

func (n *Notifier) send(ctx context.Context, issue *model.Issue, rule *model.AlertRule, throttleMinutes int) error {
	var email model.EmailSettings
	err := n.db.NewSelect().
		Model(&email).
		Limit(1).
		OrderBy("id", bun.OrderAsc).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("load email settings: %w", err)
	}
	if email.SMTPHost == "" || email.SMTPFromEmail == "" {
		return fmt.Errorf("smtp settings not configured")
	}

	var list model.RecipientList
	if err := n.db.NewSelect().
		Model(&list).
		Where("id = ?", rule.RecipientListID).
		Scan(ctx); err != nil {
		return fmt.Errorf("load recipient list: %w", err)
	}
	if len(list.Emails) == 0 {
		return fmt.Errorf("recipient list %d has no emails", list.ID)
	}

	subject := fmt.Sprintf("[WatchTower] %s: %s", rule.TriggerType, issue.Title)
	body := fmt.Sprintf(
		"WatchTower alert\n\nTrigger: %s\nIssue: %s\nProject: %s\nTag: %s\nOccurrences: %d\nFirst seen: %s\nLast seen: %s\n\nThrottle window: %d min\n\nView issue: /issues/%d",
		rule.TriggerType,
		issue.Title,
		issue.Project,
		issue.Tag,
		issue.Count,
		issue.FirstSeen.Format(time.RFC3339),
		issue.LastSeen.Format(time.RFC3339),
		throttleMinutes,
		issue.ID,
	)

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
			return err
		}
	}

	entry := &model.AlertLog{
		IssueID:    issue.ID,
		RuleID:     rule.ID,
		SentAt:     time.Now(),
		Recipients: list.Emails,
	}
	if _, err := n.db.NewInsert().Model(entry).Exec(ctx); err != nil {
		return fmt.Errorf("insert alert log: %w", err)
	}

	logger.Ctx(ctx).Info().Msgf("alert sent for issue %d via rule %d to %d recipients", issue.ID, rule.ID, len(list.Emails))
	return nil
}
