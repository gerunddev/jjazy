# Task 2: Create logger.go with Initialization and Helpers

## Objective
Create a new file `jj/internal/ffi/logger.go` that provides logger initialization and helper functions for consistent logging across all FFI operations.

## Files to Create
- `jj/internal/ffi/logger.go`

## Implementation

### logger.go Structure

```go
package ffi

import (
    "os"
    "sync"
    "time"

    "github.com/charmbracelet/log"
)

var (
    logger     *log.Logger
    loggerOnce sync.Once
    logEnabled bool
)

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
//   done := logOp("OpenRepo", "path", path)
//   defer done(nil) // or done(err) on error
//
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
//   done := logOpWithResult("ListBranches")
//   // ... operation ...
//   done(nil, "count", len(branches))
//
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
```

## Key Design Decisions

1. **sync.Once for initialization**: Ensures logger is only initialized once, even if called multiple times
2. **logEnabled flag**: Fast path for disabled logging (no allocation)
3. **File permissions 0600**: Only owner can read/write the log file
4. **Duration tracking**: Built into the helper for automatic timing
5. **Separate logOpWithResult**: For operations where we want to log result metadata (like counts)

## Verification

```go
// Quick test
func TestLoggerInit(t *testing.T) {
    tmpFile := filepath.Join(t.TempDir(), "test.log")
    err := InitLogger(tmpFile, log.InfoLevel)
    if err != nil {
        t.Fatal(err)
    }

    done := logOp("TestOp", "key", "value")
    done(nil)

    // Verify file has content
    content, _ := os.ReadFile(tmpFile)
    if !strings.Contains(string(content), "TestOp") {
        t.Error("expected log to contain operation name")
    }
}
```

## Time Estimate
30 minutes

## Acceptance Criteria
- [ ] `jj/internal/ffi/logger.go` exists
- [ ] `InitLogger` creates log file with correct permissions
- [ ] `logOp` helper logs operation with duration
- [ ] `logOpWithResult` allows adding result metadata
- [ ] Disabled logging has no overhead (fast path)
- [ ] `SetLogger` works for testing injection
