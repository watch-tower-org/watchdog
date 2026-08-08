package logger

import (
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestLevelHookGatesBelowMin(t *testing.T) {
	var mu sync.Mutex
	var got []struct {
		level   zerolog.Level
		message string
	}
	h := &levelHook{
		min: zerolog.ErrorLevel,
		fn: func(level zerolog.Level, message string) {
			mu.Lock()
			defer mu.Unlock()
			got = append(got, struct {
				level   zerolog.Level
				message string
			}{level, message})
		},
	}

	h.Run(nil, zerolog.InfoLevel, "skip me")
	h.Run(nil, zerolog.ErrorLevel, "report me")
	h.Run(nil, zerolog.FatalLevel, "report this too")

	if len(got) != 2 {
		t.Fatalf("expected 2 forwarded messages, got %d", len(got))
	}
	if got[0].message != "report me" || got[0].level != zerolog.ErrorLevel {
		t.Errorf("first forwarded = %v", got[0])
	}
	if got[1].message != "report this too" || got[1].level != zerolog.FatalLevel {
		t.Errorf("second forwarded = %v", got[1])
	}
}

func TestLevelHookNilFnNoPanic(t *testing.T) {
	h := &levelHook{min: zerolog.WarnLevel}
	h.Run(nil, zerolog.FatalLevel, "ignored")
}

func TestInstallLevelHookForwardsGlobalLogs(t *testing.T) {
	prev := log.Logger
	defer func() { log.Logger = prev }()

	var mu sync.Mutex
	var messages []string
	InstallLevelHook(zerolog.ErrorLevel, func(level zerolog.Level, message string) {
		mu.Lock()
		defer mu.Unlock()
		messages = append(messages, message)
	})

	Error().Msg("something broke")
	Info().Msg("not forwarded")

	mu.Lock()
	defer mu.Unlock()
	if len(messages) != 1 || messages[0] != "something broke" {
		t.Fatalf("messages = %v", messages)
	}
}
