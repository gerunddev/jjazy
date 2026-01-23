package interactive

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/gerunddev/jjazy/jj"
)

func runEdit(repoPath string) error {
	// Get log for revision selection
	log, err := jj.LogCLI(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get log: %w", err)
	}

	// Build options from changes
	options := buildRevisionOptions(log.Changes)
	if len(options) == 0 {
		fmt.Println("No revisions available")
		return nil
	}

	var revision string
	err = huh.NewSelect[string]().
		Title("Select revision to edit").
		Options(options...).
		Value(&revision).
		Run()

	if err != nil {
		return nil // User cancelled
	}

	// Execute edit
	if err := jj.Edit(repoPath, revision); err != nil {
		return fmt.Errorf("edit failed: %w", err)
	}

	fmt.Printf("Now editing %s\n", revision)
	return nil
}

func runRebase(repoPath string) error {
	log, err := jj.LogCLI(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get log: %w", err)
	}

	options := buildRevisionOptions(log.Changes)
	if len(options) < 2 {
		fmt.Println("Need at least 2 revisions to rebase")
		return nil
	}

	// Select source revision
	var source string
	err = huh.NewSelect[string]().
		Title("Select revision to rebase (source)").
		Options(options...).
		Value(&source).
		Run()

	if err != nil {
		return nil // Cancelled
	}

	// Select destination revision
	var dest string
	err = huh.NewSelect[string]().
		Title("Select destination (new parent)").
		Description(fmt.Sprintf("Rebasing %s onto...", source)).
		Options(options...).
		Value(&dest).
		Run()

	if err != nil {
		return nil // Cancelled
	}

	if source == dest {
		fmt.Println("Source and destination cannot be the same")
		return nil
	}

	// Execute rebase
	if err := jj.Rebase(repoPath, source, dest); err != nil {
		return fmt.Errorf("rebase failed: %w", err)
	}

	fmt.Printf("Rebased %s onto %s\n", source, dest)
	return nil
}

func runSquashRange(repoPath string) error {
	log, err := jj.LogCLI(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get log: %w", err)
	}

	options := buildRevisionOptions(log.Changes)
	if len(options) < 2 {
		fmt.Println("Need at least 2 revisions to squash range")
		return nil
	}

	// Select destination commit (where changes will be squashed into)
	var dest string
	err = huh.NewSelect[string]().
		Title("Select destination commit (squash target)").
		Options(options...).
		Value(&dest).
		Run()

	if err != nil {
		return nil // Cancelled
	}

	// Select end of range
	var endRange string
	err = huh.NewSelect[string]().
		Title("Select end of range").
		Description(fmt.Sprintf("Will squash commits between %s and selected commit", dest)).
		Options(options...).
		Value(&endRange).
		Run()

	if err != nil {
		return nil // Cancelled
	}

	if dest == endRange {
		fmt.Println("Cannot squash range: destination and end of range are the same commit")
		return nil
	}

	// Determine which is lower (ancestor) and which is higher (descendant)
	var lower, higher string
	isDestAncestor, err := jj.IsAncestor(repoPath, dest, endRange)
	if err != nil {
		return fmt.Errorf("failed to check ancestry: %w", err)
	}

	if isDestAncestor {
		// dest is ancestor of endRange
		lower = dest
		higher = endRange
	} else {
		// Check if endRange is ancestor of dest
		isEndAncestor, err := jj.IsAncestor(repoPath, endRange, dest)
		if err != nil {
			return fmt.Errorf("failed to check ancestry: %w", err)
		}
		if isEndAncestor {
			lower = endRange
			higher = dest
		} else {
			fmt.Println("Cannot squash range: commits are on divergent branches")
			return nil
		}
	}

	// Get destination's current description to pre-fill the input
	// If this fails, we proceed with an empty message rather than blocking the operation
	currentDesc, err := jj.GetDescription(repoPath, lower)
	if err != nil {
		// Non-fatal: use empty string if we can't fetch the description
		currentDesc = ""
	}
	message := currentDesc

	err = huh.NewInput().
		Title("Enter commit message for squashed result").
		Value(&message).
		Run()

	if err != nil {
		return nil // Cancelled
	}

	// Execute squash range
	if err := jj.SquashRange(repoPath, lower, higher, message); err != nil {
		return fmt.Errorf("squash range failed: %w", err)
	}

	fmt.Printf("Squashed commits from %s down to %s\n", higher, lower)
	return nil
}

func buildRevisionOptions(changes []jj.ChangeInfo) []huh.Option[string] {
	var options []huh.Option[string]
	for _, c := range changes {
		label := c.ChangeID
		if c.IsWorkingCopy {
			label += " @"
		}
		if len(c.Bookmarks) > 0 {
			label += " [" + strings.Join(c.Bookmarks, ", ") + "]"
		}
		if c.Description != "" {
			label += " " + c.Description
		} else {
			label += " (no description)"
		}
		options = append(options, huh.NewOption(label, c.ChangeID))
	}
	return options
}
