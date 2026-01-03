package panels

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jayz/ui/fixtures"
	"github.com/gerund/jayz/ui/theme"
)

// DiffViewer shows the diff/patch content with syntax highlighting
type DiffViewer struct {
	BasePanel
	viewport viewport.Model
	content  string
	ready    bool
}

// NewDiffViewer creates a new diff viewer panel
func NewDiffViewer() *DiffViewer {
	d := &DiffViewer{
		BasePanel: NewBasePanel("Diff", "changes"),
	}
	d.loadDiff()
	return d
}

func (d *DiffViewer) loadDiff() {
	// TODO: Replace with actual jj-lib call
	// d.content = d.repo.GetDiff(revisionID)
	d.content = fixtures.DiffContent
}

// SetContent updates the diff content
func (d *DiffViewer) SetContent(content string) {
	d.content = content
	if d.ready {
		d.viewport.SetContent(d.renderDiff())
	}
}

func (d *DiffViewer) Init() tea.Cmd {
	return nil
}

func (d *DiffViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if d.focused {
			switch msg.String() {
			case "up", "k":
				d.viewport.LineUp(1)
			case "down", "j":
				d.viewport.LineDown(1)
			case "pgup", "ctrl+u":
				d.viewport.HalfViewUp()
			case "pgdown", "ctrl+d":
				d.viewport.HalfViewDown()
			case "g", "home":
				d.viewport.GotoTop()
			case "G", "end":
				d.viewport.GotoBottom()
			}
		}
	}

	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

func (d *DiffViewer) View() string {
	if !d.ready {
		return d.RenderFrame("Initializing...")
	}

	return d.RenderFrame(d.viewport.View())
}

// SetSize overrides BasePanel.SetSize to also resize viewport
func (d *DiffViewer) SetSize(width, height int) {
	d.BasePanel.SetSize(width, height)

	contentWidth := d.ContentWidth()
	contentHeight := d.ContentHeight()

	if !d.ready {
		d.viewport = viewport.New(contentWidth, contentHeight)
		d.viewport.SetContent(d.renderDiff())
		d.ready = true
	} else {
		d.viewport.Width = contentWidth
		d.viewport.Height = contentHeight
	}
}

// renderDiff applies syntax highlighting to the diff content
func (d *DiffViewer) renderDiff() string {
	var lines []string

	for _, line := range strings.Split(d.content, "\n") {
		var styled string

		switch {
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			// File headers
			styled = theme.DimmedStyle.Bold(true).Render(line)
		case strings.HasPrefix(line, "@@"):
			// Hunk headers
			styled = theme.DiffHunkHeader.Render(line)
		case strings.HasPrefix(line, "+"):
			// Added lines
			styled = theme.DiffAddLine.Render(line)
		case strings.HasPrefix(line, "-"):
			// Removed lines
			styled = theme.DiffRemoveLine.Render(line)
		case strings.HasPrefix(line, "diff --git"):
			// Diff header
			styled = theme.DimmedStyle.Bold(true).Render(line)
		case strings.HasPrefix(line, "index "):
			// Index line
			styled = theme.DimmedStyle.Render(line)
		default:
			// Context lines
			styled = theme.DiffContextLine.Render(line)
		}

		lines = append(lines, styled)
	}

	return strings.Join(lines, "\n")
}

// RenderFrame overrides to use different styling for the main diff panel
func (d *DiffViewer) RenderFrame(content string) string {
	var style lipgloss.Style
	var titleStyle lipgloss.Style

	if d.focused {
		style = theme.FocusedBorder.
			Width(d.width).
			Height(d.height)
		titleStyle = theme.FocusedTitleStyle
	} else {
		style = theme.UnfocusedBorder.
			Width(d.width).
			Height(d.height)
		titleStyle = theme.TitleStyle
	}

	// Render title with scroll position
	title := d.title
	if d.ready && d.viewport.TotalLineCount() > d.viewport.Height {
		scrollPercent := int(d.viewport.ScrollPercent() * 100)
		title = d.title + " " + theme.DimmedStyle.Render(
			lipgloss.NewStyle().Render("("+string(rune('0'+scrollPercent/10))+string(rune('0'+scrollPercent%10))+"%)"),
		)
	}
	renderedTitle := titleStyle.Render(title)

	// Combine title and content
	fullContent := lipgloss.JoinVertical(lipgloss.Left, renderedTitle, content)

	return style.Render(fullContent)
}

// Ensure DiffViewer implements Panel
var _ Panel = (*DiffViewer)(nil)
