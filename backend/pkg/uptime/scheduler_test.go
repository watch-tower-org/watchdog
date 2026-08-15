package uptime

import (
	"testing"
	"time"
)

func TestIsDue(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name            string
		lastChecked     *time.Time
		intervalSeconds int
		want            bool
	}{
		{"never checked", nil, 60, true},
		{"zero interval", &now, 0, true},
		{"failed interval", &now, -5, true},
		{"not yet due", ptr(now.Add(10 * time.Second)), 60, false},
		{"exactly due", ptr(now.Add(-60 * time.Second)), 60, true},
		{"past due", ptr(now.Add(-61 * time.Second)), 60, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isDue(tc.lastChecked, tc.intervalSeconds, now); got != tc.want {
				t.Fatalf("isDue = %v, want %v", got, tc.want)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
