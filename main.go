package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerunddev/jjazy/interactive"
	"github.com/gerunddev/jjazy/jj"
	"github.com/gerunddev/jjazy/ui"
)

// Version is set by ldflags during build
var Version = "dev"

func main() {
	// Parse flags
	interactiveMode := flag.Bool("i", false, "Run in interactive mode (quick actions)")
	showVersion := flag.Bool("version", false, "Print version and exit")
	showVersionShort := flag.Bool("v", false, "Print version and exit (shorthand)")
	flag.Parse()

	// Handle version flag
	if *showVersion || *showVersionShort {
		fmt.Printf("jjazy version %s\n", Version)
		os.Exit(0)
	}

	// Open the repository in the current directory
	repo, err := jj.Open(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening repository: %v\n", err)
		os.Exit(1)
	}
	defer repo.Close()

	// Dispatch based on mode
	if *interactiveMode {
		if err := interactive.Run("."); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Full TUI mode (default)
	app := ui.NewApp(repo, ".")

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
