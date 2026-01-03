package panels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerund/jayz/jj"
	"github.com/gerund/jayz/ui/fixtures"
	"github.com/gerund/jayz/ui/theme"
)

// StatusPanel shows workspaces and current revision info
type StatusPanel struct {
	BasePanel
	repo       *jj.Repo
	workspaces []fixtures.Workspace
}

// NewStatusPanel creates a new status panel
func NewStatusPanel(repo *jj.Repo) *StatusPanel {
	p := &StatusPanel{
		BasePanel: NewBasePanel("1 Status", "workspaces"),
		repo:      repo,
	}
	p.loadWorkspaces()
	return p
}

func (p *StatusPanel) loadWorkspaces() {
	// Get workspaces from jj-lib
	workspaces, err := p.repo.Workspaces()
	if err != nil {
		// Fall back to empty list on error
		p.workspaces = nil
		return
	}

	// Convert jj.Workspace to fixtures.Workspace
	p.workspaces = make([]fixtures.Workspace, len(workspaces))
	for i, ws := range workspaces {
		p.workspaces[i] = fixtures.Workspace{
			Name:       ws.Name,
			IsCurrent:  ws.IsCurrent,
			RevisionID: ws.CommitID[:8], // Short commit ID
			// TODO: Get these from jj-lib when available
			ChangeID:    "",
			Description: "",
		}
	}
}

func (p *StatusPanel) Init() tea.Cmd {
	return nil
}

func (p *StatusPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !p.focused {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			p.CursorUp(len(p.workspaces))
		case "down", "j":
			p.CursorDown(len(p.workspaces))
		case "g", "home":
			p.CursorHome()
		case "G", "end":
			p.CursorEnd(len(p.workspaces))
		}
	}

	return p, nil
}

func (p *StatusPanel) View() string {
	var lines []string

	contentHeight := p.ContentHeight()
	contentWidth := p.ContentWidth()

	for i, ws := range p.workspaces {
		if i >= contentHeight {
			break
		}

		// Build the line
		indicator := "  "
		if ws.IsCurrent {
			indicator = theme.WorkingCopyStyle.Render("● ")
		}

		name := ws.Name
		if i == p.cursor && p.focused {
			name = theme.SelectedItemStyle.Render(name)
		} else {
			name = theme.NormalItemStyle.Render(name)
		}

		// Truncate if needed
		line := indicator + name
		if len(line) > contentWidth {
			line = line[:contentWidth-1] + "…"
		}

		lines = append(lines, line)

		// Show revision info for current workspace
		if ws.IsCurrent && i+1 < contentHeight {
			revInfo := fmt.Sprintf("  %s %s",
				theme.RevisionIDStyle.Render(ws.RevisionID),
				theme.DimmedStyle.Render(truncate(ws.Description, contentWidth-12)),
			)
			lines = append(lines, revInfo)
		}
	}

	// Pad remaining space
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return p.RenderFrame(content)
}

// SelectedWorkspace returns the currently selected workspace
func (p *StatusPanel) SelectedWorkspace() *fixtures.Workspace {
	if p.cursor >= 0 && p.cursor < len(p.workspaces) {
		return &p.workspaces[p.cursor]
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}

// Ensure StatusPanel implements Panel
var _ Panel = (*StatusPanel)(nil)
