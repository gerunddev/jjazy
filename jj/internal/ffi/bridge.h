#ifndef JJ_BRIDGE_H
#define JJ_BRIDGE_H

#include <stdint.h>

// Opaque handle to a jj repository
typedef struct RepoHandle RepoHandle;

// Result structure for FFI calls
typedef struct {
    char* data;   // JSON data on success, NULL on error
    char* error;  // Error message on failure, NULL on success
} JjResult;

// Open a jj repository at the given path
// Returns NULL on error
RepoHandle* jj_open_repo(const char* path);

// List branches in the repository
// Returns JjResult with JSON array of branch info
JjResult jj_list_branches(RepoHandle* handle);

// List workspaces in the repository
// Returns JjResult with JSON array of workspace info
JjResult jj_list_workspaces(RepoHandle* handle);

// Get working copy file changes
// Returns JjResult with JSON array of file change info
JjResult jj_get_working_copy_changes(RepoHandle* handle);

// List operations in the repository
// Returns JjResult with JSON array of operation info
JjResult jj_list_operations(RepoHandle* handle);

// Get revision log
// Returns JjResult with JSON array of revision info
JjResult jj_get_log(RepoHandle* handle);

// Get diff for working copy
// Returns JjResult with unified diff string
JjResult jj_get_diff(RepoHandle* handle);

// Get diff for a specific file in working copy
// Returns JjResult with unified diff string
JjResult jj_get_file_diff(RepoHandle* handle, const char* path);

// Get before/after file contents
// Returns JjResult with JSON containing before and after content
JjResult jj_get_file_contents(RepoHandle* handle, const char* path);

// Get diff for a revision compared to its parent
// Returns JjResult with unified diff string
JjResult jj_get_revision_diff(RepoHandle* handle, const char* revision_id);

// Close a repository handle and free its memory
void jj_close_repo(RepoHandle* handle);

// Free a JjResult's memory
void jj_free_result(JjResult result);

// Free a string allocated by Rust
void jj_free_string(char* s);

// Set a bookmark to point to a specific revision
// Parameters:
// - handle: repository handle
// - name: bookmark name
// - revision_id: target revision ID prefix
// - allow_backwards: if true, allow moving bookmark backwards in history
// - ignore_immutable: if true, allow setting bookmark on immutable revisions
// Returns JjResult with empty success or error message
JjResult jj_set_bookmark(RepoHandle* handle, const char* name, const char* revision_id,
                         int allow_backwards, int ignore_immutable);

#endif // JJ_BRIDGE_H
