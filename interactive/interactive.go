package interactive

import (
	"github.com/charmbracelet/huh"
)

// Run starts the interactive mode
func Run(repoPath string) error {
	var action string

	err := huh.NewSelect[string]().
		Title("jjazy - Quick Actions").
		Options(
			huh.NewOption("Edit - Switch working copy to revision", "edit"),
			huh.NewOption("Rebase - Move revision to new parent", "rebase"),
		).
		Value(&action).
		Run()

	if err != nil {
		return err // User cancelled
	}

	switch action {
	case "edit":
		return runEdit(repoPath)
	case "rebase":
		return runRebase(repoPath)
	}

	return nil
}
