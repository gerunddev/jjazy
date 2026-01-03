package jj

import (
	"encoding/json"

	"github.com/gerund/jayz/jj/internal/ffi"
)

// Repo represents an open jj repository.
type Repo struct {
	ptr ffi.RepoPtr
}

// Open opens a jj repository at the given path.
func Open(path string) (*Repo, error) {
	ptr, err := ffi.OpenRepo(path)
	if err != nil {
		return nil, err
	}
	return &Repo{ptr: ptr}, nil
}

// Branches returns a list of branches (bookmarks) in the repository.
func (r *Repo) Branches() ([]Branch, error) {
	data, err := ffi.ListBranches(r.ptr)
	if err != nil {
		return nil, err
	}

	var branches []Branch
	if err := json.Unmarshal(data, &branches); err != nil {
		return nil, err
	}
	return branches, nil
}

// Close closes the repository and frees associated resources.
func (r *Repo) Close() {
	if r.ptr != nil {
		ffi.CloseRepo(r.ptr)
		r.ptr = nil
	}
}
