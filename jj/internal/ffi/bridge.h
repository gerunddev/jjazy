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

// Close a repository handle and free its memory
void jj_close_repo(RepoHandle* handle);

// Free a JjResult's memory
void jj_free_result(JjResult result);

// Free a string allocated by Rust
void jj_free_string(char* s);

#endif // JJ_BRIDGE_H
