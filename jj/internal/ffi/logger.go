package ffi

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

var (
	logger     *log.Logger
	loggerOnce sync.Once
	logEnabled bool
)

// init auto-initializes the logger from environment variables.
// Set JJAZY_LOG_FILE to enable logging to a file.
// Set JJAZY_LOG_LEVEL to control verbosity (debug, info, warn, error).
func init() {
	logPath := os.Getenv("JJAZY_LOG_FILE")
	if logPath == "" {
		return // Logging disabled by default
	}

	levelStr := os.Getenv("JJAZY_LOG_LEVEL")
	level := log.InfoLevel
	switch strings.ToLower(levelStr) {
	case "debug":
		level = log.DebugLevel
	case "warn", "warning":
		level = log.WarnLevel
	case "error":
		level = log.ErrorLevel
	}

	if err := InitLogger(logPath, level); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize FFI logger: %v\n", err)
	}
}

// InitLogger initializes the FFI logger to write to the specified file.
// If logPath is empty, logging is disabled.
// This should be called early in application startup.
func InitLogger(logPath string, level log.Level) error {
	var initErr error
	loggerOnce.Do(func() {
		if logPath == "" {
			logEnabled = false
			return
		}

		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			initErr = err
			return
		}

		logger = log.NewWithOptions(f, log.Options{
			Level:           level,
			Prefix:          "FFI",
			ReportTimestamp: true,
			ReportCaller:    false,
		})
		logEnabled = true
	})
	return initErr
}

// SetLogger allows injecting a custom logger (useful for testing).
func SetLogger(l *log.Logger) {
	logger = l
	logEnabled = l != nil
}

// logOp creates a logging context for an operation.
// Returns a function that should be called when the operation completes.
//
// Usage:
//
//	done := logOp("OpenRepo", "path", path)
//	defer done(nil) // or done(err) on error
func logOp(op string, keyvals ...any) func(error) {
	if !logEnabled || logger == nil {
		return func(error) {}
	}

	start := time.Now()
	return func(err error) {
		duration := time.Since(start)

		// Build args: operation name, duration, then provided keyvals
		args := make([]any, 0, len(keyvals)+4)
		args = append(args, "op", op)
		args = append(args, "duration", duration.String())
		args = append(args, keyvals...)

		if err != nil {
			args = append(args, "error", err.Error())
			logger.Error("operation failed", args...)
		} else {
			logger.Info("operation complete", args...)
		}
	}
}

// logOpWithResult is like logOp but allows adding result info at completion.
//
// Usage:
//
//	done := logOpWithResult("ListBranches")
//	// ... operation ...
//	done(nil, "count", len(branches))
func logOpWithResult(op string, keyvals ...any) func(error, ...any) {
	if !logEnabled || logger == nil {
		return func(error, ...any) {}
	}

	start := time.Now()
	return func(err error, resultKeyvals ...any) {
		duration := time.Since(start)

		args := make([]any, 0, len(keyvals)+len(resultKeyvals)+4)
		args = append(args, "op", op)
		args = append(args, "duration", duration.String())
		args = append(args, keyvals...)
		args = append(args, resultKeyvals...)

		if err != nil {
			args = append(args, "error", err.Error())
			logger.Error("operation failed", args...)
		} else {
			logger.Info("operation complete", args...)
		}
	}
}

// truncate truncates a string to maxLen characters for safe logging.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
