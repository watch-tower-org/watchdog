package uptime

import (
	"strings"
	"testing"
	"time"

	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

func serviceStatus(t *testing.T, v string) model.ServiceStatus {
	t.Helper()
	s := model.ServiceStatus(v)
	if s != model.ServiceStatusUp && s != model.ServiceStatusDown {
		t.Fatalf("unexpected status %q", v)
	}
	return s
}

func TestComputeTransition_NoAlertBelowThreshold(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	upd := computeTransition(nil, 0, 3, nil, serviceStatus(t, "down"), now)

	if upd.consecutiveFailures != 1 {
		t.Fatalf("expected 1 consecutive failure, got %d", upd.consecutiveFailures)
	}
	if upd.alert != alertNone {
		t.Fatalf("expected no alert below threshold, got %q", upd.alert)
	}
	if upd.lastDownAt != nil {
		t.Fatalf("expected no last_down_at, got %v", upd.lastDownAt)
	}
}

func TestComputeTransition_DownAlertOnTransition(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	upd := computeTransition(ptr(model.ServiceStatusUp), 0, 1, nil, serviceStatus(t, "down"), now)

	if upd.consecutiveFailures != 1 {
		t.Fatalf("expected 1 consecutive failure, got %d", upd.consecutiveFailures)
	}
	if upd.alert != alertDown {
		t.Fatalf("expected down alert, got %q", upd.alert)
	}
	if upd.lastDownAt == nil || !upd.lastDownAt.Equal(now) {
		t.Fatalf("expected last_down_at set to now, got %v", upd.lastDownAt)
	}
}

func TestComputeTransition_AlertWhenFailuresReachThreshold(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	// Two failures already recorded, threshold is 3.
	upd := computeTransition(nil, 2, 3, nil, serviceStatus(t, "down"), now)

	if upd.consecutiveFailures != 3 {
		t.Fatalf("expected 3 consecutive failures, got %d", upd.consecutiveFailures)
	}
	if upd.alert != alertDown {
		t.Fatalf("expected down alert at threshold, got %q", upd.alert)
	}
}

func TestComputeTransition_NoRealertWhileDown(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	lastDown := now.Add(-30 * time.Minute)

	upd := computeTransition(ptr(model.ServiceStatusDown), 4, 1, &lastDown, serviceStatus(t, "down"), now)

	if upd.consecutiveFailures != 5 {
		t.Fatalf("expected 5 consecutive failures, got %d", upd.consecutiveFailures)
	}
	if upd.alert != alertNone {
		t.Fatalf("expected no re-alert while down, got %q", upd.alert)
	}
	if upd.lastDownAt != nil {
		t.Fatalf("expected last_down_at unchanged, got %v", upd.lastDownAt)
	}
}

func TestComputeTransition_RecoveryAlert(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	lastDown := now.Add(-30 * time.Minute)

	upd := computeTransition(ptr(model.ServiceStatusDown), 3, 1, &lastDown, serviceStatus(t, "up"), now)

	if upd.consecutiveFailures != 0 {
		t.Fatalf("expected failures reset to 0, got %d", upd.consecutiveFailures)
	}
	if upd.alert != alertUp {
		t.Fatalf("expected recovery alert, got %q", upd.alert)
	}
	if upd.lastUpAt == nil || !upd.lastUpAt.Equal(now) {
		t.Fatalf("expected last_up_at set to now, got %v", upd.lastUpAt)
	}
	if upd.downSince == nil || !upd.downSince.Equal(lastDown) {
		t.Fatalf("expected downSince to carry the previous down time, got %v", upd.downSince)
	}
}

func TestComputeTransition_NoRecoveryWithoutAlertedDown(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	// Down with nil last_down_at means DOWN was never alerted; going up
	// should not send a recovery email for a silent failure.
	upd := computeTransition(ptr(model.ServiceStatusDown), 1, 5, nil, serviceStatus(t, "up"), now)

	if upd.consecutiveFailures != 0 {
		t.Fatalf("expected failures reset to 0, got %d", upd.consecutiveFailures)
	}
	if upd.alert != alertNone {
		t.Fatalf("expected no recovery email, got %q", upd.alert)
	}
	if upd.lastUpAt != nil {
		t.Fatalf("expected no last_up_at, got %v", upd.lastUpAt)
	}
}

func TestComputeTransition_NewServiceDownFirstCheck(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	upd := computeTransition(nil, 0, 1, nil, serviceStatus(t, "down"), now)

	if upd.alert != alertDown {
		t.Fatalf("expected down alert on first check, got %q", upd.alert)
	}
}

func TestBuildEmailBody_Down(t *testing.T) {
	svc := &model.MonitoredService{Name: "payments", URL: "https://payments.example", IntervalSeconds: 60}
	errMsg := "dial tcp: connection refused"
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	alert := &alertEmail{
		kind:      "down",
		service:   svc,
		result:    &CheckResult{Status: model.ServiceStatusDown, Error: &errMsg},
		downSince: ptr(now),
		failures:  2,
	}

	subject, body := buildEmailBody(alert)

	if subject != "[WatchTower] DOWN: payments" {
		t.Fatalf("unexpected subject %q", subject)
	}
	for _, want := range []string{"service is DOWN", "https://payments.example", "DOWN", "Failures:   2", "dial tcp: connection refused", "Interval:   60s"} {
		if !strings.Contains(body, want) {
			t.Fatalf("email body missing %q:\n%s", want, body)
		}
	}
}

func TestBuildEmailBody_Up(t *testing.T) {
	downSince := time.Date(2026, 8, 15, 11, 0, 0, 0, time.UTC)
	checkedAt := downSince.Add(time.Hour)
	svc := &model.MonitoredService{
		Name:            "payments",
		URL:             "https://payments.example",
		IntervalSeconds: 30,
		LastCheckedAt:   ptr(checkedAt),
	}
	alert := &alertEmail{
		kind:      "up",
		service:   svc,
		result:    &CheckResult{Status: model.ServiceStatusUp, StatusCode: 200},
		downSince: &downSince,
	}

	subject, body := buildEmailBody(alert)

	if subject != "[WatchTower] UP: payments" {
		t.Fatalf("unexpected subject %q", subject)
	}
	for _, want := range []string{"UP (recovered)", "https://payments.example", "1h0m0s", downSince.Format(time.RFC3339)} {
		if !strings.Contains(body, want) {
			t.Fatalf("email body missing %q:\n%s", want, body)
		}
	}
}
