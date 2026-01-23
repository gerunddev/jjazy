package panels

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerunddev/jjazy/jj"
	"github.com/gerunddev/jjazy/ui/fixtures"
	"github.com/gerunddev/jjazy/ui/theme"
)

// OperationsPanel shows operation history (undo stack)
type OperationsPanel struct {
	BasePanel
	repo       *jj.Repo
	operations []fixtures.Operation
	viewport   viewport.Model
	ready      bool
}

// NewOperationsPanel creates a new operations panel
func NewOperationsPanel(repo *jj.Repo) *OperationsPanel {
	p := &OperationsPanel{
		BasePanel: NewBasePanel("4 Operations", "undo"),
		repo:      repo,
	}
	p.loadOperations()
	return p
}

func (p *OperationsPanel) loadOperations() {
	// Get operations from jj-lib
	ops, err := p.repo.Operations()
	if err != nil {
		// Fall back to empty list on error
		p.operations = nil
		return
	}

	// Convert jj.Operation to fixtures.Operation
	p.operations = make([]fixtures.Operation, len(ops))
	for i, op := range ops {
		p.operations[i] = fixtures.Operation{
			ID:          op.ID,
			Description: op.Description,
			Timestamp:   op.Timestamp, // Unix timestamp (TODO: format as relative time)
			IsCurrent:   op.IsCurrent,
		}
	}
}

func (p *OperationsPanel) Init() tea.Cmd {
	return nil
}

func (p *OperationsPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft {
			p.HandleMouseClick(msg, &p.viewport, len(p.operations))
		} else {
			p.HandleMouseScroll(msg, &p.viewport)
		}

	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		p.HandleCursorKeys(msg, len(p.operations), &p.viewport)
	}

	if p.ready {
		p.viewport.SetContent(p.renderContent())
	}

	return p, nil
}

func (p *OperationsPanel) View() string {
	if !p.ready {
		return p.RenderFrame("Loading...")
	}
	return p.RenderFrame(p.viewport.View())
}

// SetSize initializes or resizes the viewport
func (p *OperationsPanel) SetSize(width, height int) {
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

func (p *OperationsPanel) renderContent() string {
	var lines []string
	contentWidth := p.ContentWidth()

	for i, op := range p.operations {
		// Build the line with indicator for current operation
		var indicator string
		if op.IsCurrent {
			indicator = theme.WorkingCopyStyle.Render("● ")
		} else {
			indicator = theme.DimmedStyle.Render("○ ")
		}

		// Timestamp
		timestamp := theme.TimestampStyle.Render(op.Timestamp)

		// Calculate space for description
		maxDescLen := contentWidth - len(op.Timestamp) - 4 // indicator + space + timestamp
		desc := op.Description
		if len(desc) > maxDescLen && maxDescLen > 0 {
			desc = truncate(desc, maxDescLen)
		}

		if i == p.cursor && p.focused {
			desc = theme.SelectedItemStyle.Render(desc)
		} else {
			desc = theme.NormalItemStyle.Render(desc)
		}

		line := indicator + desc + " " + timestamp
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// SelectedOperation returns the currently selected operation
func (p *OperationsPanel) SelectedOperation() *fixtures.Operation {
	if p.cursor >= 0 && p.cursor < len(p.operations) {
		return &p.operations[p.cursor]
	}
	return nil
}

// Count returns the number of operations
func (p *OperationsPanel) Count() int {
	return len(p.operations)
}

// Ensure OperationsPanel implements Panel
var _ Panel = (*OperationsPanel)(nil)
