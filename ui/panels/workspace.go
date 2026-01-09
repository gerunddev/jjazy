package panels

import (
	tea "github.com/charmbracelet/bubbletea"
)

// WorkspacePanel is a placeholder for workspace information.
type WorkspacePanel struct {
	BasePanel
}

// NewWorkspacePanel creates a new workspace panel.
func NewWorkspacePanel() *WorkspacePanel {
	return &WorkspacePanel{
		BasePanel: NewBasePanel("1 Workspace", "workspace"),
	}
}

func (p *WorkspacePanel) Init() tea.Cmd {
	return nil
}

func (p *WorkspacePanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return p, nil
}

func (p *WorkspacePanel) View() string {
	return p.RenderFrame("")
}

// Ensure WorkspacePanel implements Panel
var _ Panel = (*WorkspacePanel)(nil)
