package jj

import (
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
	ChangeID  string
	CommitID  string
	StartLine int // First line in RawANSI (0-indexed)
	EndLine   int // Last line (exclusive)
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

	// Pass 2: Get structured metadata
	structuredCmd := exec.Command("jj", "log", "--no-graph", "-T",
		`"[" ++ change_id.short(8) ++ "|" ++ commit_id.short(8) ++ "]\n"`)
	structuredCmd.Dir = repoPath
	structuredOutput, err := structuredCmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse structured output to get change/commit IDs
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
	// Match [changeID|commitID]
	re := regexp.MustCompile(`\[([a-z]+)\|([a-f0-9]+)\]`)

	for _, line := range strings.Split(output, "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			changes = append(changes, ChangeInfo{
				ChangeID: matches[1],
				CommitID: matches[2],
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
