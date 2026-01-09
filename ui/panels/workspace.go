package panels

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerund/jjazy/jj"
	"github.com/gerund/jjazy/ui/borders"
	"github.com/gerund/jjazy/ui/theme"
)

// WorkspacePanel shows workspaces in the repository.
// This is a "browsable" panel that requires Enter to show cursor.
type WorkspacePanel struct {
	BasePanel
	repo       *jj.Repo
	workspaces []jj.Workspace
	viewport   viewport.Model
	ready      bool
}

// SetEntered overrides BasePanel to also reset viewport and re-render
func (p *WorkspacePanel) SetEntered(entered bool) {
	p.BasePanel.SetEntered(entered)
	if p.ready {
		if entered {
			p.viewport.GotoTop()
		}
		// Re-render to show/hide cursor styling
		p.viewport.SetContent(p.renderContent())
	}
}

// NewWorkspacePanel creates a new workspace panel.
func NewWorkspacePanel(repo *jj.Repo) *WorkspacePanel {
	p := &WorkspacePanel{
		BasePanel: NewBasePanel("1 Workspace", "workspace"),
		repo:      repo,
	}
	p.loadWorkspaces()
	return p
}

func (p *WorkspacePanel) loadWorkspaces() {
	workspaces, err := p.repo.Workspaces()
	if err != nil {
		p.workspaces = nil
		return
	}
	p.workspaces = workspaces
}

// Refresh reloads workspace data and re-renders.
func (p *WorkspacePanel) Refresh() {
	p.loadWorkspaces()
	if p.ready {
		p.viewport.SetContent(p.renderContent())
	}
}

func (p *WorkspacePanel) Init() tea.Cmd {
	return nil
}

func (p *WorkspacePanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress && p.entered {
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
		// Only process cursor keys when focused AND entered
		if !p.focused || !p.entered {
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

func (p *WorkspacePanel) ensureCursorVisible() {
	if p.cursor < p.viewport.YOffset {
		p.viewport.SetYOffset(p.cursor)
	} else if p.cursor >= p.viewport.YOffset+p.viewport.Height {
		p.viewport.SetYOffset(p.cursor - p.viewport.Height + 1)
	}
}

func (p *WorkspacePanel) View() string {
	if !p.ready {
		return p.RenderFrame("Loading...")
	}
	return p.RenderFrame(p.viewport.View())
}

// RenderFrame overrides BasePanel to show white border in cursor mode
func (p *WorkspacePanel) RenderFrame(content string) string {
	// Focus mode: yellow border (focused && !entered)
	// Cursor mode: white border (entered)
	showFocusBorder := p.focused && !p.entered
	return borders.RenderTitledBorder(content, p.title, p.width, p.height, showFocusBorder)
}

// SetSize initializes or resizes the viewport
func (p *WorkspacePanel) SetSize(width, height int) {
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

func (p *WorkspacePanel) renderContent() string {
	var lines []string
	contentWidth := p.ContentWidth()

	for i, ws := range p.workspaces {
		// Truncate if needed
		name := ws.Name
		if len(name)+2 > contentWidth && contentWidth > 3 {
			name = truncate(name, contentWidth-3)
		}

		// Style the name based on current/selected state
		// Cursor (yellow) takes priority when entered
		var styledName string
		if i == p.cursor && p.focused && p.entered {
			// Selected + entered: YELLOW (overrides current color)
			styledName = theme.SelectedItemStyle.Render(name)
		} else if ws.IsCurrent {
			// Current workspace is GREEN
			styledName = theme.WorkingCopyStyle.Render(name)
		} else {
			// Normal: WHITE
			styledName = theme.NormalItemStyle.Render(name)
		}

		line := styledName
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// SelectedWorkspace returns the currently selected workspace
func (p *WorkspacePanel) SelectedWorkspace() *jj.Workspace {
	if p.cursor >= 0 && p.cursor < len(p.workspaces) {
		return &p.workspaces[p.cursor]
	}
	return nil
}

// Count returns the number of workspaces
func (p *WorkspacePanel) Count() int {
	return len(p.workspaces)
}

// Ensure WorkspacePanel implements Panel
var _ Panel = (*WorkspacePanel)(nil)
