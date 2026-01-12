# FFI Logging Implementation Notes

## Implementation Summary

All tasks from the plan have been completed:

1. **Task 1: Add charmbracelet/log dependency** - Done
   - Added `github.com/charmbracelet/log v0.4.2` to go.mod
   - Some dependent packages were upgraded (lipgloss, termenv, etc.)

2. **Task 2: Create logger.go** - Done
   - Created `jj/internal/ffi/logger.go` with:
     - `InitLogger()` for file-based logging initialization
     - `SetLogger()` for test injection
     - `logOp()` helper for simple operations
     - `logOpWithResult()` helper for operations that return data
     - `truncate()` helper for safe string truncation

3. **Task 3: Add logging to all FFI functions** - Done
   - All 12 FFI functions now have logging:
     - `OpenRepo`, `CloseRepo`, `SetBookmark` use `logOp()`
     - All other functions use `logOpWithResult()` to log bytes returned

4. **Task 4: Environment variable configuration** - Done
   - Added `init()` function in logger.go for auto-initialization
   - `JJAZY_LOG_FILE` - path to log file (empty = disabled)
   - `JJAZY_LOG_LEVEL` - debug, info, warn, error (default: info)

5. **Task 5: Logger tests** - Done
   - Created `jj/internal/ffi/logger_test.go` with 12 tests
   - All tests pass

## Implementation Decisions

### Log at FFI Layer
Following the plan, logging is done at the FFI layer (`jj/internal/ffi/ffi.go`) rather than the high-level API. This captures all operations including internal errors.

### Auto-init via init()
Used Option A from Task 4 - auto-initialization via `init()` function. This is simpler than requiring explicit initialization in main.go, and the environment variable check is cheap (no-op when disabled).

### Result Size Logging
For functions that return data, we log `bytes` (the size of the returned data) rather than trying to count items. This is simpler and doesn't require parsing JSON.

### Truncation Lengths
- Paths: 100 characters
- Revision IDs: 12 characters (standard short form)
- Bookmark names: 50 characters

## Security Notes
- File contents and diff contents are NOT logged
- Only metadata (paths, sizes, durations) is logged
- Log files are created with 0600 permissions

## Testing Notes
- Tests use `sync.Once` reset hack to allow testing logger initialization multiple times
- The `resetLogger()` helper resets all global state between tests
- Buffer-based logger is used for most tests to avoid file I/O

## Usage

```bash
# Enable logging
JJAZY_LOG_FILE=~/.jjazy.log jjazy

# Enable debug logging
JJAZY_LOG_FILE=/tmp/debug.log JJAZY_LOG_LEVEL=debug jjazy
```
