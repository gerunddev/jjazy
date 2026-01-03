package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jayz/jj"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list list.Model
	repo *jj.Repo
}

func initialModel() (model, error) {
	// Open the repository in the current directory
	repo, err := jj.Open(".")
	if err != nil {
		return model{}, fmt.Errorf("failed to open repo: %w", err)
	}

	// Get branches
	branches, err := repo.Branches()
	if err != nil {
		repo.Close()
		return model{}, fmt.Errorf("failed to list branches: %w", err)
	}

	// Convert to list items
	items := make([]list.Item, len(branches))
	for i, b := range branches {
		desc := "local"
		if !b.IsLocal {
			desc = "remote"
		}
		items[i] = item{title: b.Name, desc: desc}
	}

	// Create the list
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Branches"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return model{list: l, repo: repo}, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			if m.repo != nil {
				m.repo.Close()
			}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func main() {
	m, err := initialModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
