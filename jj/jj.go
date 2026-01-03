// Package jj provides a Go interface to jj (Jujutsu) repositories.
// This package wraps the jj-lib Rust library via CGO/FFI.
// All FFI details are hidden - consumers of this package interact
// with pure Go types.
package jj

// Branch represents a branch (bookmark) in a jj repository.
type Branch struct {
	Name    string `json:"name"`
	IsLocal bool   `json:"is_local"`
}

// Workspace represents a jj workspace.
type Workspace struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current"`
	CommitID  string `json:"commit_id"`
}

// FileChange represents a changed file in the working copy.
type FileChange struct {
	Path   string `json:"path"`
	Status string `json:"status"` // "modified", "added", "deleted"
}

// Operation represents an operation in the undo history.
type Operation struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Timestamp   string `json:"timestamp"`
	IsCurrent   bool   `json:"is_current"`
}

// Revision represents a commit/revision in the log.
type Revision struct {
	ID            string   `json:"id"`
	ChangeID      string   `json:"change_id"`
	Description   string   `json:"description"`
	Author        string   `json:"author"`
	Timestamp     string   `json:"timestamp"`
	IsWorkingCopy bool     `json:"is_working_copy"`
	WorkspaceName *string  `json:"workspace_name"`
	IsRoot        bool     `json:"is_root"`
	Parents       []string `json:"parents"`
}
