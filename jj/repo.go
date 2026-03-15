package jj

import (
	"encoding/json"
	"sync"

	"github.com/gerunddev/jjazy/jj/internal/ffi"
)

// Repo represents an open jj repository.
type Repo struct {
	mu  sync.Mutex
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
	r.mu.Lock()
	data, err := ffi.ListBranches(r.ptr)
	r.mu.Unlock()
	if err != nil {
		return nil, err
	}

	var branches []Branch
	if err := json.Unmarshal(data, &branches); err != nil {
		return nil, err
	}
	return branches, nil
}

// Workspaces returns a list of workspaces in the repository.
func (r *Repo) Workspaces() ([]Workspace, error) {
	r.mu.Lock()
	data, err := ffi.ListWorkspaces(r.ptr)
	r.mu.Unlock()
	if err != nil {
		return nil, err
	}

	var workspaces []Workspace
	if err := json.Unmarshal(data, &workspaces); err != nil {
		return nil, err
	}
	return workspaces, nil
}

// WorkingCopyChanges returns a list of changed files in the working copy.
func (r *Repo) WorkingCopyChanges() ([]FileChange, error) {
	r.mu.Lock()
	data, err := ffi.GetWorkingCopyChanges(r.ptr)
	r.mu.Unlock()
	if err != nil {
		return nil, err
	}

	var changes []FileChange
	if err := json.Unmarshal(data, &changes); err != nil {
		return nil, err
	}
	return changes, nil
}

// Operations returns a list of operations in the undo history.
func (r *Repo) Operations() ([]Operation, error) {
	r.mu.Lock()
	data, err := ffi.ListOperations(r.ptr)
	r.mu.Unlock()
	if err != nil {
		return nil, err
	}

	var operations []Operation
	if err := json.Unmarshal(data, &operations); err != nil {
		return nil, err
	}
	return operations, nil
}

// Log returns the revision log.
func (r *Repo) Log() ([]Revision, error) {
	r.mu.Lock()
	data, err := ffi.GetLog(r.ptr)
	r.mu.Unlock()
	if err != nil {
		return nil, err
	}

	var revisions []Revision
	if err := json.Unmarshal(data, &revisions); err != nil {
		return nil, err
	}
	return revisions, nil
}

// Diff returns the unified diff for the working copy.
func (r *Repo) Diff() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return ffi.GetDiff(r.ptr)
}

// FileDiff returns the unified diff for a specific file in the working copy.
func (r *Repo) FileDiff(path string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return ffi.GetFileDiff(r.ptr, path)
}

// FileContents returns the before/after contents of a specific file.
func (r *Repo) FileContents(path string) (*FileContents, error) {
	r.mu.Lock()
	data, err := ffi.GetFileContents(r.ptr, path)
	r.mu.Unlock()
	if err != nil {
		return nil, err
	}

	var contents FileContents
	if err := json.Unmarshal(data, &contents); err != nil {
		return nil, err
	}
	return &contents, nil
}

// RevisionDiff returns the unified diff for a revision compared to its parent.
func (r *Repo) RevisionDiff(revisionID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return ffi.GetRevisionDiff(r.ptr, revisionID)
}

// SetBookmark sets a bookmark to point to a specific revision.
// If allowBackwards is true, the bookmark can be moved to an ancestor.
// If ignoreImmutable is true, the bookmark can be set on immutable revisions.
func (r *Repo) SetBookmark(name, revisionID string, allowBackwards, ignoreImmutable bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return ffi.SetBookmark(r.ptr, name, revisionID, allowBackwards, ignoreImmutable)
}

// WorkspaceAdd creates a new workspace at the given path.
// If workspaceName is empty, it will be derived from the path basename.
// If revisionIDs is empty, the new workspace starts from the same parent(s)
// as the current workspace's working copy (siblings).
// If revisionIDs is provided, the new workspace starts on top of those revisions.
func (r *Repo) WorkspaceAdd(destinationPath, workspaceName string, revisionIDs ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return ffi.WorkspaceAdd(r.ptr, destinationPath, workspaceName, revisionIDs)
}

// WorkspaceForget removes workspace tracking (keeps files on disk).
func (r *Repo) WorkspaceForget(workspaceName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return ffi.WorkspaceForget(r.ptr, workspaceName)
}

// GitPush pushes a bookmark to the "origin" remote via git.
// If allowBackwards is true, allows force-push even if non-fast-forward.
func (r *Repo) GitPush(bookmarkName string, allowBackwards bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return ffi.GitPush(r.ptr, bookmarkName, allowBackwards)
}

// Close closes the repository and frees associated resources.
func (r *Repo) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ptr != nil {
		ffi.CloseRepo(r.ptr)
		r.ptr = nil
	}
}
