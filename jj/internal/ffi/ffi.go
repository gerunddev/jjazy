package ffi

/*
#cgo LDFLAGS: -L${SRCDIR}/../../../rust/target/release -ljjbridge
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

// CloseRepo closes a repository handle
func CloseRepo(repo RepoPtr) {
	if repo != nil {
		C.jj_close_repo((*C.RepoHandle)(repo))
	}
}
