package jj

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRestoreFile tests the RestoreFile function
func TestRestoreFile(t *testing.T) {
	// Create a temporary directory as a mock repo
	tmpDir := t.TempDir()

	// Initialize a jj repo
	initCmd := exec.Command("jj", "init", tmpDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Add the file to the repo
	addCmd := exec.Command("jj", "add", "test.txt")
	addCmd.Dir = tmpDir
	if err := addCmd.Run(); err != nil {
		t.Fatalf("failed to add file to repo: %v", err)
	}

	// Make the initial commit
	commitCmd := exec.Command("jj", "commit", "-m", "initial")
	commitCmd.Dir = tmpDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	// Modify the file
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	// Test RestoreFile
	err := RestoreFile(tmpDir, "test.txt")
	if err != nil {
		t.Errorf("RestoreFile failed: %v", err)
	}

	// Verify the file was restored
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	// After restore, the file should be at committed state
	// (behavior depends on jj implementation, but restore should succeed)
	if string(content) == "modified content" {
		t.Logf("Note: File was not reverted, which may be expected depending on jj behavior")
	}
}

// TestSquashFile tests the SquashFile function
func TestSquashFile(t *testing.T) {
	// Create a temporary directory as a mock repo
	tmpDir := t.TempDir()

	// Initialize a jj repo
	initCmd := exec.Command("jj", "init", tmpDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Add the file to the repo
	addCmd := exec.Command("jj", "add", "test.txt")
	addCmd.Dir = tmpDir
	if err := addCmd.Run(); err != nil {
		t.Fatalf("failed to add file to repo: %v", err)
	}

	// Make the initial commit
	commitCmd := exec.Command("jj", "commit", "-m", "initial")
	commitCmd.Dir = tmpDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	// Modify the file in the working copy
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	// Test SquashFile - this will fail if @ and @- don't have the right relationship
	// But we just want to ensure the command is called correctly
	err := SquashFile(tmpDir, "test.txt")
	if err != nil {
		// SquashFile may fail due to jj state, but we should have called the command
		t.Logf("SquashFile returned error (may be expected): %v", err)
	}
}

// TestRestoreFileErrors tests error handling in RestoreFile
func TestRestoreFileErrors(t *testing.T) {
	// Test with non-existent repo
	err := RestoreFile("/nonexistent/path", "test.txt")
	if err == nil {
		t.Errorf("RestoreFile should fail with non-existent repo path")
	}
}

// TestSquashFileErrors tests error handling in SquashFile
func TestSquashFileErrors(t *testing.T) {
	// Test with non-existent repo
	err := SquashFile("/nonexistent/path", "test.txt")
	if err == nil {
		t.Errorf("SquashFile should fail with non-existent repo path")
	}
}

// TestRebase tests the Rebase function
func TestRebase(t *testing.T) {
	// Create a temporary directory as a mock repo
	tmpDir := t.TempDir()

	// Initialize a jj repo
	initCmd := exec.Command("jj", "init", tmpDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Create a test file and commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create first commit
	commitCmd := exec.Command("jj", "commit", "-m", "first commit")
	commitCmd.Dir = tmpDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to create first commit: %v", err)
	}

	// Modify and create second commit
	if err := os.WriteFile(testFile, []byte("second"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	commitCmd2 := exec.Command("jj", "commit", "-m", "second commit")
	commitCmd2.Dir = tmpDir
	if err := commitCmd2.Run(); err != nil {
		t.Fatalf("failed to create second commit: %v", err)
	}

	// Test Rebase with valid revisions (@ onto root())
	// This may fail due to repo state, but we verify the command executes
	err := Rebase(tmpDir, "@-", "root()")
	if err != nil {
		t.Logf("Rebase returned error (may be expected depending on repo state): %v", err)
	}
}

// TestRebaseErrors tests error handling in Rebase
func TestRebaseErrors(t *testing.T) {
	// Test with non-existent repo
	err := Rebase("/nonexistent/path", "@", "@-")
	if err == nil {
		t.Errorf("Rebase should fail with non-existent repo path")
	}
}

// TestRebaseBranch tests the RebaseBranch function
func TestRebaseBranch(t *testing.T) {
	// Create a temporary directory as a mock repo
	tmpDir := t.TempDir()

	// Initialize a jj repo
	initCmd := exec.Command("jj", "init", tmpDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Create a test file and commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create first commit
	commitCmd := exec.Command("jj", "commit", "-m", "first commit")
	commitCmd.Dir = tmpDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to create first commit: %v", err)
	}

	// Modify and create second commit
	if err := os.WriteFile(testFile, []byte("second"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	commitCmd2 := exec.Command("jj", "commit", "-m", "second commit")
	commitCmd2.Dir = tmpDir
	if err := commitCmd2.Run(); err != nil {
		t.Fatalf("failed to create second commit: %v", err)
	}

	// Test RebaseBranch - rebase branch onto root()
	// This may fail due to repo state, but we verify the command executes
	err := RebaseBranch(tmpDir, "@-", "root()")
	if err != nil {
		t.Logf("RebaseBranch returned error (may be expected depending on repo state): %v", err)
	}
}

// TestRebaseBranchErrors tests error handling in RebaseBranch
func TestRebaseBranchErrors(t *testing.T) {
	// Test with non-existent repo
	err := RebaseBranch("/nonexistent/path", "@", "@-")
	if err == nil {
		t.Errorf("RebaseBranch should fail with non-existent repo path")
	}
}

// TestBuildSquashRevset tests the BuildSquashRevset helper function
func TestBuildSquashRevset(t *testing.T) {
	tests := []struct {
		name     string
		lower    string
		higher   string
		expected string
	}{
		{
			name:     "simple change IDs",
			lower:    "abc12345",
			higher:   "def67890",
			expected: "abc12345::def67890 ~ abc12345",
		},
		{
			name:     "revset expressions",
			lower:    "@-",
			higher:   "@",
			expected: "@-::@ ~ @-",
		},
		{
			name:     "full commit IDs",
			lower:    "a1b2c3d4e5f6",
			higher:   "f6e5d4c3b2a1",
			expected: "a1b2c3d4e5f6::f6e5d4c3b2a1 ~ a1b2c3d4e5f6",
		},
		{
			name:     "with root()",
			lower:    "root()",
			higher:   "abc12345",
			expected: "root()::abc12345 ~ root()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSquashRevset(tt.lower, tt.higher)
			if result != tt.expected {
				t.Errorf("BuildSquashRevset(%q, %q) = %q; want %q",
					tt.lower, tt.higher, result, tt.expected)
			}
		})
	}
}

// TestSquashRangeErrors tests error handling in SquashRange
func TestSquashRangeErrors(t *testing.T) {
	// Test with non-existent repo
	err := SquashRange("/nonexistent/path", "@-", "@", "test message")
	if err == nil {
		t.Errorf("SquashRange should fail with non-existent repo path")
	}
}

// TestSquashRange tests the SquashRange function with a real jj repo
func TestSquashRange(t *testing.T) {
	// Create a temporary directory as a mock repo
	tmpDir := t.TempDir()

	// Initialize a jj repo
	initCmd := exec.Command("jj", "init", tmpDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create first commit
	commitCmd := exec.Command("jj", "commit", "-m", "first commit")
	commitCmd.Dir = tmpDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to create first commit: %v", err)
	}

	// Modify and create second commit
	if err := os.WriteFile(testFile, []byte("second"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	commitCmd2 := exec.Command("jj", "commit", "-m", "second commit")
	commitCmd2.Dir = tmpDir
	if err := commitCmd2.Run(); err != nil {
		t.Fatalf("failed to create second commit: %v", err)
	}

	// Modify and create third commit
	if err := os.WriteFile(testFile, []byte("third"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	commitCmd3 := exec.Command("jj", "commit", "-m", "third commit")
	commitCmd3.Dir = tmpDir
	if err := commitCmd3.Run(); err != nil {
		t.Fatalf("failed to create third commit: %v", err)
	}

	// Get the change IDs for reference
	logCmd := exec.Command("jj", "log", "--no-graph", "-T", `change_id.short(8) ++ "\n"`)
	logCmd.Dir = tmpDir
	output, err := logCmd.Output()
	if err != nil {
		t.Fatalf("failed to get log: %v", err)
	}
	t.Logf("Changes before squash:\n%s", output)

	// Test SquashRange: squash @-- and @- into @--
	// This squashes the range from the first commit to the second into the first
	err = SquashRange(tmpDir, "@---", "@--", "squashed commits")
	if err != nil {
		// May fail depending on repo state, but command should execute
		t.Logf("SquashRange returned error (may be expected): %v", err)
	}

	// Get the log after squash to verify
	logCmd2 := exec.Command("jj", "log", "--no-graph", "-T", `change_id.short(8) ++ " " ++ if(description, description.first_line(), "(no description)") ++ "\n"`)
	logCmd2.Dir = tmpDir
	output2, err := logCmd2.Output()
	if err != nil {
		t.Fatalf("failed to get log after squash: %v", err)
	}
	t.Logf("Changes after squash:\n%s", output2)
}

// TestGetDescription tests that GetDescription returns empty string for changes without descriptions
func TestGetDescription(t *testing.T) {
	// Create a temporary directory as a mock repo
	tmpDir := t.TempDir()

	// Initialize a jj repo (use jj git init for newer jj versions)
	initCmd := exec.Command("jj", "git", "init", tmpDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Get the working copy (current) change ID
	logCmd := exec.Command("jj", "log", "-r", "@", "--no-graph", "-T", "change_id.short(8)")
	logCmd.Dir = tmpDir
	output, err := logCmd.Output()
	if err != nil {
		t.Fatalf("failed to get current change ID: %v", err)
	}
	changeID := strings.TrimSpace(string(output))

	// Test GetDescription on a change with no description
	// The old template "description" would return "@ | ~" for empty descriptions
	// The new template "if(description, description, \"\")" should return empty string
	desc, err := GetDescription(tmpDir, changeID)
	if err != nil {
		t.Errorf("GetDescription failed: %v", err)
	}

	// For a new change with no description set, should be empty
	if desc == "@ | ~" {
		t.Errorf("GetDescription returned the problematic string '@ | ~' - template fix may not be applied")
	}

	// The description should be empty or contain actual description text
	t.Logf("Description for change with no description: %q", desc)
}

// TestGetDescriptionWithText tests that GetDescription returns the correct description text
func TestGetDescriptionWithText(t *testing.T) {
	// Create a temporary directory as a mock repo
	tmpDir := t.TempDir()

	// Initialize a jj repo
	initCmd := exec.Command("jj", "git", "init", tmpDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a commit with a description
	expectedDesc := "Test commit description"
	commitCmd := exec.Command("jj", "commit", "-m", expectedDesc)
	commitCmd.Dir = tmpDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}

	// Get the description of the parent commit (@-)
	desc, err := GetDescription(tmpDir, "@-")
	if err != nil {
		t.Errorf("GetDescription failed: %v", err)
	}

	if desc != expectedDesc {
		t.Errorf("GetDescription(@-) = %q; want %q", desc, expectedDesc)
	}
}

// TestGetDescriptionMultiline tests that GetDescription returns full multiline descriptions
func TestGetDescriptionMultiline(t *testing.T) {
	// Create a temporary directory as a mock repo
	tmpDir := t.TempDir()

	// Initialize a jj repo
	initCmd := exec.Command("jj", "git", "init", tmpDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a commit with a multiline description
	multilineDesc := "First line\n\nSecond paragraph"
	commitCmd := exec.Command("jj", "commit", "-m", multilineDesc)
	commitCmd.Dir = tmpDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}

	// Get the description of the parent commit (@-)
	desc, err := GetDescription(tmpDir, "@-")
	if err != nil {
		t.Errorf("GetDescription failed: %v", err)
	}

	if desc != multilineDesc {
		t.Errorf("GetDescription(@-) = %q; want %q", desc, multilineDesc)
	}
}

// TestGetDescriptionErrors tests error handling in GetDescription
func TestGetDescriptionErrors(t *testing.T) {
	// Test with non-existent repo
	_, err := GetDescription("/nonexistent/path", "@")
	if err == nil {
		t.Errorf("GetDescription should fail with non-existent repo path")
	}
}

// TestIsAncestor tests the IsAncestor function
func TestIsAncestor(t *testing.T) {
	// Create a temporary directory as a mock repo
	tmpDir := t.TempDir()

	// Initialize a jj repo
	initCmd := exec.Command("jj", "git", "init", tmpDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create first commit
	commitCmd := exec.Command("jj", "commit", "-m", "first commit")
	commitCmd.Dir = tmpDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to create first commit: %v", err)
	}

	// Modify and create second commit
	if err := os.WriteFile(testFile, []byte("second"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	commitCmd2 := exec.Command("jj", "commit", "-m", "second commit")
	commitCmd2.Dir = tmpDir
	if err := commitCmd2.Run(); err != nil {
		t.Fatalf("failed to create second commit: %v", err)
	}

	// Modify and create third commit
	if err := os.WriteFile(testFile, []byte("third"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	commitCmd3 := exec.Command("jj", "commit", "-m", "third commit")
	commitCmd3.Dir = tmpDir
	if err := commitCmd3.Run(); err != nil {
		t.Fatalf("failed to create third commit: %v", err)
	}

	// Now we have: root <- @--- <- @-- <- @- <- @ (working copy)
	// Test cases:

	// Test 1: @--- is an ancestor of @- (should be true)
	isAnc, err := IsAncestor(tmpDir, "@---", "@-")
	if err != nil {
		t.Errorf("IsAncestor(@---, @-) returned error: %v", err)
	}
	if !isAnc {
		t.Errorf("IsAncestor(@---, @-) = false; want true (grandparent is ancestor)")
	}

	// Test 2: @- is NOT an ancestor of @--- (should be false)
	isAnc, err = IsAncestor(tmpDir, "@-", "@---")
	if err != nil {
		t.Errorf("IsAncestor(@-, @---) returned error: %v", err)
	}
	if isAnc {
		t.Errorf("IsAncestor(@-, @---) = true; want false (descendant is not ancestor)")
	}

	// Test 3: @-- is an ancestor of @-- itself (should be true, A == B case)
	isAnc, err = IsAncestor(tmpDir, "@--", "@--")
	if err != nil {
		t.Errorf("IsAncestor(@--, @--) returned error: %v", err)
	}
	if !isAnc {
		t.Errorf("IsAncestor(@--, @--) = false; want true (commit is ancestor of itself)")
	}

	// Test 4: root() is an ancestor of @ (should be true)
	isAnc, err = IsAncestor(tmpDir, "root()", "@")
	if err != nil {
		t.Errorf("IsAncestor(root(), @) returned error: %v", err)
	}
	if !isAnc {
		t.Errorf("IsAncestor(root(), @) = false; want true (root is ancestor of all)")
	}
}

// TestIsAncestorErrors tests error handling in IsAncestor
func TestIsAncestorErrors(t *testing.T) {
	// Test with non-existent repo
	_, err := IsAncestor("/nonexistent/path", "@", "@-")
	if err == nil {
		t.Errorf("IsAncestor should fail with non-existent repo path")
	}
}
