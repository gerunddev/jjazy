# Task 3: Add Logging to All FFI Functions

## Objective
Wrap every FFI function in `jj/internal/ffi/ffi.go` with logging using the helpers from task-2.

## Files to Modify
- `jj/internal/ffi/ffi.go`

## Implementation

### Pattern for Each Function

**Before:**
```go
func SomeFunc(repo RepoPtr) ([]byte, error) {
    // ... implementation ...
    if result.error != nil {
        return nil, errors.New(errMsg)
    }
    return data, nil
}
```

**After:**
```go
func SomeFunc(repo RepoPtr) ([]byte, error) {
    done := logOp("SomeFunc")

    // ... implementation ...
    if result.error != nil {
        err := errors.New(errMsg)
        done(err)
        return nil, err
    }

    done(nil)
    return data, nil
}
```

### Function-Specific Implementation

#### OpenRepo
```go
func OpenRepo(path string) (RepoPtr, error) {
    done := logOp("OpenRepo", "path", truncate(path, 100))

    cpath := C.CString(path)
    defer C.free(unsafe.Pointer(cpath))

    handle := C.jj_open_repo(cpath)
    if handle == nil {
        err := errors.New("failed to open repository")
        done(err)
        return nil, err
    }
    done(nil)
    return RepoPtr(handle), nil
}
```

#### ListBranches
```go
func ListBranches(repo RepoPtr) ([]byte, error) {
    done := logOpWithResult("ListBranches")

    result := C.jj_list_branches((*C.RepoHandle)(repo))
    defer C.jj_free_result(result)

    if result.error != nil {
        errMsg := C.GoString(result.error)
        err := errors.New(errMsg)
        done(err)
        return nil, err
    }

    if result.data == nil {
        err := errors.New("no data returned")
        done(err)
        return nil, err
    }

    data := []byte(C.GoString(result.data))
    // Count branches for logging (quick JSON parse or len estimate)
    done(nil, "bytes", len(data))
    return data, nil
}
```

#### ListWorkspaces
```go
func ListWorkspaces(repo RepoPtr) ([]byte, error) {
    done := logOpWithResult("ListWorkspaces")

    result := C.jj_list_workspaces((*C.RepoHandle)(repo))
    defer C.jj_free_result(result)

    if result.error != nil {
        errMsg := C.GoString(result.error)
        err := errors.New(errMsg)
        done(err)
        return nil, err
    }

    if result.data == nil {
        err := errors.New("no data returned")
        done(err)
        return nil, err
    }

    data := []byte(C.GoString(result.data))
    done(nil, "bytes", len(data))
    return data, nil
}
```

#### GetWorkingCopyChanges
```go
func GetWorkingCopyChanges(repo RepoPtr) ([]byte, error) {
    done := logOpWithResult("GetWorkingCopyChanges")

    result := C.jj_get_working_copy_changes((*C.RepoHandle)(repo))
    defer C.jj_free_result(result)

    if result.error != nil {
        errMsg := C.GoString(result.error)
        err := errors.New(errMsg)
        done(err)
        return nil, err
    }

    if result.data == nil {
        err := errors.New("no data returned")
        done(err)
        return nil, err
    }

    data := []byte(C.GoString(result.data))
    done(nil, "bytes", len(data))
    return data, nil
}
```

#### ListOperations
```go
func ListOperations(repo RepoPtr) ([]byte, error) {
    done := logOpWithResult("ListOperations")

    result := C.jj_list_operations((*C.RepoHandle)(repo))
    defer C.jj_free_result(result)

    if result.error != nil {
        errMsg := C.GoString(result.error)
        err := errors.New(errMsg)
        done(err)
        return nil, err
    }

    if result.data == nil {
        err := errors.New("no data returned")
        done(err)
        return nil, err
    }

    data := []byte(C.GoString(result.data))
    done(nil, "bytes", len(data))
    return data, nil
}
```

#### GetLog
```go
func GetLog(repo RepoPtr) ([]byte, error) {
    done := logOpWithResult("GetLog")

    result := C.jj_get_log((*C.RepoHandle)(repo))
    defer C.jj_free_result(result)

    if result.error != nil {
        errMsg := C.GoString(result.error)
        err := errors.New(errMsg)
        done(err)
        return nil, err
    }

    if result.data == nil {
        err := errors.New("no data returned")
        done(err)
        return nil, err
    }

    data := []byte(C.GoString(result.data))
    done(nil, "bytes", len(data))
    return data, nil
}
```

#### GetDiff
```go
func GetDiff(repo RepoPtr) (string, error) {
    done := logOpWithResult("GetDiff")

    result := C.jj_get_diff((*C.RepoHandle)(repo))
    defer C.jj_free_result(result)

    if result.error != nil {
        errMsg := C.GoString(result.error)
        err := errors.New(errMsg)
        done(err)
        return "", err
    }

    if result.data == nil {
        done(nil, "bytes", 0)
        return "", nil
    }

    data := C.GoString(result.data)
    done(nil, "bytes", len(data))
    return data, nil
}
```

#### GetFileDiff
```go
func GetFileDiff(repo RepoPtr, path string) (string, error) {
    done := logOpWithResult("GetFileDiff", "path", truncate(path, 100))

    cpath := C.CString(path)
    defer C.free(unsafe.Pointer(cpath))

    result := C.jj_get_file_diff((*C.RepoHandle)(repo), cpath)
    defer C.jj_free_result(result)

    if result.error != nil {
        errMsg := C.GoString(result.error)
        err := errors.New(errMsg)
        done(err)
        return "", err
    }

    if result.data == nil {
        done(nil, "bytes", 0)
        return "", nil
    }

    data := C.GoString(result.data)
    done(nil, "bytes", len(data))
    return data, nil
}
```

#### GetFileContents
```go
func GetFileContents(repo RepoPtr, path string) ([]byte, error) {
    done := logOpWithResult("GetFileContents", "path", truncate(path, 100))

    cpath := C.CString(path)
    defer C.free(unsafe.Pointer(cpath))

    result := C.jj_get_file_contents((*C.RepoHandle)(repo), cpath)
    defer C.jj_free_result(result)

    if result.error != nil {
        errMsg := C.GoString(result.error)
        err := errors.New(errMsg)
        done(err)
        return nil, err
    }

    if result.data == nil {
        err := errors.New("no data returned")
        done(err)
        return nil, err
    }

    data := []byte(C.GoString(result.data))
    done(nil, "bytes", len(data))
    return data, nil
}
```

#### GetRevisionDiff
```go
func GetRevisionDiff(repo RepoPtr, revisionID string) (string, error) {
    done := logOpWithResult("GetRevisionDiff", "revision", truncate(revisionID, 12))

    crevID := C.CString(revisionID)
    defer C.free(unsafe.Pointer(crevID))

    result := C.jj_get_revision_diff((*C.RepoHandle)(repo), crevID)
    defer C.jj_free_result(result)

    if result.error != nil {
        errMsg := C.GoString(result.error)
        err := errors.New(errMsg)
        done(err)
        return "", err
    }

    if result.data == nil {
        done(nil, "bytes", 0)
        return "", nil
    }

    data := C.GoString(result.data)
    done(nil, "bytes", len(data))
    return data, nil
}
```

#### CloseRepo
```go
func CloseRepo(repo RepoPtr) {
    done := logOp("CloseRepo")

    if repo != nil {
        C.jj_close_repo((*C.RepoHandle)(repo))
    }
    done(nil)
}
```

#### SetBookmark
```go
func SetBookmark(repo RepoPtr, name, revisionID string, allowBackwards, ignoreImmutable bool) error {
    done := logOp("SetBookmark",
        "name", truncate(name, 50),
        "revision", truncate(revisionID, 12),
        "allowBackwards", allowBackwards,
        "ignoreImmutable", ignoreImmutable,
    )

    cname := C.CString(name)
    defer C.free(unsafe.Pointer(cname))

    crevID := C.CString(revisionID)
    defer C.free(unsafe.Pointer(crevID))

    var allowBackwardsInt, ignoreImmutableInt C.int
    if allowBackwards {
        allowBackwardsInt = 1
    }
    if ignoreImmutable {
        ignoreImmutableInt = 1
    }

    result := C.jj_set_bookmark((*C.RepoHandle)(repo), cname, crevID, allowBackwardsInt, ignoreImmutableInt)
    defer C.jj_free_result(result)

    if result.error != nil {
        errMsg := C.GoString(result.error)
        err := errors.New(errMsg)
        done(err)
        return err
    }

    done(nil)
    return nil
}
```

## Security Notes
- All paths are truncated to 100 chars
- Revision IDs truncated to 12 chars
- Bookmark names truncated to 50 chars
- No diff content is logged
- No file content is logged

## Time Estimate
45 minutes

## Acceptance Criteria
- [ ] All 12 FFI functions have logging
- [ ] Error paths log at ERROR level
- [ ] Success paths log at INFO level
- [ ] All logs include duration
- [ ] Paths and IDs are truncated for safety
- [ ] No content (diffs, file contents) is logged
- [ ] Build succeeds with no warnings
