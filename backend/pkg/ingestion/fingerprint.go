package ingestion

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/watch-tower-org/watchtower/backend/internal/model"
)

const maxFrames = 10

// FingerprintMode selects what inputs feed the dedup key.
type FingerprintMode string

const (
	// ModeTypeAndFrames (default) hashes (project, error type, top N normalized
	// stack frames). Identical code paths group; distinct call sites do not.
	ModeTypeAndFrames FingerprintMode = "type+frames"
	// ModeTypeOnly hashes (project, error type) — every event of the same type
	// in a project lands in one issue regardless of where it was raised.
	ModeTypeOnly FingerprintMode = "type"
	// ModeMessage hashes (project, error type, normalized message) — the
	// strictest grouping; only byte-identical messages group together.
	ModeMessage FingerprintMode = "message"
	// ModeHeuristic hashes (project, error type, top frame function, normalized
	// message). Volatile values (ids, timestamps, hex pointers, etc.) in the
	// message are masked, so "user 123 not found" and "user 456 not found"
	// raised from the same code path group together.
	ModeHeuristic FingerprintMode = "heuristic"
)

// ParseFingerprintMode normalizes a config string into a FingerprintMode,
// defaulting to ModeTypeAndFrames for unknown or empty input.
func ParseFingerprintMode(s string) FingerprintMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "type":
		return ModeTypeOnly
	case "message":
		return ModeMessage
	case "heuristic":
		return ModeHeuristic
	default:
		return ModeTypeAndFrames
	}
}

var (
	addrRe  = regexp.MustCompile(`0x[0-9a-f]+`)
	posRe   = regexp.MustCompile(`\+0x[0-9a-f]+$`)
	argsRe  = regexp.MustCompile(`\(0x[0-9a-f]+(?:, 0x[0-9a-f]+)*\)`)
	fileRe  = regexp.MustCompile(`\.go:\d+`)
	spaceRe = regexp.MustCompile(`\s+`)

	tsRe   = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}(:\d{2})?(\.\d+)?(Z|[+-]\d{2}:?\d{2})?`)
	emailRe = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	urlRe  = regexp.MustCompile(`https?://[^\s]+`)
	uuidRe = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	ipRe   = regexp.MustCompile(`\b\d{1,3}(\.\d{1,3}){3}\b`)
	numRe  = regexp.MustCompile(`-?\d[\d.,]*(e-?\d+)?`)
)

// ExtractErrorType returns the error type for a report. If the caller provided
// one it wins; otherwise it is derived from the stack trace (first panic/error
// marker) or falls back to "error". A message is only used as a type when it
// follows the "type: detail" convention — a colon-less message (e.g.
// "user 123 not found") is variable content, not a stable type, and would
// otherwise split identical code paths into different issues.
func ExtractErrorType(msg, stackTrace string) string {
	if msg != "" {
		if i := strings.Index(msg, ":"); i >= 0 {
			if t := strings.TrimSpace(msg[:i]); t != "" {
				return t
			}
		}
	}
	if stackTrace != "" {
		first := stackTrace
		if i := strings.Index(stackTrace, "\n"); i >= 0 {
			first = stackTrace[:i]
		}
		first = strings.TrimSpace(first)
		if first != "" {
			return first
		}
	}
	return "error"
}

// ResolveErrorType returns the error type used for fingerprinting. A
// caller-provided ErrorType (e.g. the SDK's Go type) wins; otherwise it is
// derived from the message or stack trace.
func ResolveErrorType(req *model.IngestEventRequest) string {
	if t := strings.TrimSpace(req.ErrorType); t != "" {
		return t
	}
	return ExtractErrorType(req.Message, req.StackTrace)
}

// ExtractTitle returns the human-readable title used for an issue. The full
// message is preferred; otherwise the error type.
func ExtractTitle(req *model.IngestEventRequest) string {
	if strings.TrimSpace(req.Message) != "" {
		return truncate(strings.TrimSpace(req.Message), 200)
	}
	t := ExtractErrorType(req.Message, req.StackTrace)
	return truncate(t, 200)
}

// NormalizeFrame strips volatile detail (addresses, offsets, arguments) from a
// single stack frame so functionally identical stacks hash the same.
func NormalizeFrame(frame string) string {
	f := strings.TrimSpace(frame)
	f = posRe.ReplaceAllString(f, "")
	f = addrRe.ReplaceAllString(f, "0x0")
	f = argsRe.ReplaceAllString(f, "(args)")
	f = fileRe.ReplaceAllString(f, ".go")
	f = spaceRe.ReplaceAllString(f, " ")
	return strings.TrimSpace(f)
}

// ExtractFrames splits a stack trace into a normalized slice of frames.
func ExtractFrames(stackTrace string) []string {
	lines := strings.Split(stackTrace, "\n")
	frames := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "goroutine") {
			continue
		}
		f := NormalizeFrame(line)
		if f != "" {
			frames = append(frames, f)
		}
	}
	return frames
}

// ExtractFrameFunction returns the package-qualified function name of a stack
// frame line, e.g. "main.processOrder" from
// "main.processOrder(0x1400012a000, 0x1400012a001)". Lines that are not
// function frames (bare file paths, panic messages, goroutine headers) return
// "".
func ExtractFrameFunction(frame string) string {
	f := NormalizeFrame(frame)
	if f == "" || strings.HasPrefix(f, "/") || strings.HasPrefix(f, "goroutine") || !strings.HasSuffix(f, ")") {
		return ""
	}
	fields := strings.Fields(f)
	if len(fields) == 0 {
		return ""
	}
	fn := fields[0]
	if i := strings.LastIndex(fn, "("); i >= 0 {
		fn = fn[:i]
	}
	return fn
}

// normalizeMessageForGrouping masks volatile values (timestamps, emails, URLs,
// UUIDs, IPs, hex pointers, numbers) so messages that differ only in those
// group together, Sentry-style.
func normalizeMessageForGrouping(msg string) string {
	m := strings.TrimSpace(msg)
	m = tsRe.ReplaceAllString(m, "*")
	m = emailRe.ReplaceAllString(m, "*")
	m = urlRe.ReplaceAllString(m, "*")
	m = uuidRe.ReplaceAllString(m, "*")
	m = addrRe.ReplaceAllString(m, "*")
	m = ipRe.ReplaceAllString(m, "*")
	m = numRe.ReplaceAllString(m, "*")
	return spaceRe.ReplaceAllString(m, " ")
}

// ComputeFingerprint hashes (project, error type, top N normalized frames) into
// the dedup key. Identical errors across runs produce the same fingerprint.
// It is equivalent to ComputeFingerprintMode with ModeTypeAndFrames.
func ComputeFingerprint(project, errorType string, stackTrace string) string {
	return ComputeFingerprintMode(project, errorType, stackTrace, "", ModeTypeAndFrames)
}

// ComputeFingerprintMode hashes the configured inputs into the dedup key. In
// "message" mode the normalized message is included; in "type" mode neither
// the message nor the stack is.
func ComputeFingerprintMode(project, errorType, stackTrace, message string, mode FingerprintMode) string {
	h := sha256.New()
	h.Write([]byte(project))
	h.Write([]byte{0})
	h.Write([]byte(errorType))
	h.Write([]byte{0})

	switch mode {
	case ModeTypeOnly:
		return hex.EncodeToString(h.Sum(nil))
	case ModeMessage:
		h.Write([]byte(normalizeMessage(message)))
		h.Write([]byte{0})
		return hex.EncodeToString(h.Sum(nil))
	case ModeHeuristic:
		fn := ""
		for _, line := range strings.Split(stackTrace, "\n") {
			if f := ExtractFrameFunction(line); f != "" {
				fn = f
				break
			}
		}
		h.Write([]byte(fn))
		h.Write([]byte{0})
		h.Write([]byte(normalizeMessageForGrouping(message)))
		h.Write([]byte{0})
		return hex.EncodeToString(h.Sum(nil))
	default:
		frames := ExtractFrames(stackTrace)
		if len(frames) > maxFrames {
			frames = frames[:maxFrames]
		}
		for _, f := range frames {
			h.Write([]byte(f))
			h.Write([]byte{0})
		}
		return hex.EncodeToString(h.Sum(nil))
	}
}

func normalizeMessage(msg string) string {
	return spaceRe.ReplaceAllString(strings.TrimSpace(msg), " ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
