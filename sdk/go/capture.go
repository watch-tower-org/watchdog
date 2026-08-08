package wt

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
)

// sdkPackagePrefix identifies frames that belong to this SDK so they can be
// trimmed from captured stacks (the first non-SDK frame is the real caller).
const sdkPackagePrefix = "github.com/watch-tower-org/watchdog/sdk/go"

// captureStack returns a formatted stack trace beginning at the first caller
// outside this SDK. Each frame is formatted as "funcname\n\t/file.go:line",
// which matches the format the backend's fingerprint normalizer expects.
//
// The function is marked noinline so it always has its own stack frame;
// otherwise the compiler may inline it and runtime.Callers would elide the
// callers we care about.
//
//go:noinline
func captureStack() string {
	pcs := make([]uintptr, 64)
	n := runtime.Callers(0, pcs)
	pcs = pcs[:n]

	var b strings.Builder
	seenFirst := false
	for _, pc := range pcs {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		name := fn.Name()
		if !seenFirst {
			// Skip this SDK's own frames as well as runtime bookkeeping
			// frames (runtime.Callers et al) until the first real caller.
			if strings.HasPrefix(name, sdkPackagePrefix) || strings.HasPrefix(name, "runtime.") {
				continue
			}
			seenFirst = true
		}
		file, line := fn.FileLine(pc)
		b.WriteString(name)
		b.WriteString("\n\t")
		b.WriteString(file)
		b.WriteString(":")
		b.WriteString(fmt.Sprintf("%d", line))
		b.WriteString("\n")
	}
	return b.String()
}

// capturePanicStack returns the goroutine dump captured while recovering a
// panic. Go keeps the panicking frames on the stack during unwinding, so the
// report shows where the panic originated.
func capturePanicStack() string {
	return trimPanicStack(string(debug.Stack()))
}

// trimPanicStack removes the goroutine header and any frames belonging to this
// SDK (including runtime/debug.Stack itself) so the reported stack starts at
// the panicking code.
func trimPanicStack(raw string) string {
	lines := strings.Split(raw, "\n")
	var b strings.Builder
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "goroutine ") || trimmed == "" || strings.HasPrefix(line, "\t") {
			continue
		}
		if strings.HasPrefix(trimmed, sdkPackagePrefix) || strings.HasPrefix(trimmed, "runtime/debug.Stack") {
			i++ // skip the paired location line
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
		if i+1 < len(lines) {
			b.WriteString(lines[i+1])
			b.WriteString("\n")
			i++
		}
	}
	return b.String()
}

// enrich pushes automatically-collected fields (hostname, release) into the
// event context, without clobbering user-supplied keys.
func enrich(e *Event, release string) {
	if e.Context == nil {
		e.Context = map[string]any{}
	}
	if _, ok := e.Context["hostname"]; !ok {
		if h, err := os.Hostname(); err == nil {
			e.Context["hostname"] = h
		}
	}
	if release != "" {
		if _, ok := e.Context["release"]; !ok {
			e.Context["release"] = release
		}
	}
}
