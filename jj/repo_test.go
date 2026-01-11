package jj

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestWorkspaceAddDefaultBehavior tests that creating a workspace without revision IDs
// creates a sibling of the current workspace's working copy (same parents).
func TestWorkspaceAddDefaultBehavior(t *testing.T) {
	// Create a temporary directory for test repos
	tmpDir := t.TempDir()
	mainRepoDir := filepath.Join(tmpDir, "main")

	// Initialize main jj repo
	initCmd := exec.Command("jj", "git", "init", mainRepoDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Create a test file to have some history
	testFile := filepath.Join(mainRepoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Commit the initial file
	commitCmd := exec.Command("jj", "commit", "-m", "initial commit")
	commitCmd.Dir = mainRepoDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	// Get the parent commit ID of current working copy (should be the commit we just made)
	logCmd := exec.Command("jj", "log", "-r", "@-", "-T", "commit_id.short(12)", "--no-graph")
	logCmd.Dir = mainRepoDir
	parentOutput, err := logCmd.Output()
	if err != nil {
		t.Fatalf("failed to get parent commit ID: %v", err)
	}
	parentCommitID := strings.TrimSpace(string(parentOutput))

	// Open repository via FFI
	repo, err := Open(mainRepoDir)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}
	defer repo.Close()

	// Create a new workspace with default behavior (no revision IDs)
	newWsPath := filepath.Join(tmpDir, "workspace2")
	err = repo.WorkspaceAdd(newWsPath, "workspace2")
	if err != nil {
		t.Fatalf("WorkspaceAdd failed: %v", err)
	}

	// Verify workspace was created
	if _, err := os.Stat(newWsPath); os.IsNotExist(err) {
		t.Errorf("workspace directory was not created")
	}

	// Verify workspace appears in list
	workspaces, err := repo.Workspaces()
	if err != nil {
		t.Fatalf("failed to list workspaces: %v", err)
	}

	var found bool
	for _, ws := range workspaces {
		if ws.Name == "workspace2" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("new workspace not found in workspace list")
	}

	// Verify the new workspace's working copy has the same parent as the original
	// by checking the log shows the workspace at the right position
	logCmd2 := exec.Command("jj", "log", "-r", "workspace2@", "-T", "parents.map(|c| c.commit_id().short(12)).join(\",\")", "--no-graph")
	logCmd2.Dir = mainRepoDir
	ws2ParentOutput, err := logCmd2.Output()
	if err != nil {
		t.Fatalf("failed to get workspace2 parent: %v", err)
	}
	ws2Parent := strings.TrimSpace(string(ws2ParentOutput))

	if ws2Parent != parentCommitID {
		t.Errorf("new workspace parent mismatch: got %s, want %s", ws2Parent, parentCommitID)
	}
}

// TestWorkspaceAddExplicitRevision tests creating a workspace with a specific revision.
func TestWorkspaceAddExplicitRevision(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepoDir := filepath.Join(tmpDir, "main")

	// Initialize main jj repo
	initCmd := exec.Command("jj", "git", "init", mainRepoDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Create first commit
	testFile := filepath.Join(mainRepoDir, "file1.txt")
	if err := os.WriteFile(testFile, []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	commitCmd := exec.Command("jj", "commit", "-m", "first commit")
	commitCmd.Dir = mainRepoDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to create first commit: %v", err)
	}

	// Get the first commit ID (we want to use this as explicit revision)
	logCmd := exec.Command("jj", "log", "-r", "@-", "-T", "commit_id.short(12)", "--no-graph")
	logCmd.Dir = mainRepoDir
	firstCommitOutput, err := logCmd.Output()
	if err != nil {
		t.Fatalf("failed to get first commit ID: %v", err)
	}
	firstCommitID := strings.TrimSpace(string(firstCommitOutput))

	// Create second commit
	testFile2 := filepath.Join(mainRepoDir, "file2.txt")
	if err := os.WriteFile(testFile2, []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	commitCmd2 := exec.Command("jj", "commit", "-m", "second commit")
	commitCmd2.Dir = mainRepoDir
	if err := commitCmd2.Run(); err != nil {
		t.Fatalf("failed to create second commit: %v", err)
	}

	// Get the second commit ID
	logCmd2 := exec.Command("jj", "log", "-r", "@-", "-T", "commit_id.short(12)", "--no-graph")
	logCmd2.Dir = mainRepoDir
	secondCommitOutput, err := logCmd2.Output()
	if err != nil {
		t.Fatalf("failed to get second commit ID: %v", err)
	}
	secondCommitID := strings.TrimSpace(string(secondCommitOutput))

	// Open repository via FFI
	repo, err := Open(mainRepoDir)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}
	defer repo.Close()

	// Create workspace at the FIRST commit (not the current parent)
	newWsPath := filepath.Join(tmpDir, "workspace2")
	err = repo.WorkspaceAdd(newWsPath, "workspace2", firstCommitID)
	if err != nil {
		t.Fatalf("WorkspaceAdd with explicit revision failed: %v", err)
	}

	// Verify the new workspace's parent is the first commit, not the second
	logCmd3 := exec.Command("jj", "log", "-r", "workspace2@", "-T", "parents.map(|c| c.commit_id().short(12)).join(\",\")", "--no-graph")
	logCmd3.Dir = mainRepoDir
	ws2ParentOutput, err := logCmd3.Output()
	if err != nil {
		t.Fatalf("failed to get workspace2 parent: %v", err)
	}
	ws2Parent := strings.TrimSpace(string(ws2ParentOutput))

	if ws2Parent != firstCommitID {
		t.Errorf("workspace parent should be first commit %s, got %s (second commit was %s)",
			firstCommitID, ws2Parent, secondCommitID)
	}
}

// TestWorkspaceAddInvalidRevision tests error handling for invalid revision IDs.
func TestWorkspaceAddInvalidRevision(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepoDir := filepath.Join(tmpDir, "main")

	// Initialize main jj repo
	initCmd := exec.Command("jj", "git", "init", mainRepoDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Open repository via FFI
	repo, err := Open(mainRepoDir)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}
	defer repo.Close()

	// Try to create workspace with invalid revision ID
	newWsPath := filepath.Join(tmpDir, "workspace2")
	err = repo.WorkspaceAdd(newWsPath, "workspace2", "invalid_revision_id_that_does_not_exist")

	if err == nil {
		t.Errorf("WorkspaceAdd with invalid revision should have failed")
	}

	// Verify error message mentions the invalid revision
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Logf("Error message: %v", err)
	}
}

// TestWorkspaceAddMultipleParents tests creating a workspace with multiple parent revisions.
func TestWorkspaceAddMultipleParents(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepoDir := filepath.Join(tmpDir, "main")

	// Initialize main jj repo
	initCmd := exec.Command("jj", "git", "init", mainRepoDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Create first commit
	testFile := filepath.Join(mainRepoDir, "file1.txt")
	if err := os.WriteFile(testFile, []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	commitCmd := exec.Command("jj", "commit", "-m", "first commit")
	commitCmd.Dir = mainRepoDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to create first commit: %v", err)
	}

	// Get the first commit ID
	logCmd := exec.Command("jj", "log", "-r", "@-", "-T", "commit_id.short(12)", "--no-graph")
	logCmd.Dir = mainRepoDir
	firstCommitOutput, err := logCmd.Output()
	if err != nil {
		t.Fatalf("failed to get first commit ID: %v", err)
	}
	firstCommitID := strings.TrimSpace(string(firstCommitOutput))

	// Create second commit (linear history - both reachable from current @)
	testFile2 := filepath.Join(mainRepoDir, "file2.txt")
	if err := os.WriteFile(testFile2, []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	commitCmd2 := exec.Command("jj", "commit", "-m", "second commit")
	commitCmd2.Dir = mainRepoDir
	if err := commitCmd2.Run(); err != nil {
		t.Fatalf("failed to create second commit: %v", err)
	}

	// Get the second commit ID
	logCmd2 := exec.Command("jj", "log", "-r", "@-", "-T", "commit_id.short(12)", "--no-graph")
	logCmd2.Dir = mainRepoDir
	secondCommitOutput, err := logCmd2.Output()
	if err != nil {
		t.Fatalf("failed to get second commit ID: %v", err)
	}
	secondCommitID := strings.TrimSpace(string(secondCommitOutput))

	// Open repository via FFI
	repo, err := Open(mainRepoDir)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}
	defer repo.Close()

	// Create workspace with BOTH commits as parents (merge scenario)
	// This is a valid merge case: second commit is a child of first,
	// but we explicitly ask for both as parents
	newWsPath := filepath.Join(tmpDir, "workspace2")
	err = repo.WorkspaceAdd(newWsPath, "workspace2", firstCommitID, secondCommitID)
	if err != nil {
		t.Fatalf("WorkspaceAdd with multiple parents failed: %v", err)
	}

	// Verify the new workspace's working copy has both parents
	logCmd3 := exec.Command("jj", "log", "-r", "workspace2@", "-T", "parents.map(|c| c.commit_id().short(12)).join(\",\")", "--no-graph")
	logCmd3.Dir = mainRepoDir
	ws2ParentsOutput, err := logCmd3.Output()
	if err != nil {
		t.Fatalf("failed to get workspace2 parents: %v", err)
	}
	ws2Parents := strings.TrimSpace(string(ws2ParentsOutput))

	// Should have both commits as parents
	if !strings.Contains(ws2Parents, firstCommitID) {
		t.Errorf("workspace should have first commit %s as parent, got: %s", firstCommitID, ws2Parents)
	}
	if !strings.Contains(ws2Parents, secondCommitID) {
		t.Errorf("workspace should have second commit %s as parent, got: %s", secondCommitID, ws2Parents)
	}
}

// TestWorkspaceAddNonExistentPath tests workspace creation with auto-created directory.
func TestWorkspaceAddNonExistentPath(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepoDir := filepath.Join(tmpDir, "main")

	// Initialize main jj repo
	initCmd := exec.Command("jj", "git", "init", mainRepoDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Open repository via FFI
	repo, err := Open(mainRepoDir)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}
	defer repo.Close()

	// Path that doesn't exist yet (nested)
	newWsPath := filepath.Join(tmpDir, "nested", "workspace2")

	err = repo.WorkspaceAdd(newWsPath, "workspace2")
	if err != nil {
		t.Fatalf("WorkspaceAdd should create nested directory: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(newWsPath); os.IsNotExist(err) {
		t.Errorf("nested workspace directory was not created")
	}
}

// TestWorkspaceAddExistingNonEmptyPath tests that adding workspace to non-empty directory fails.
func TestWorkspaceAddExistingNonEmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepoDir := filepath.Join(tmpDir, "main")

	// Initialize main jj repo
	initCmd := exec.Command("jj", "git", "init", mainRepoDir)
	if err := initCmd.Run(); err != nil {
		t.Skipf("jj not available or unable to initialize repo: %v", err)
	}

	// Open repository via FFI
	repo, err := Open(mainRepoDir)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}
	defer repo.Close()

	// Create a non-empty directory
	nonEmptyDir := filepath.Join(tmpDir, "nonempty")
	if err := os.MkdirAll(nonEmptyDir, 0755); err != nil {
		t.Fatalf("failed to create non-empty dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "existing.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create file in non-empty dir: %v", err)
	}

	err = repo.WorkspaceAdd(nonEmptyDir, "workspace2")
	if err == nil {
		t.Errorf("WorkspaceAdd should fail for non-empty directory")
	}
}
