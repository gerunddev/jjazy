package jj

import (
	"encoding/json"

	"github.com/gerund/jjazy/jj/internal/ffi"
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

// Workspaces returns a list of workspaces in the repository.
func (r *Repo) Workspaces() ([]Workspace, error) {
	data, err := ffi.ListWorkspaces(r.ptr)
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
	data, err := ffi.GetWorkingCopyChanges(r.ptr)
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
	data, err := ffi.ListOperations(r.ptr)
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
	data, err := ffi.GetLog(r.ptr)
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
	return ffi.GetDiff(r.ptr)
}

// FileDiff returns the unified diff for a specific file in the working copy.
func (r *Repo) FileDiff(path string) (string, error) {
	return ffi.GetFileDiff(r.ptr, path)
}

// FileContents returns the before/after contents of a specific file.
func (r *Repo) FileContents(path string) (*FileContents, error) {
	data, err := ffi.GetFileContents(r.ptr, path)
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
	return ffi.GetRevisionDiff(r.ptr, revisionID)
}

// SetBookmark sets a bookmark to point to a specific revision.
// If allowBackwards is true, the bookmark can be moved to an ancestor.
// If ignoreImmutable is true, the bookmark can be set on immutable revisions.
func (r *Repo) SetBookmark(name, revisionID string, allowBackwards, ignoreImmutable bool) error {
	return ffi.SetBookmark(r.ptr, name, revisionID, allowBackwards, ignoreImmutable)
}

// WorkspaceAdd creates a new workspace at the given path.
// If workspaceName is empty, it will be derived from the path basename.
// If revisionIDs is empty, the new workspace starts from the same parent(s)
// as the current workspace's working copy (siblings).
// If revisionIDs is provided, the new workspace starts on top of those revisions.
func (r *Repo) WorkspaceAdd(destinationPath, workspaceName string, revisionIDs ...string) error {
	return ffi.WorkspaceAdd(r.ptr, destinationPath, workspaceName, revisionIDs)
}

// WorkspaceForget removes workspace tracking (keeps files on disk).
func (r *Repo) WorkspaceForget(workspaceName string) error {
	return ffi.WorkspaceForget(r.ptr, workspaceName)
}

// Close closes the repository and frees associated resources.
func (r *Repo) Close() {
	if r.ptr != nil {
		ffi.CloseRepo(r.ptr)
		r.ptr = nil
	}
}
