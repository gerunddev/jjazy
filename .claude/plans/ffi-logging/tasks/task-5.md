# Task 5: Add Logger Tests

## Objective
Create unit tests for the logger functionality to ensure correct behavior.

## Files to Create
- `jj/internal/ffi/logger_test.go`

## Implementation

```go
package ffi

import (
    "bytes"
    "errors"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/charmbracelet/log"
)

func TestInitLogger_CreatesFile(t *testing.T) {
    // Reset state for test
    resetLogger()

    tmpFile := filepath.Join(t.TempDir(), "test.log")
    err := InitLogger(tmpFile, log.InfoLevel)
    if err != nil {
        t.Fatalf("InitLogger failed: %v", err)
    }

    // Verify file exists
    if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
        t.Error("log file was not created")
    }
}

func TestInitLogger_FilePermissions(t *testing.T) {
    resetLogger()

    tmpFile := filepath.Join(t.TempDir(), "test.log")
    err := InitLogger(tmpFile, log.InfoLevel)
    if err != nil {
        t.Fatalf("InitLogger failed: %v", err)
    }

    info, err := os.Stat(tmpFile)
    if err != nil {
        t.Fatalf("stat failed: %v", err)
    }

    perm := info.Mode().Perm()
    if perm != 0600 {
        t.Errorf("expected permissions 0600, got %o", perm)
    }
}

func TestInitLogger_EmptyPath_DisablesLogging(t *testing.T) {
    resetLogger()

    err := InitLogger("", log.InfoLevel)
    if err != nil {
        t.Fatalf("InitLogger failed: %v", err)
    }

    if logEnabled {
        t.Error("logging should be disabled with empty path")
    }
}

func TestInitLogger_OnlyOnce(t *testing.T) {
    resetLogger()

    tmpFile1 := filepath.Join(t.TempDir(), "test1.log")
    tmpFile2 := filepath.Join(t.TempDir(), "test2.log")

    InitLogger(tmpFile1, log.InfoLevel)
    InitLogger(tmpFile2, log.InfoLevel) // Should be ignored

    // Write a log
    done := logOp("Test")
    done(nil)

    // Only first file should have content
    content1, _ := os.ReadFile(tmpFile1)
    content2, _ := os.ReadFile(tmpFile2)

    if len(content1) == 0 {
        t.Error("first log file should have content")
    }
    if len(content2) > 0 {
        t.Error("second log file should be empty")
    }
}

func TestLogOp_Success(t *testing.T) {
    buf := setupTestLogger(t)

    done := logOp("TestOperation", "key", "value")
    done(nil)

    output := buf.String()

    if !strings.Contains(output, "TestOperation") {
        t.Error("log should contain operation name")
    }
    if !strings.Contains(output, "duration") {
        t.Error("log should contain duration")
    }
    if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
        t.Error("log should contain key-value pair")
    }
    if strings.Contains(output, "ERROR") {
        t.Error("success should not log at ERROR level")
    }
}

func TestLogOp_Error(t *testing.T) {
    buf := setupTestLogger(t)

    done := logOp("TestOperation")
    done(errors.New("test error"))

    output := buf.String()

    if !strings.Contains(output, "TestOperation") {
        t.Error("log should contain operation name")
    }
    if !strings.Contains(output, "test error") {
        t.Error("log should contain error message")
    }
    if !strings.Contains(output, "ERROR") {
        t.Error("error should log at ERROR level")
    }
}

func TestLogOpWithResult_AddsResultInfo(t *testing.T) {
    buf := setupTestLogger(t)

    done := logOpWithResult("TestOperation")
    done(nil, "count", 42, "extra", "data")

    output := buf.String()

    if !strings.Contains(output, "count") {
        t.Error("log should contain result key")
    }
    if !strings.Contains(output, "42") {
        t.Error("log should contain result value")
    }
}

func TestTruncate(t *testing.T) {
    tests := []struct {
        input    string
        maxLen   int
        expected string
    }{
        {"short", 10, "short"},
        {"exactly10!", 10, "exactly10!"},
        {"this is too long", 10, "this is to..."},
        {"", 10, ""},
    }

    for _, tc := range tests {
        result := truncate(tc.input, tc.maxLen)
        if result != tc.expected {
            t.Errorf("truncate(%q, %d) = %q, want %q",
                tc.input, tc.maxLen, result, tc.expected)
        }
    }
}

func TestLogOp_DisabledLogging(t *testing.T) {
    resetLogger()
    // Don't initialize logger

    // Should not panic
    done := logOp("TestOperation")
    done(nil)
    done(errors.New("error"))
}

// Helper functions

func resetLogger() {
    // Reset the sync.Once - this is a hack for testing
    // In production, InitLogger is only called once
    loggerOnce = sync.Once{}
    logger = nil
    logEnabled = false
}

func setupTestLogger(t *testing.T) *bytes.Buffer {
    t.Helper()
    resetLogger()

    buf := &bytes.Buffer{}
    logger = log.NewWithOptions(buf, log.Options{
        Level:           log.DebugLevel,
        Prefix:          "FFI",
        ReportTimestamp: false, // Easier to test without timestamps
    })
    logEnabled = true

    return buf
}
```

## Test Coverage Goals

| Function | Coverage |
|----------|----------|
| `InitLogger` | File creation, permissions, empty path, once semantics |
| `SetLogger` | Injection works |
| `logOp` | Success logging, error logging, disabled state |
| `logOpWithResult` | Result metadata included |
| `truncate` | Various lengths |

## Notes

- Tests use `sync.Once` reset hack - this is acceptable for testing
- Buffer-based logger avoids file I/O in most tests
- Timestamp disabled in test logger for easier assertions

## Time Estimate
30 minutes

## Acceptance Criteria
- [ ] All logger functions have test coverage
- [ ] Tests pass: `go test ./jj/internal/ffi/...`
- [ ] Tests verify log format includes operation name
- [ ] Tests verify error vs success logging levels
- [ ] Tests verify truncation works correctly
- [ ] Tests verify disabled logging doesn't panic
