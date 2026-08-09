package notifier

import (
	"testing"
	"time"

	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

func TestMatchesRule(t *testing.T) {
	rule := &model.AlertRule{Project: "payments-api", Tag: "payment"}

	cases := []struct {
		name    string
		project string
		tag     string
		want    bool
	}{
		{"exact match", "payments-api", "payment", true},
		{"different project", "checkout-api", "payment", false},
		{"different tag", "payments-api", "http", false},
		{"wildcard project", "checkout-api", "payment", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchesRule(rule, tc.project, tc.tag); got != tc.want {
				t.Fatalf("matchesRule = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMatchesRule_Wildcards(t *testing.T) {
	wild := &model.AlertRule{Project: "", Tag: ""}
	if !matchesRule(wild, "anything", "anytag") {
		t.Fatal("empty matchers should match everything")
	}
	tagOnly := &model.AlertRule{Tag: "email"}
	if !matchesRule(tagOnly, "whatever", "email") {
		t.Fatal("project wildcard should allow any project")
	}
	if matchesRule(tagOnly, "whatever", "queue") {
		t.Fatal("tag should still filter")
	}
}

func TestTriggerFired(t *testing.T) {
	cases := []struct {
		name          string
		trigger       model.AlertTriggerType
		isNew         bool
		regression    bool
		windowedCount int
		threshold     int
		want          bool
	}{
		{"new issue fires", model.AlertTriggerNewIssue, true, false, 0, 0, true},
		{"existing issue no fire", model.AlertTriggerNewIssue, false, false, 0, 0, false},
		{"regression fires", model.AlertTriggerRegression, false, true, 0, 0, true},
		{"not regression no fire", model.AlertTriggerRegression, false, false, 0, 0, false},
		{"spike at threshold", model.AlertTriggerSpike, false, false, 10, 10, true},
		{"spike below threshold", model.AlertTriggerSpike, false, false, 9, 10, false},
		{"spike zero threshold", model.AlertTriggerSpike, false, false, 100, 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := triggerFired(tc.trigger, tc.isNew, tc.regression, tc.windowedCount, tc.threshold)
			if got != tc.want {
				t.Fatalf("triggerFired = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsThrottled(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-5 * time.Minute)
	old := now.Add(-2 * time.Hour)

	if !isThrottled(&recent, 60, now) {
		t.Fatal("recent alert within window should be throttled")
	}
	if isThrottled(&old, 60, now) {
		t.Fatal("old alert outside window should not be throttled")
	}
	if isThrottled(nil, 60, now) {
		t.Fatal("never-sent should not be throttled")
	}
	if isThrottled(&recent, 0, now) {
		t.Fatal("zero throttle window should not throttle")
	}
}
