package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorYellow  = "\033[33m"
	colorWhite   = "\033[37m"
	colorGreen   = "\033[32m"
	colorBlue    = "\033[34m"
	colorCyan    = "\033[36m"
	colorMagenta = "\033[35m"

	timeFormat = "2006-01-02 15:04:05.000"
)

var (
	once sync.Once
	dw   *dailyWriter
)

type dailyWriter struct {
	mu         sync.Mutex
	inner      *lumberjack.Logger
	dir        string
	prefix     string
	date       string
	maxSize    int
	maxBackups int
	maxAge     int
	compress   bool
}

func newDailyWriter(dir, prefix string, maxSize, maxBackups, maxAge int, compress bool) *dailyWriter {
	w := &dailyWriter{
		dir:        dir,
		prefix:     prefix,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxAge:     maxAge,
		compress:   compress,
	}
	w.rotate(time.Now())
	return w
}

func (w *dailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	today := now.Format("2006-01-02")
	if today != w.date {
		if err := w.inner.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "logger: failed to close previous log file: %v\n", err)
		}
		w.rotate(now)
	}
	return w.inner.Write(p)
}

func (w *dailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.inner != nil {
		return w.inner.Close()
	}
	return nil
}

func (w *dailyWriter) rotate(now time.Time) {
	date := now.Format("2006-01-02")
	w.date = date
	w.inner = &lumberjack.Logger{
		Filename:   filepath.Join(w.dir, fmt.Sprintf("%s-%s.log", w.prefix, date)),
		MaxSize:    w.maxSize,
		MaxBackups: w.maxBackups,
		MaxAge:     w.maxAge,
		Compress:   w.compress,
	}
}

func InitLogger(cfg config.LoggerConfig) {
	once.Do(func() {
		level := ParseLevel(cfg.Level)
		zerolog.SetGlobalLevel(level)
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		zerolog.TimeFieldFormat = timeFormat

		var writers []io.Writer

		if cfg.Output == config.OutputStdout || cfg.Output == config.OutputBoth {
			writers = append(writers, consoleWriter(cfg))
		}

		if cfg.Output == config.OutputFile || cfg.Output == config.OutputBoth {
			dw = newDailyWriter(
				cfg.LogDir, cfg.Filename,
				cfg.MaxSizeMB, cfg.MaxBackups, cfg.MaxAgeDays, cfg.Compress,
			)
			writers = append(writers, dw)
		}

		l := zerolog.New(io.MultiWriter(writers...)).With().Timestamp()
		if cfg.EnableCaller {
			l = l.Caller()
		}
		log.Logger = l.Logger()
	})
}

func Close() {
	if dw != nil {
		if err := dw.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "logger: error closing log file: %v\n", err)
		}
	}
}

// LevelHookFn receives a log level and its message. It is invoked for every
// message at or above the installed minimum level. Fatal/Panic levels should be
// handled synchronously (zerolog calls os.Exit immediately after writing).
type LevelHookFn func(level zerolog.Level, message string)

type levelHook struct {
	min zerolog.Level
	fn  LevelHookFn
}

func (h *levelHook) Run(e *zerolog.Event, level zerolog.Level, message string) {
	if h == nil || h.fn == nil {
		return
	}
	if level < h.min {
		return
	}
	h.fn(level, message)
}

// InstallLevelHook wraps the global logger with a hook that forwards every
// message at or above min to fn. It must be called after InitLogger.
func InstallLevelHook(min zerolog.Level, fn LevelHookFn) {
	log.Logger = log.Logger.Hook(&levelHook{min: min, fn: fn})
}

func consoleWriter(cfg config.LoggerConfig) zerolog.ConsoleWriter {
	return zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    false,
		TimeFormat: timeFormat,
		FormatCaller: func(i interface{}) string {
			s, ok := i.(string)
			if !ok || !cfg.EnableCaller {
				return ""
			}
			return colorBlue + filepath.Base(s) + " >" + colorReset
		},
		FormatTimestamp: func(i interface{}) string {
			s, ok := i.(string)
			if !ok {
				return ""
			}
			return colorWhite + "• " + s + " •" + colorReset
		},
		FormatLevel: func(i interface{}) string {
			s, ok := i.(string)
			if !ok {
				return ""
			}
			return "|" + levelColor(s) + "|"
		},
		FormatMessage: func(i interface{}) string {
			s, _ := i.(string)
			return s
		},
		PartsOrder: []string{
			zerolog.TimestampFieldName,
			zerolog.LevelFieldName,
			zerolog.CallerFieldName,
			zerolog.MessageFieldName,
		},
	}
}

func levelColor(level string) string {
	switch strings.ToUpper(level) {
	case "INFO":
		return colorYellow + level + colorReset
	case "ERROR", "FATAL":
		return colorRed + level + colorReset
	case "WARN":
		return colorMagenta + level + colorReset
	case "DEBUG":
		return colorGreen + level + colorReset
	case "TRACE":
		return colorCyan + level + colorReset
	default:
		return colorWhite + level + colorReset
	}
}

// ParseLevel maps a config string ("info", "error", ...) to a zerolog level.
// Unknown or empty values fall back to InfoLevel.
func ParseLevel(logLevel string) zerolog.Level {
	switch strings.ToUpper(logLevel) {
	case "INFO":
		return zerolog.InfoLevel
	case "WARN":
		return zerolog.WarnLevel
	case "ERROR":
		return zerolog.ErrorLevel
	case "DEBUG":
		return zerolog.DebugLevel
	case "TRACE":
		return zerolog.TraceLevel
	case "FATAL":
		return zerolog.FatalLevel
	case "NONE":
		return zerolog.NoLevel
	case "SILENT":
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
}

func Ctx(ctx context.Context) *zerolog.Logger {
	l := zerolog.Ctx(ctx)
	if l.GetLevel() == zerolog.Disabled {
		return &log.Logger
	}
	return l
}

func Warn() *zerolog.Event  { return log.Warn() }
func Info() *zerolog.Event  { return log.Info() }
func Error() *zerolog.Event { return log.Error() }
func Debug() *zerolog.Event { return log.Debug() }
func Trace() *zerolog.Event { return log.Trace() }
func Fatal() *zerolog.Event { return log.Fatal() }
