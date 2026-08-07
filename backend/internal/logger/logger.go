package logger

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
)

var (
	logger zerolog.Logger
	once   sync.Once
)

func InitLogger(cfg config.LoggerConfig) {
	once.Do(func() {
		zerolog.TimeFieldFormat = time.RFC3339
		level := parseLevel(cfg.Level)
		zerolog.SetGlobalLevel(level)

		var writers []io.Writer

		if cfg.Output == config.OutputStdout || cfg.Output == config.OutputBoth {
			writers = append(writers, os.Stdout)
		}
		if cfg.Output == config.OutputFile || cfg.Output == config.OutputBoth {
			lj := &lumberjack.Logger{
				Filename:   cfg.LogDir + "/" + cfg.Filename + ".log",
				MaxSize:    cfg.MaxSizeMB,
				MaxBackups: cfg.MaxBackups,
				MaxAge:     cfg.MaxAgeDays,
				Compress:   cfg.Compress,
			}
			writers = append(writers, lj)
		}

		if len(writers) == 0 {
			writers = append(writers, os.Stdout)
		}

		var writer io.Writer
		if len(writers) == 1 {
			writer = writers[0]
		} else {
			writer = zerolog.MultiLevelWriter(writers...)
		}

		if cfg.EnableCaller {
			logger = zerolog.New(writer).With().Timestamp().Caller().Logger()
		} else {
			logger = zerolog.New(writer).With().Timestamp().Logger()
		}
	})
}

func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "trace":
		return zerolog.TraceLevel
	default:
		return zerolog.InfoLevel
	}
}

func Info() *zerolog.Event {
	return logger.Info()
}

func Debug() *zerolog.Event {
	return logger.Debug()
}

func Warn() *zerolog.Event {
	return logger.Warn()
}

func Error() *zerolog.Event {
	return logger.Error()
}

func Fatal() *zerolog.Event {
	return logger.Fatal()
}

func Ctx(ctx context.Context) *zerolog.Logger {
	l := zerolog.Ctx(ctx)
	if l == nil {
		return &logger
	}
	return l
}

func Close() {}
