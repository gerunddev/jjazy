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
	StartLine     int  // First line in RawANSI (0-indexed)
	EndLine       int  // Last line (exclusive)
	IsWorkingCopy bool // True if this is the current working copy (@)
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

	// Pass 2: Get structured metadata (including working copy detection)
	structuredCmd := exec.Command("jj", "log", "--no-graph", "-T",
		`"[" ++ change_id.short(8) ++ "|" ++ commit_id.short(8) ++ "|" ++ if(self.current_working_copy(), "wc", "no") ++ "]\n"`)
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
func parseStructuredLog(output string) []ChangeInfo {
	var changes []ChangeInfo
	// Match [changeID|commitID|wc/no]
	re := regexp.MustCompile(`\[([a-z]+)\|([a-f0-9]+)\|(wc|no)\]`)

	for _, line := range strings.Split(output, "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 4 {
			changes = append(changes, ChangeInfo{
				ChangeID:      matches[1],
				CommitID:      matches[2],
				IsWorkingCopy: matches[3] == "wc",
			})
		}
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

// WorkspaceSwitch switches to a different workspace.
func WorkspaceSwitch(repoPath, workspaceName string) error {
	cmd := exec.Command("jj", "workspace", "switch", workspaceName)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("workspace switch failed: %s", string(output))
	}
	return nil
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
