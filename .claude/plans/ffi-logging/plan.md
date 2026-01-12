# Feature: FFI Logging with Charm Log

## Overview

Add comprehensive logging to the Go wrapper around the Rust jj-lib FFI layer using Charm's logging library (`github.com/charmbracelet/log`). Every FFI operation will be logged at INFO level, and all errors will be captured with ERROR level logging.

## Requirements

### Acceptance Criteria
- [ ] All FFI operations in `jj/internal/ffi/ffi.go` are wrapped with logging
- [ ] INFO level logs capture: operation name, parameters (sanitized), duration
- [ ] ERROR level logs capture: operation name, error message, context
- [ ] Logging is configurable (enable/disable, log level, output destination)
- [ ] Log output is compatible with bubbletea TUI (writes to file, not stdout)
- [ ] No performance regression on hot paths

### User Stories
1. As a developer, I want to see what FFI operations are happening so I can debug issues
2. As a developer, I want to see timing information to identify slow operations
3. As a user, I want logs to go to a file so they don't interfere with the TUI

## Architecture

### Current Structure
```
jj/
  jj.go           # Type definitions (Branch, Workspace, etc.)
  repo.go         # High-level Go API (Repo struct and methods)
  cli.go          # CLI fallback (uses jj command directly)
  internal/
    ffi/
      ffi.go      # CGO bindings to Rust library
      bridge.h    # C header for FFI
```

### Design Decision: Log at FFI Layer

**Option A: Log in `jj/repo.go` (high-level)**
- Pros: Cleaner separation, easier to maintain
- Cons: Misses internal errors, doesn't capture raw FFI details

**Option B: Log in `jj/internal/ffi/ffi.go` (low-level)** - CHOSEN
- Pros: Captures ALL FFI calls including errors, closest to the metal
- Cons: More verbose, tightly coupled to FFI

**Rationale**: Since the goal is to "wrap every operation and capture everything in logging", Option B provides the most comprehensive coverage.

### New Structure
```
jj/
  jj.go           # Type definitions (unchanged)
  repo.go         # High-level Go API (unchanged)
  cli.go          # CLI fallback (unchanged)
  internal/
    ffi/
      ffi.go      # CGO bindings (add logging to each function)
      bridge.h    # C header (unchanged)
      logger.go   # NEW: Logger configuration and initialization
```

### Logger Design

```go
// jj/internal/ffi/logger.go
package ffi

import (
    "os"
    "time"
    "github.com/charmbracelet/log"
)

var logger *log.Logger

// InitLogger initializes the FFI logger.
// If logPath is empty, logging is disabled.
func InitLogger(logPath string, level log.Level) error

// SetLogger allows injecting a custom logger (for testing)
func SetLogger(l *log.Logger)

// logOp logs an operation start and returns a function to log completion
func logOp(op string, args ...any) func(err error)
```

### Logging Pattern

Each FFI function will follow this pattern:

```go
func SomeOperation(repo RepoPtr, param string) ([]byte, error) {
    done := logOp("SomeOperation", "param", param)

    // ... existing FFI code ...

    if result.error != nil {
        err := errors.New(errMsg)
        done(err)  // logs ERROR
        return nil, err
    }

    done(nil)  // logs INFO with duration
    return data, nil
}
```

### Log Format

```
INFO  [FFI] OpenRepo path=/Users/foo/repo duration=12ms
INFO  [FFI] ListBranches duration=3ms count=5
ERROR [FFI] GetDiff error="null repo handle"
INFO  [FFI] SetBookmark name=main revision=abc123 duration=45ms
```

## Testability

### Test Strategy
- Unit tests for logger initialization and configuration
- Unit tests verify log output format (capture log output in tests)
- Integration tests ensure logging doesn't break FFI operations
- No mocking of FFI layer itself (already tested)

### Test Files Needed
- `jj/internal/ffi/logger_test.go` - Logger unit tests

### Test Considerations
- Logger must be injectable for testing
- Tests should verify log messages contain expected fields
- Tests should verify errors are logged at ERROR level

## Deployability

### Dependencies
- Add `github.com/charmbracelet/log` to go.mod
- No Rust-side changes needed
- No infrastructure changes

### CI/CD Changes
- None required (library addition only)

### Rollout Strategy
1. Add logger with logging disabled by default
2. Enable via environment variable: `JJAZY_LOG_FILE=/path/to/log`
3. Optional: Add `--debug` flag to CLI

## Security

### Risks Identified
1. **Log injection**: Malicious file paths or bookmark names could inject log entries
2. **Sensitive data exposure**: File contents or diffs might contain secrets
3. **Log file permissions**: Log file could be world-readable

### Mitigations
1. Sanitize logged parameters (truncate, escape newlines)
2. Never log file contents - only log paths and operation metadata
3. Create log files with 0600 permissions
4. Document that log files may contain repo metadata

### Security Review Checklist
- [ ] No file contents are logged
- [ ] No diff contents are logged
- [ ] Parameters are truncated to reasonable lengths
- [ ] Log file permissions are restrictive

## Tasks

Implementation tasks are in the `tasks/` directory:

1. **task-1.md**: Add charmbracelet/log dependency
2. **task-2.md**: Create logger.go with initialization and helpers
3. **task-3.md**: Add logging to all FFI functions
4. **task-4.md**: Add environment variable configuration
5. **task-5.md**: Add logger tests

### Suggested Order
Tasks should be implemented in order (1 -> 2 -> 3 -> 4 -> 5).

### Dependencies
- Task 2 depends on Task 1
- Task 3 depends on Task 2
- Task 4 depends on Task 2
- Task 5 depends on Task 2, Task 3

## FFI Functions to Wrap

All functions in `jj/internal/ffi/ffi.go`:

| Function | Parameters to Log | Notes |
|----------|------------------|-------|
| `OpenRepo` | path | Log success/failure |
| `ListBranches` | (none) | Log count of branches |
| `ListWorkspaces` | (none) | Log count of workspaces |
| `GetWorkingCopyChanges` | (none) | Log count of changes |
| `ListOperations` | (none) | Log count of operations |
| `GetLog` | (none) | Log count of revisions |
| `GetDiff` | (none) | Log success only, not content |
| `GetFileDiff` | path | Log path, not content |
| `GetFileContents` | path | Log path, not content |
| `GetRevisionDiff` | revisionID | Log revision, not content |
| `CloseRepo` | (none) | Log close event |
| `SetBookmark` | name, revisionID, flags | Log all params |

## Open Questions

1. Should we log at the `jj/repo.go` level too for higher-level context?
   - **Decision**: No, FFI layer is sufficient for debugging

2. Should we add structured logging (JSON format)?
   - **Decision**: No, human-readable is fine for debugging. Can add later if needed.

3. Should logging be opt-in or opt-out?
   - **Decision**: Opt-in via environment variable to avoid file creation by default
