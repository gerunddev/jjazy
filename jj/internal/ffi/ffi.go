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
	"unsafe"
)

// RepoPtr is an opaque pointer to a Rust repository handle
type RepoPtr unsafe.Pointer

// OpenRepo opens a jj repository at the given path
// Returns nil and an error if the repo cannot be opened
func OpenRepo(path string) (RepoPtr, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	handle := C.jj_open_repo(cpath)
	if handle == nil {
		return nil, errors.New("failed to open repository")
	}
	return RepoPtr(handle), nil
}

// ListBranches returns JSON-encoded branch data from the repository
func ListBranches(repo RepoPtr) ([]byte, error) {
	result := C.jj_list_branches((*C.RepoHandle)(repo))
	defer C.jj_free_result(result)

	if result.error != nil {
		errMsg := C.GoString(result.error)
		return nil, errors.New(errMsg)
	}

	if result.data == nil {
		return nil, errors.New("no data returned")
	}

	return []byte(C.GoString(result.data)), nil
}

// ListWorkspaces returns JSON-encoded workspace data from the repository
func ListWorkspaces(repo RepoPtr) ([]byte, error) {
	result := C.jj_list_workspaces((*C.RepoHandle)(repo))
	defer C.jj_free_result(result)

	if result.error != nil {
		errMsg := C.GoString(result.error)
		return nil, errors.New(errMsg)
	}

	if result.data == nil {
		return nil, errors.New("no data returned")
	}

	return []byte(C.GoString(result.data)), nil
}

// GetWorkingCopyChanges returns JSON-encoded file change data from the repository
func GetWorkingCopyChanges(repo RepoPtr) ([]byte, error) {
	result := C.jj_get_working_copy_changes((*C.RepoHandle)(repo))
	defer C.jj_free_result(result)

	if result.error != nil {
		errMsg := C.GoString(result.error)
		return nil, errors.New(errMsg)
	}

	if result.data == nil {
		return nil, errors.New("no data returned")
	}

	return []byte(C.GoString(result.data)), nil
}

// ListOperations returns JSON-encoded operation data from the repository
func ListOperations(repo RepoPtr) ([]byte, error) {
	result := C.jj_list_operations((*C.RepoHandle)(repo))
	defer C.jj_free_result(result)

	if result.error != nil {
		errMsg := C.GoString(result.error)
		return nil, errors.New(errMsg)
	}

	if result.data == nil {
		return nil, errors.New("no data returned")
	}

	return []byte(C.GoString(result.data)), nil
}

// GetLog returns JSON-encoded revision log data from the repository
func GetLog(repo RepoPtr) ([]byte, error) {
	result := C.jj_get_log((*C.RepoHandle)(repo))
	defer C.jj_free_result(result)

	if result.error != nil {
		errMsg := C.GoString(result.error)
		return nil, errors.New(errMsg)
	}

	if result.data == nil {
		return nil, errors.New("no data returned")
	}

	return []byte(C.GoString(result.data)), nil
}

// GetDiff returns the unified diff for the working copy
func GetDiff(repo RepoPtr) (string, error) {
	result := C.jj_get_diff((*C.RepoHandle)(repo))
	defer C.jj_free_result(result)

	if result.error != nil {
		errMsg := C.GoString(result.error)
		return "", errors.New(errMsg)
	}

	if result.data == nil {
		return "", nil
	}

	return C.GoString(result.data), nil
}

// GetFileDiff returns the unified diff for a specific file in the working copy
func GetFileDiff(repo RepoPtr, path string) (string, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	result := C.jj_get_file_diff((*C.RepoHandle)(repo), cpath)
	defer C.jj_free_result(result)

	if result.error != nil {
		errMsg := C.GoString(result.error)
		return "", errors.New(errMsg)
	}

	if result.data == nil {
		return "", nil
	}

	return C.GoString(result.data), nil
}

// GetFileContents returns JSON-encoded before/after file contents
func GetFileContents(repo RepoPtr, path string) ([]byte, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	result := C.jj_get_file_contents((*C.RepoHandle)(repo), cpath)
	defer C.jj_free_result(result)

	if result.error != nil {
		errMsg := C.GoString(result.error)
		return nil, errors.New(errMsg)
	}

	if result.data == nil {
		return nil, errors.New("no data returned")
	}

	return []byte(C.GoString(result.data)), nil
}

// GetRevisionDiff returns the unified diff for a revision compared to its parent
func GetRevisionDiff(repo RepoPtr, revisionID string) (string, error) {
	crevID := C.CString(revisionID)
	defer C.free(unsafe.Pointer(crevID))

	result := C.jj_get_revision_diff((*C.RepoHandle)(repo), crevID)
	defer C.jj_free_result(result)

	if result.error != nil {
		errMsg := C.GoString(result.error)
		return "", errors.New(errMsg)
	}

	if result.data == nil {
		return "", nil
	}

	return C.GoString(result.data), nil
}

// CloseRepo closes a repository handle
func CloseRepo(repo RepoPtr) {
	if repo != nil {
		C.jj_close_repo((*C.RepoHandle)(repo))
	}
}
