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

func TestExtractErrorType_FromMessageColon(t *testing.T) {
	if got := ExtractErrorType("context deadline exceeded: read tcp", ""); got != "context deadline exceeded" {
		t.Fatalf("expected colon-prefixed type, got %q", got)
	}
}

func TestExtractErrorType_ColonlessMessageNotUsedAsType(t *testing.T) {
	if got := ExtractErrorType("user 123 not found", ""); got != "error" {
		t.Fatalf("expected colon-less message to fall back to 'error', got %q", got)
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

func TestResolveErrorType_PrefersProvided(t *testing.T) {
	req := &model.IngestEventRequest{
		Message:   "user 123 not found",
		ErrorType: "notfound.UserError",
	}
	if got := ResolveErrorType(req); got != "notfound.UserError" {
		t.Fatalf("expected provided ErrorType to win, got %q", got)
	}
}

func TestResolveErrorType_TrimsWhitespace(t *testing.T) {
	req := &model.IngestEventRequest{
		Message:   "user 123 not found",
		ErrorType: "  ",
	}
	if got := ResolveErrorType(req); got != "error" {
		t.Fatalf("expected blank ErrorType to fall back, got %q", got)
	}
}

func TestResolveErrorType_FallsBackToMessage(t *testing.T) {
	req := &model.IngestEventRequest{Message: "connection refused: dial tcp"}
	if got := ResolveErrorType(req); got != "connection refused" {
		t.Fatalf("expected colon-prefixed message type, got %q", got)
	}
}

func TestComputeFingerprint_SameTypeDifferentMessages(t *testing.T) {
	a := ComputeFingerprint("payments-api", "notfound.UserError", sampleTrace)
	b := ComputeFingerprint("payments-api", "notfound.UserError", sampleTrace)
	if a != b {
		t.Fatalf("same type + frames must collide: %q", a)
	}
	if got := ResolveErrorType(&model.IngestEventRequest{Message: "user 456 not found", ErrorType: "notfound.UserError"}); got != "notfound.UserError" {
		t.Fatalf("expected provided type to win, got %q", got)
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

func TestParseFingerprintMode(t *testing.T) {
	cases := map[string]FingerprintMode{
		"type+frames": ModeTypeAndFrames,
		"type":        ModeTypeOnly,
		"message":     ModeMessage,
		"TYPE":        ModeTypeOnly,
		"Message":     ModeMessage,
		"":            ModeTypeAndFrames,
		"bogus":       ModeTypeAndFrames,
	}
	for in, want := range cases {
		if got := ParseFingerprintMode(in); got != want {
			t.Fatalf("ParseFingerprintMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestComputeFingerprintMode_TypeOnlyGroupsAcrossSites(t *testing.T) {
	traceB := "main.otherFunction(0x1400012a000)\n\t/workspace/other.go:10 +0x8f"
	a := ComputeFingerprintMode("payments-api", "notfound.UserError", sampleTrace, "", ModeTypeOnly)
	b := ComputeFingerprintMode("payments-api", "notfound.UserError", traceB, "", ModeTypeOnly)
	if a != b {
		t.Fatalf("type mode must ignore stack frames: %q != %q", a, b)
	}
	if a == ComputeFingerprintMode("payments-api", "db.ConnError", sampleTrace, "", ModeTypeOnly) {
		t.Fatal("type mode must still distinguish error types")
	}
}

func TestComputeFingerprintMode_MessageModeSplitsOnMessage(t *testing.T) {
	a := ComputeFingerprintMode("payments-api", "notfound.UserError", sampleTrace, "user 123 not found", ModeMessage)
	b := ComputeFingerprintMode("payments-api", "notfound.UserError", sampleTrace, "user 456 not found", ModeMessage)
	if a == b {
		t.Fatal("message mode must split on different messages")
	}
	if a != ComputeFingerprintMode("payments-api", "notfound.UserError", sampleTrace, "user 123 not found", ModeMessage) {
		t.Fatal("message mode must group identical messages")
	}
}

func TestComputeFingerprintMode_DefaultIsTypeAndFrames(t *testing.T) {
	if got := ComputeFingerprint("payments-api", "panic", sampleTrace); got != ComputeFingerprintMode("payments-api", "panic", sampleTrace, "", "") {
		t.Fatalf("ComputeFingerprint must match default mode")
	}
}

func TestParseFingerprintMode_Heuristic(t *testing.T) {
	cases := map[string]FingerprintMode{
		"heuristic": ModeHeuristic,
		"Heuristic": ModeHeuristic,
		" heuristic ": ModeHeuristic,
	}
	for in, want := range cases {
		if got := ParseFingerprintMode(in); got != want {
			t.Fatalf("ParseFingerprintMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractFrameFunction(t *testing.T) {
	cases := map[string]string{
		"main.processOrder(0x1400012a000, 0x1400012a001)": "main.processOrder",
		"main.handleRequest()":                            "main.handleRequest",
		"net/http.(*Server).ServeHTTP(0x1400010a000)":     "net/http.(*Server).ServeHTTP",
		"/workspace/payment/process.go:42 +0x8f":          "",
		"goroutine 12 [running]:":                         "",
		"":                                                "",
	}
	for in, want := range cases {
		if got := ExtractFrameFunction(in); got != want {
			t.Fatalf("ExtractFrameFunction(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestComputeFingerprintMode_HeuristicGroupsVolatileMessages(t *testing.T) {
	a := ComputeFingerprintMode("payments-api", "notfound.UserError", sampleTrace, "user 123 not found", ModeHeuristic)
	b := ComputeFingerprintMode("payments-api", "notfound.UserError", sampleTrace, "user 456 not found", ModeHeuristic)
	if a != b {
		t.Fatalf("heuristic mode must mask volatile ids in messages: %q != %q", a, b)
	}
	if a != ComputeFingerprintMode("payments-api", "notfound.UserError", sampleTrace, "user 123 not found", ModeHeuristic) {
		t.Fatal("heuristic mode must be stable for identical input")
	}
}

func TestComputeFingerprintMode_HeuristicSplitsOnFunction(t *testing.T) {
	traceB := `panic: database connection pool exhausted

goroutine 12 [running]:
main.otherFunction(0x1400012a000)
	/workspace/payment/other.go:10 +0x8f`
	a := ComputeFingerprintMode("payments-api", "notfound.UserError", sampleTrace, "user 123 not found", ModeHeuristic)
	b := ComputeFingerprintMode("payments-api", "notfound.UserError", traceB, "user 123 not found", ModeHeuristic)
	if a == b {
		t.Fatalf("heuristic mode must split on different top frame functions: %q", a)
	}
}

func TestComputeFingerprintMode_HeuristicSplitsOnErrorType(t *testing.T) {
	a := ComputeFingerprintMode("payments-api", "notfound.UserError", sampleTrace, "user 123 not found", ModeHeuristic)
	b := ComputeFingerprintMode("payments-api", "db.ConnError", sampleTrace, "user 123 not found", ModeHeuristic)
	if a == b {
		t.Fatalf("heuristic mode must split on different error types: %q", a)
	}
}

func TestNormalizeMessageForGrouping(t *testing.T) {
	in := "order 12345 failed for user abc@example.com at 2026-01-02T15:04:05Z from 10.0.0.1 id 6f9619ff-8b86-d011-b42d-00cf4fc964ff ptr 0xc0000b1f20 cost 1.5ms"
	want := "order * failed for user * at * from * id * ptr * cost *ms"
	if got := normalizeMessageForGrouping(in); got != want {
		t.Fatalf("normalizeMessageForGrouping(%q) = %q, want %q", in, got, want)
	}
}
