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
