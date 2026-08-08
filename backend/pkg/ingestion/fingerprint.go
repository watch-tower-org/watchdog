package ingestion

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/watch-tower-org/watchdog/backend/internal/model"
)

const maxFrames = 10

var (
	addrRe  = regexp.MustCompile(`0x[0-9a-f]+`)
	posRe   = regexp.MustCompile(`\+0x[0-9a-f]+$`)
	argsRe  = regexp.MustCompile(`\(0x[0-9a-f]+(?:, 0x[0-9a-f]+)*\)`)
	fileRe  = regexp.MustCompile(`\.go:\d+`)
	spaceRe = regexp.MustCompile(`\s+`)
)

// ExtractErrorType returns the error type for a report. If the caller provided
// one it wins; otherwise it is derived from the stack trace (first panic/error
// marker) or falls back to the message.
func ExtractErrorType(msg, stackTrace string) string {
	if msg != "" {
		parts := strings.SplitN(msg, ":", 2)
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			return strings.TrimSpace(parts[0])
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

// ComputeFingerprint hashes (project, error type, top N normalized frames) into
// the dedup key. Identical errors across runs produce the same fingerprint.
func ComputeFingerprint(project, errorType string, stackTrace string) string {
	frames := ExtractFrames(stackTrace)
	if len(frames) > maxFrames {
		frames = frames[:maxFrames]
	}

	h := sha256.New()
	h.Write([]byte(project))
	h.Write([]byte{0})
	h.Write([]byte(errorType))
	h.Write([]byte{0})
	for _, f := range frames {
		h.Write([]byte(f))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
