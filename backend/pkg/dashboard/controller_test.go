package dashboard

import (
	"testing"
	"time"
)

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("bad test time %q: %v", value, err)
	}
	return parsed
}

func TestBuildSeriesFillsGaps(t *testing.T) {
	start := mustTime(t, "2026-08-03T00:00:00Z")
	now := mustTime(t, "2026-08-03T05:00:00Z")
	step := time.Hour

	events := map[time.Time]int64{
		mustTime(t, "2026-08-03T01:00:00Z"): 10,
		mustTime(t, "2026-08-03T04:00:00Z"): 2,
	}
	issues := map[time.Time]int64{
		mustTime(t, "2026-08-03T00:00:00Z"): 1,
		mustTime(t, "2026-08-03T03:00:00Z"): 5,
	}
	alerts := map[time.Time]int64{}

	buckets := buildSeries(start, step, now, events, issues, alerts)

	wantTimes := []string{
		"2026-08-03T00:00:00Z",
		"2026-08-03T01:00:00Z",
		"2026-08-03T02:00:00Z",
		"2026-08-03T03:00:00Z",
		"2026-08-03T04:00:00Z",
		"2026-08-03T05:00:00Z",
	}
	if len(buckets) != len(wantTimes) {
		t.Fatalf("expected %d buckets, got %d", len(wantTimes), len(buckets))
	}

	want := []struct {
		newIssues, events, alerts int64
	}{
		{1, 0, 0},
		{0, 10, 0},
		{0, 0, 0},
		{5, 0, 0},
		{0, 2, 0},
		{0, 0, 0},
	}

	for i, b := range buckets {
		if b.TS.Format(time.RFC3339) != wantTimes[i] {
			t.Fatalf("bucket %d: expected ts %s, got %s", i, wantTimes[i], b.TS.Format(time.RFC3339))
		}
		if b.NewIssues != want[i].newIssues || b.Events != want[i].events || b.Alerts != want[i].alerts {
			t.Fatalf("bucket %d: expected %+v, got %+v", i, want[i], b)
		}
	}
}

func TestBuildSeriesEmptyMaps(t *testing.T) {
	start := mustTime(t, "2026-08-01T00:00:00Z")
	now := mustTime(t, "2026-08-03T00:00:00Z")
	step := 24 * time.Hour

	buckets := buildSeries(start, step, now, nil, nil, nil)

	if len(buckets) != 3 {
		t.Fatalf("expected 3 daily buckets, got %d", len(buckets))
	}
	for _, b := range buckets {
		if b.NewIssues != 0 || b.Events != 0 || b.Alerts != 0 {
			t.Fatalf("expected zero-filled bucket, got %+v", b)
		}
	}
}

func TestBuildSeriesDoesNotExceedNow(t *testing.T) {
	start := mustTime(t, "2026-08-03T00:00:00Z")
	now := mustTime(t, "2026-08-03T02:30:00Z")
	step := time.Hour

	buckets := buildSeries(start, step, now, nil, nil, nil)

	if len(buckets) != 3 {
		t.Fatalf("expected 3 buckets (00, 01, 02), got %d", len(buckets))
	}
	last := buckets[len(buckets)-1]
	if last.TS.After(now) {
		t.Fatalf("last bucket %s must not exceed now %s", last.TS, now)
	}
}
