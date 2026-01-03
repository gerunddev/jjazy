package panels

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerund/jayz/jj"
	"github.com/gerund/jayz/ui/fixtures"
	"github.com/gerund/jayz/ui/theme"
)

// OperationsPanel shows operation history (undo stack)
type OperationsPanel struct {
	BasePanel
	repo       *jj.Repo
	operations []fixtures.Operation
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
	if !p.focused {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			p.CursorUp(len(p.operations))
		case "down", "j":
			p.CursorDown(len(p.operations))
		case "g", "home":
			p.CursorHome()
		case "G", "end":
			p.CursorEnd(len(p.operations))
		}
	}

	return p, nil
}

func (p *OperationsPanel) View() string {
	var lines []string

	contentHeight := p.ContentHeight()
	contentWidth := p.ContentWidth()

	for i, op := range p.operations {
		if i >= contentHeight {
			break
		}

		// Build the line with indicator for current operation
		indicator := "  "
		if op.IsCurrent {
			indicator = theme.WorkingCopyStyle.Render("● ")
		} else {
			indicator = theme.DimmedStyle.Render("○ ")
		}

		// Operation description
		desc := op.Description
		if i == p.cursor && p.focused {
			desc = theme.SelectedItemStyle.Render(desc)
		} else {
			desc = theme.NormalItemStyle.Render(desc)
		}

		// Timestamp
		timestamp := theme.TimestampStyle.Render(op.Timestamp)

		// Calculate space for description
		maxDescLen := contentWidth - len(op.Timestamp) - 4 // indicator + space + timestamp
		if len(op.Description) > maxDescLen {
			truncatedDesc := truncate(op.Description, maxDescLen)
			if i == p.cursor && p.focused {
				desc = theme.SelectedItemStyle.Render(truncatedDesc)
			} else {
				desc = theme.NormalItemStyle.Render(truncatedDesc)
			}
		}

		line := indicator + desc + " " + timestamp
		lines = append(lines, line)
	}

	// Pad remaining space
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return p.RenderFrame(content)
}

// SelectedOperation returns the currently selected operation
func (p *OperationsPanel) SelectedOperation() *fixtures.Operation {
	if p.cursor >= 0 && p.cursor < len(p.operations) {
		return &p.operations[p.cursor]
	}
	return nil
}

// Ensure OperationsPanel implements Panel
var _ Panel = (*OperationsPanel)(nil)
