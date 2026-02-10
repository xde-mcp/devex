package logging

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	clog "github.com/charmbracelet/log"
)

const (
	envLevel      = "LOG_LEVEL"
	envFormat     = "LOG_FORMAT"
	envCaller     = "LOG_CALLER"
	envTimestamp  = "LOG_TIMESTAMP"
	envTimeFormat = "LOG_TIME_FORMAT"
)

var defaultLogger atomic.Pointer[clog.Logger]

// Init builds and installs the shared logger for the current process.
func Init(app string) *clog.Logger {
	logger := New(app)
	defaultLogger.Store(logger)
	clog.SetDefault(logger)
	return logger
}

// Get returns the shared logger if initialized, or the default logger otherwise.
func Get() *clog.Logger {
	if logger := defaultLogger.Load(); logger != nil {
		return logger
	}
	return clog.Default()
}

// New constructs a logger using environment-driven configuration.
func New(app string) *clog.Logger {
	opts := clog.Options{
		Level:           parseLevel(envString(envLevel, "info")),
		Formatter:       parseFormatter(envString(envFormat, "text")),
		ReportCaller:    envBool(envCaller, false),
		ReportTimestamp: envBool(envTimestamp, true),
		TimeFormat:      envString(envTimeFormat, clog.DefaultTimeFormat),
		Fields:          []interface{}{"app", app},
	}

	return clog.NewWithOptions(os.Stderr, opts)
}

// Println mirrors log.Println using the shared logger.
func Println(args ...interface{}) {
	logger := Get()
	logger.Helper()
	logger.Info(fmt.Sprint(args...))
}

// Debug logs with Debug level.
func Debug(msg interface{}, keyvals ...interface{}) {
	logger := Get()
	logger.Helper()
	logger.Debug(msg, keyvals...)
}

// Info logs with Info level.
func Info(msg interface{}, keyvals ...interface{}) {
	logger := Get()
	logger.Helper()
	logger.Info(msg, keyvals...)
}

// Warn logs with Warn level.
func Warn(msg interface{}, keyvals ...interface{}) {
	logger := Get()
	logger.Helper()
	logger.Warn(msg, keyvals...)
}

// Error logs with Error level.
func Error(msg interface{}, keyvals ...interface{}) {
	logger := Get()
	logger.Helper()
	logger.Error(msg, keyvals...)
}

// Debugf logs a formatted message with Debug level.
func Debugf(format string, args ...interface{}) {
	logger := Get()
	logger.Helper()
	logger.Debugf(format, args...)
}

// Infof logs a formatted message with Info level.
func Infof(format string, args ...interface{}) {
	logger := Get()
	logger.Helper()
	logger.Infof(format, args...)
}

// Warnf logs a formatted message with Warn level.
func Warnf(format string, args ...interface{}) {
	logger := Get()
	logger.Helper()
	logger.Warnf(format, args...)
}

// Errorf logs a formatted message with Error level.
func Errorf(format string, args ...interface{}) {
	logger := Get()
	logger.Helper()
	logger.Errorf(format, args...)
}

// Printf mirrors log.Printf using the shared logger.
func Printf(format string, args ...interface{}) {
	logger := Get()
	logger.Helper()
	logger.Infof(format, args...)
}

// Fatal mirrors log.Fatal using the shared logger.
func Fatal(args ...interface{}) {
	logger := Get()
	logger.Helper()
	logger.Fatal(fmt.Sprint(args...))
}

// Fatalf mirrors log.Fatalf using the shared logger.
func Fatalf(format string, args ...interface{}) {
	logger := Get()
	logger.Helper()
	logger.Fatalf(format, args...)
}

func envString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func parseLevel(value string) clog.Level {
	level, err := clog.ParseLevel(strings.ToLower(strings.TrimSpace(value)))
	if err != nil {
		return clog.InfoLevel
	}
	return level
}

func parseFormatter(value string) clog.Formatter {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "json":
		return clog.JSONFormatter
	case "logfmt":
		return clog.LogfmtFormatter
	default:
		return clog.TextFormatter
	}
}
