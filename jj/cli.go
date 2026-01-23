package jj

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// LogOutput contains the CLI log output and metadata for selection.
type LogOutput struct {
	RawANSI      string       // Pretty output from jj log --color=always
	LineToChange []string     // lineIndex → changeID (empty string for continuation lines)
	Changes      []ChangeInfo // Unique changes in order
}

// ChangeInfo represents a change in the log output.
type ChangeInfo struct {
	ChangeID      string
	CommitID      string
	Description   string   // First line of description (empty if none)
	Bookmarks     []string // Bookmarks pointing to this change
	StartLine     int      // First line in RawANSI (0-indexed)
	EndLine       int      // Last line (exclusive)
	IsWorkingCopy bool     // True if this is the current working copy (@)
}

// LogCLI fetches the log using the jj CLI and returns structured output.
// This uses a two-pass approach:
// 1. Get pretty ANSI output for display
// 2. Get structured data to map lines to changes
func LogCLI(repoPath string) (*LogOutput, error) {
	// Pass 1: Get pretty output with colors
	prettyCmd := exec.Command("jj", "log", "--color=always")
	prettyCmd.Dir = repoPath
	prettyOutput, err := prettyCmd.Output()
	if err != nil {
		return nil, err
	}
	rawANSI := string(prettyOutput)

	// Pass 2: Get structured metadata (including working copy detection, description, bookmarks)
	structuredCmd := exec.Command("jj", "log", "--no-graph", "-T",
		`change_id.short(8) ++ "<<SEP>>" ++ commit_id.short(8) ++ "<<SEP>>" ++ if(self.current_working_copy(), "wc", "no") ++ "<<SEP>>" ++ if(description, description.first_line(), "") ++ "<<SEP>>" ++ bookmarks.join(",") ++ "\n"`)
	structuredCmd.Dir = repoPath
	structuredOutput, err := structuredCmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse structured output to get change/commit IDs and working copy status
	changes := parseStructuredLog(string(structuredOutput))

	// Build line-to-change mapping by finding change IDs in the pretty output
	lines := strings.Split(rawANSI, "\n")
	lineToChange := make([]string, len(lines))

	// For each line, check if it contains a change ID
	// The change ID appears on the first line of each revision
	currentChangeIdx := -1
	for i, line := range lines {
		// Strip ANSI codes for searching
		plainLine := stripANSI(line)

		// Check if this line starts a new change
		for idx, change := range changes {
			if strings.Contains(plainLine, change.ChangeID) {
				currentChangeIdx = idx
				changes[idx].StartLine = i
				break
			}
		}

		// Assign current change to this line
		if currentChangeIdx >= 0 && currentChangeIdx < len(changes) {
			lineToChange[i] = changes[currentChangeIdx].ChangeID
		}
	}

	// Calculate end lines for each change
	for i := range changes {
		if i < len(changes)-1 {
			changes[i].EndLine = changes[i+1].StartLine
		} else {
			changes[i].EndLine = len(lines)
		}
	}

	return &LogOutput{
		RawANSI:      rawANSI,
		LineToChange: lineToChange,
		Changes:      changes,
	}, nil
}

// parseStructuredLog parses the structured template output into ChangeInfo slices.
// Format: changeID<<SEP>>commitID<<SEP>>wc/no<<SEP>>description<<SEP>>bookmarks
func parseStructuredLog(output string) []ChangeInfo {
	var changes []ChangeInfo
	const sep = "<<SEP>>"

	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, sep)
		if len(parts) < 5 {
			continue
		}

		var bookmarks []string
		if parts[4] != "" {
			bookmarks = strings.Split(parts[4], ",")
		}

		changes = append(changes, ChangeInfo{
			ChangeID:      parts[0],
			CommitID:      parts[1],
			IsWorkingCopy: parts[2] == "wc",
			Description:   parts[3],
			Bookmarks:     bookmarks,
		})
	}

	return changes
}

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

// FileChange represents a file changed in a revision (from CLI).
type CLIFileChange struct {
	Path   string
	Status string // "M" for modified, "A" for added, "D" for deleted
}

// FilesForChange returns the files changed in a specific change using CLI.
func FilesForChange(repoPath, changeID string) ([]CLIFileChange, error) {
	// Use jj diff --summary to get file list
	cmd := exec.Command("jj", "diff", "-r", changeID, "--summary")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []CLIFileChange
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "M path/to/file" or "A path" or "D path"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			files = append(files, CLIFileChange{
				Status: parts[0],
				Path:   parts[1],
			})
		}
	}

	return files, nil
}

// DiffForChange returns the diff content for a specific change using CLI.
func DiffForChange(repoPath, changeID string) (string, error) {
	cmd := exec.Command("jj", "diff", "-r", changeID, "--color=never")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// DiffForChangeFile returns the diff for a specific file within a change.
func DiffForChangeFile(repoPath, changeID, filePath string) (string, error) {
	cmd := exec.Command("jj", "diff", "-r", changeID, "--color=never", filePath)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// Edit runs jj edit to edit a specific revision.
func Edit(repoPath, revisionSpec string) error {
	cmd := exec.Command("jj", "edit", revisionSpec)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("edit failed: %s", string(output))
	}
	return nil
}

// RestoreFile discards changes to a file in the working copy
func RestoreFile(repoPath, filePath string) error {
	cmd := exec.Command("jj", "restore", filePath)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restore failed: %s", string(output))
	}
	return nil
}

// SquashFile moves file changes from working copy to parent
func SquashFile(repoPath, filePath string) error {
	cmd := exec.Command("jj", "squash", "--from", "@", "--into", "@-", filePath)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("squash failed: %s", string(output))
	}
	return nil
}

// NewChange creates a new change after the specified change
func NewChange(repoPath, changeID string) error {
	cmd := exec.Command("jj", "new", "--after", changeID)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("new failed: %s", string(output))
	}
	return nil
}

// GetDescription returns the description of a change
func GetDescription(repoPath, changeID string) (string, error) {
	cmd := exec.Command("jj", "log", "-r", changeID, "--no-graph", "-T", "if(description, description, \"\")")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("get description failed: %s", string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

// Describe sets the description of a change
func Describe(repoPath, changeID, message string) error {
	cmd := exec.Command("jj", "describe", "-r", changeID, "-m", message)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("describe failed: %s", string(output))
	}
	return nil
}

// Abandon removes a change and rebases its descendants
func Abandon(repoPath, changeID string) error {
	cmd := exec.Command("jj", "abandon", changeID)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("abandon failed: %s", string(output))
	}
	return nil
}

// Squash squashes a change into its parent
func Squash(repoPath, changeID string) error {
	cmd := exec.Command("jj", "squash", "-r", changeID)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("squash failed: %s", string(output))
	}
	return nil
}

// Rebase moves a revision to a new parent
// jj rebase -r <source> -d <destination>
func Rebase(repoPath, sourceRev, destRev string) error {
	cmd := exec.Command("jj", "rebase", "-r", sourceRev, "-d", destRev)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rebase failed: %s", string(output))
	}
	return nil
}

// RebaseBranch rebases a revision and its descendants
// jj rebase -b <branch> -d <destination>
func RebaseBranch(repoPath, branchRev, destRev string) error {
	cmd := exec.Command("jj", "rebase", "-b", branchRev, "-d", destRev)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rebase failed: %s", string(output))
	}
	return nil
}

// BuildSquashRevset creates a revset for selecting a range of commits
// from lower (ancestor) to higher (descendant), excluding the lower commit itself.
// The format is "lower::higher ~ lower" which selects all commits from lower to higher
// (inclusive), then excludes lower since we don't want to squash the destination itself.
func BuildSquashRevset(lower, higher string) string {
	return fmt.Sprintf("%s::%s ~ %s", lower, higher, lower)
}

// SquashRange squashes a range of commits into a destination commit.
// It uses BuildSquashRevset to create the revset for --from, and squashes
// into the specified destination with the given message.
// jj squash --from <revset> --into <dest> -m <message>
func SquashRange(repoPath, lower, higher, message string) error {
	revset := BuildSquashRevset(lower, higher)
	cmd := exec.Command("jj", "squash", "--from", revset, "--into", lower, "-m", message)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("squash range failed: %s", string(output))
	}
	return nil
}

// IsAncestor checks if commit A is an ancestor of commit B using the jj CLI.
// It uses the revset "A & ::B" - if A is an ancestor of B (or A equals B),
// the intersection will return A. Otherwise, it returns nothing.
// Returns true if A is an ancestor of (or equal to) B, false otherwise.
func IsAncestor(repoPath, commitA, commitB string) (bool, error) {
	// The revset "A & ::B" means "A intersected with ancestors of B (including B)"
	// If A is an ancestor of B, this returns A. Otherwise, it returns nothing.
	revset := fmt.Sprintf("%s & ::%s", commitA, commitB)
	cmd := exec.Command("jj", "log", "-r", revset, "--no-graph", "-T", `change_id.short(8)`)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		// Check if this is just an empty result (no match) vs actual error
		if exitErr, ok := err.(*exec.ExitError); ok {
			// jj returns exit code 1 for empty revsets
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to check ancestry: %w", err)
	}
	// If output is non-empty, A is an ancestor of B
	return strings.TrimSpace(string(output)) != "", nil
}
