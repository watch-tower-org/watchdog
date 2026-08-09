package issues

import "testing"

func TestDedupeIDs(t *testing.T) {
	in := []int64{1, 2, 2, 3, 1, 4}
	got := dedupeIDs(in)
	want := []int64{1, 2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("expected %d ids, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestTruncateTitle(t *testing.T) {
	short := "short title"
	if got := truncateTitle(short); got != short {
		t.Fatalf("short title should be unchanged, got %q", got)
	}
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	got := truncateTitle(string(long))
	if len(got) != 203 {
		t.Fatalf("expected truncated title of length 203, got %d", len(got))
	}
	if got[len(got)-3:] != "..." {
		t.Fatalf("expected ellipsis suffix, got %q", got[len(got)-3:])
	}
}

func TestRandomFingerprint(t *testing.T) {
	fp, err := randomFingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if len(fp) != 64 {
		t.Fatalf("expected 64-char hex fingerprint, got %q", fp)
	}
	other, _ := randomFingerprint()
	if fp == other {
		t.Fatal("random fingerprints must differ")
	}
}
