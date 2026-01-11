package ffi

/*
#cgo LDFLAGS: ${SRCDIR}/../../../rust/target/release/libjjbridge.a
#cgo LDFLAGS: -framework CoreFoundation -framework Security -framework SystemConfiguration -liconv -lresolv
#include "bridge.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"strings"
	"unsafe"
)

// RepoPtr is an opaque pointer to a Rust repository handle
type RepoPtr unsafe.Pointer

// OpenRepo opens a jj repository at the given path
// Returns nil and an error if the repo cannot be opened
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

// ListBranches returns JSON-encoded branch data from the repository
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
	done(nil, "bytes", len(data))
	return data, nil
}

// ListWorkspaces returns JSON-encoded workspace data from the repository
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

// GetWorkingCopyChanges returns JSON-encoded file change data from the repository
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

// ListOperations returns JSON-encoded operation data from the repository
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

// GetLog returns JSON-encoded revision log data from the repository
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

// GetDiff returns the unified diff for the working copy
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

// GetFileDiff returns the unified diff for a specific file in the working copy
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

// GetFileContents returns JSON-encoded before/after file contents
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

// GetRevisionDiff returns the unified diff for a revision compared to its parent
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

// CloseRepo closes a repository handle
func CloseRepo(repo RepoPtr) {
	done := logOp("CloseRepo")

	if repo != nil {
		C.jj_close_repo((*C.RepoHandle)(repo))
	}
	done(nil)
}

// SetBookmark sets a bookmark to point to a specific revision
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

// WorkspaceAdd creates a new workspace at the given path.
// revisionIDs specifies parent commits for the new workspace's working copy.
// If empty, defaults to parent(s) of current workspace's working copy.
func WorkspaceAdd(repo RepoPtr, destinationPath, workspaceName string, revisionIDs []string) error {
	done := logOp("WorkspaceAdd",
		"destinationPath", truncate(destinationPath, 100),
		"workspaceName", truncate(workspaceName, 50),
		"revisionIDs", revisionIDs,
	)

	cDestPath := C.CString(destinationPath)
	defer C.free(unsafe.Pointer(cDestPath))

	var cWsName *C.char
	if workspaceName != "" {
		cWsName = C.CString(workspaceName)
		defer C.free(unsafe.Pointer(cWsName))
	}

	// Convert revision IDs to comma-separated string
	var cRevIDs *C.char
	if len(revisionIDs) > 0 {
		revIDStr := strings.Join(revisionIDs, ",")
		cRevIDs = C.CString(revIDStr)
		defer C.free(unsafe.Pointer(cRevIDs))
	}

	result := C.jj_workspace_add((*C.RepoHandle)(repo), cDestPath, cWsName, cRevIDs)
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

// WorkspaceForget removes workspace tracking (keeps files on disk)
func WorkspaceForget(repo RepoPtr, workspaceName string) error {
	done := logOp("WorkspaceForget",
		"workspaceName", truncate(workspaceName, 50),
	)

	cWsName := C.CString(workspaceName)
	defer C.free(unsafe.Pointer(cWsName))

	result := C.jj_workspace_forget((*C.RepoHandle)(repo), cWsName)
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
