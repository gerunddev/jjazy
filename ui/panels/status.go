package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerunddev/jjazy/jj"
	"github.com/gerunddev/jjazy/ui/fixtures"
	"github.com/gerunddev/jjazy/ui/theme"
)

// StatusPanel shows workspaces and current revision info
type StatusPanel struct {
	BasePanel
	repo       *jj.Repo
	workspaces []fixtures.Workspace
	viewport   viewport.Model
	ready      bool
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
	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress {
				itemIndex := msg.Y - 1 + p.viewport.YOffset
				if itemIndex >= 0 && itemIndex < len(p.workspaces) {
					p.cursor = itemIndex
					p.ensureCursorVisible()
				}
			}
		case tea.MouseButtonWheelUp:
			p.viewport.LineUp(3)
		case tea.MouseButtonWheelDown:
			p.viewport.LineDown(3)
		}

	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		switch msg.String() {
		case "up", "k":
			p.CursorUp(len(p.workspaces))
			p.ensureCursorVisible()
		case "down", "j":
			p.CursorDown(len(p.workspaces))
			p.ensureCursorVisible()
		case "g", "home":
			p.CursorHome()
			p.viewport.GotoTop()
		case "G", "end":
			p.CursorEnd(len(p.workspaces))
			p.viewport.GotoBottom()
		case "ctrl+u", "pgup":
			p.viewport.HalfViewUp()
		case "ctrl+d", "pgdown":
			p.viewport.HalfViewDown()
		}
	}

	if p.ready {
		p.viewport.SetContent(p.renderContent())
	}

	return p, nil
}

func (p *StatusPanel) ensureCursorVisible() {
	if p.cursor < p.viewport.YOffset {
		p.viewport.SetYOffset(p.cursor)
	} else if p.cursor >= p.viewport.YOffset+p.viewport.Height {
		p.viewport.SetYOffset(p.cursor - p.viewport.Height + 1)
	}
}

func (p *StatusPanel) View() string {
	if !p.ready {
		return p.RenderFrame("Loading...")
	}
	return p.RenderFrame(p.viewport.View())
}

// SetSize initializes or resizes the viewport
func (p *StatusPanel) SetSize(width, height int) {
	p.BasePanel.SetSize(width, height)

	contentWidth := p.ContentWidth()
	contentHeight := p.ContentHeight()

	if !p.ready {
		p.viewport = viewport.New(contentWidth, contentHeight)
		p.viewport.SetContent(p.renderContent())
		p.ready = true
	} else {
		p.viewport.Width = contentWidth
		p.viewport.Height = contentHeight
		p.viewport.SetContent(p.renderContent())
	}
}

func (p *StatusPanel) renderContent() string {
	var lines []string
	contentWidth := p.ContentWidth()

	for i, ws := range p.workspaces {
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

		line := indicator + name
		if len(ws.Name)+2 > contentWidth && contentWidth > 3 {
			name = truncate(ws.Name, contentWidth-3)
			if i == p.cursor && p.focused {
				name = theme.SelectedItemStyle.Render(name)
			} else {
				name = theme.NormalItemStyle.Render(name)
			}
			line = indicator + name
		}

		lines = append(lines, line)

		// Show revision info for current workspace
		if ws.IsCurrent {
			revInfo := fmt.Sprintf("  %s %s",
				theme.RevisionIDStyle.Render(ws.RevisionID),
				theme.DimmedStyle.Render(truncate(ws.Description, contentWidth-12)),
			)
			lines = append(lines, revInfo)
		}
	}

	return strings.Join(lines, "\n")
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
