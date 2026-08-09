package ingestion

import (
	"strings"
	"testing"

	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

const sampleTrace = `panic: database connection pool exhausted

goroutine 12 [running]:
main.processOrder(0x1400012a000, 0x1400012a001)
	/workspace/payment/process.go:42 +0x8f
main.handleRequest(0x1400012a000)
	/workspace/payment/server.go:88 +0x12
net/http.(*Server).ServeHTTP(0x1400010a000, 0x1400012a000)
	/usr/local/go/src/net/http/server.go:2936 +0x2b`

func TestComputeFingerprint_StableAcrossRuns(t *testing.T) {
	a := ComputeFingerprint("payments-api", "panic", sampleTrace)
	b := ComputeFingerprint("payments-api", "panic", sampleTrace)
	if a != b {
		t.Fatalf("expected stable fingerprint, got %q and %q", a, b)
	}
	if a == "" || len(a) != 64 {
		t.Fatalf("expected 64-char hex fingerprint, got %q", a)
	}
}

func TestComputeFingerprint_DifferentProject(t *testing.T) {
	a := ComputeFingerprint("payments-api", "panic", sampleTrace)
	b := ComputeFingerprint("checkout-api", "panic", sampleTrace)
	if a == b {
		t.Fatalf("different projects should not collide: %q", a)
	}
}

func TestComputeFingerprint_DifferentErrorType(t *testing.T) {
	a := ComputeFingerprint("payments-api", "panic", sampleTrace)
	b := ComputeFingerprint("payments-api", "timeout", sampleTrace)
	if a == b {
		t.Fatalf("different error types should not collide: %q", a)
	}
}

func TestComputeFingerprint_AddressesNormalized(t *testing.T) {
	traceB := `panic: database connection pool exhausted

goroutine 7 [running]:
main.processOrder(0x7ffe1234abcd, 0x7ffe5678ef01)
	/workspace/payment/process.go:42 +0x1a
main.handleRequest(0x7ffe1234abcd)
	/workspace/payment/server.go:88 +0x77
net/http.(*Server).ServeHTTP(0x7ffe1234abcd, 0x7ffe5678ef01)
	/usr/local/go/src/net/http/server.go:2936 +0x1f`
	a := ComputeFingerprint("payments-api", "panic", sampleTrace)
	b := ComputeFingerprint("payments-api", "panic", traceB)
	if a != b {
		t.Fatalf("volatile addresses/offsets should normalize to same fingerprint:\n%q\n%q", a, b)
	}
}

func TestComputeFingerprint_DifferentStackFrames(t *testing.T) {
	traceB := `panic: database connection pool exhausted

goroutine 7 [running]:
main.otherFunction(0x1400012a000)
	/workspace/payment/other.go:10 +0x8f`
	a := ComputeFingerprint("payments-api", "panic", sampleTrace)
	b := ComputeFingerprint("payments-api", "panic", traceB)
	if a == b {
		t.Fatalf("different stack frames should not collide: %q", a)
	}
}

func TestExtractErrorType_FromMessage(t *testing.T) {
	if got := ExtractErrorType("database connection pool exhausted", ""); got != "database connection pool exhausted" {
		t.Fatalf("expected message-derived type, got %q", got)
	}
}

func TestExtractErrorType_FromStackTrace(t *testing.T) {
	got := ExtractErrorType("", sampleTrace)
	if got == "" {
		t.Fatal("expected a derived error type from stack trace")
	}
}

func TestExtractErrorType_Fallback(t *testing.T) {
	if got := ExtractErrorType("", ""); got != "error" {
		t.Fatalf("expected fallback 'error', got %q", got)
	}
}

func TestExtractTitle_PrefersMessage(t *testing.T) {
	req := &model.IngestEventRequest{
		Message:    "database connection pool exhausted",
		StackTrace: sampleTrace,
	}
	if got := ExtractTitle(req); got != "database connection pool exhausted" {
		t.Fatalf("expected message as title, got %q", got)
	}
}

func TestExtractTitle_Truncates(t *testing.T) {
	long := make([]rune, 300)
	for i := range long {
		long[i] = 'a'
	}
	req := &model.IngestEventRequest{Message: string(long)}
	if got := ExtractTitle(req); len(got) > 204 {
		t.Fatalf("expected truncated title, got length %d", len(got))
	}
}

func TestExtractFrames(t *testing.T) {
	frames := ExtractFrames(sampleTrace)
	if len(frames) == 0 {
		t.Fatal("expected frames extracted from stack trace")
	}
	for _, f := range frames {
		if strings.Contains(f, "0x") {
			t.Fatalf("frame not normalized: %q", f)
		}
	}
}
