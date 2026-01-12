# Task 4: Add Environment Variable Configuration

## Objective
Allow users to enable FFI logging via environment variable without code changes.

## Files to Modify
- `jj/internal/ffi/logger.go` (add auto-init)
- `main.go` (optional: explicit init call)

## Implementation

### Option A: Auto-init via init() function

Add to `logger.go`:

```go
func init() {
    // Auto-initialize from environment variables
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
        // Can't log the error, so print to stderr
        fmt.Fprintf(os.Stderr, "Warning: failed to initialize FFI logger: %v\n", err)
    }
}
```

### Option B: Explicit init in main.go (preferred)

This gives more control and visibility:

```go
// main.go
package main

import (
    "os"

    "github.com/charmbracelet/log"
    "github.com/gerund/jjazy/jj/internal/ffi"
)

func main() {
    // Initialize logging if configured
    if logPath := os.Getenv("JJAZY_LOG_FILE"); logPath != "" {
        level := parseLogLevel(os.Getenv("JJAZY_LOG_LEVEL"))
        if err := ffi.InitLogger(logPath, level); err != nil {
            fmt.Fprintf(os.Stderr, "Warning: failed to initialize logger: %v\n", err)
        }
    }

    // ... rest of main
}

func parseLogLevel(s string) log.Level {
    switch strings.ToLower(s) {
    case "debug":
        return log.DebugLevel
    case "warn", "warning":
        return log.WarnLevel
    case "error":
        return log.ErrorLevel
    default:
        return log.InfoLevel
    }
}
```

### Recommendation

Use **Option A** (auto-init) for simplicity. The `init()` function will run automatically when the package is imported, and the environment check is cheap.

However, this requires **exporting InitLogger** from the internal package. Since it's in `internal/`, only code within the jjazy module can import it anyway.

### Alternative: Export through jj package

Add to `jj/repo.go`:

```go
import "github.com/gerund/jjazy/jj/internal/ffi"

// InitLogging enables FFI operation logging to the specified file.
// Level can be: debug, info, warn, error
func InitLogging(logPath string, level string) error {
    lvl := log.InfoLevel
    switch strings.ToLower(level) {
    case "debug":
        lvl = log.DebugLevel
    case "warn", "warning":
        lvl = log.WarnLevel
    case "error":
        lvl = log.ErrorLevel
    }
    return ffi.InitLogger(logPath, lvl)
}
```

Then in `main.go`:

```go
if logPath := os.Getenv("JJAZY_LOG_FILE"); logPath != "" {
    level := os.Getenv("JJAZY_LOG_LEVEL")
    if err := jj.InitLogging(logPath, level); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: failed to initialize logger: %v\n", err)
    }
}
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `JJAZY_LOG_FILE` | Path to log file. If empty, logging is disabled. | (empty) |
| `JJAZY_LOG_LEVEL` | Log level: debug, info, warn, error | info |

## Usage Examples

```bash
# Enable logging to file
JJAZY_LOG_FILE=~/.jjazy.log jjazy

# Enable debug logging
JJAZY_LOG_FILE=/tmp/jjazy-debug.log JJAZY_LOG_LEVEL=debug jjazy

# One-time debugging
JJAZY_LOG_FILE=/tmp/debug.log jjazy && cat /tmp/debug.log
```

## Time Estimate
20 minutes

## Acceptance Criteria
- [ ] Setting `JJAZY_LOG_FILE` enables logging to that file
- [ ] `JJAZY_LOG_LEVEL` controls log verbosity
- [ ] Logging is disabled by default (no file created)
- [ ] Invalid log path shows warning but doesn't crash
- [ ] Documentation in README (optional, can defer)
