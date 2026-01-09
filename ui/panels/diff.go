package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerund/jjazy/jj"
	"github.com/gerund/jjazy/ui/borders"
	"github.com/gerund/jjazy/ui/theme"
)

// DiffViewer shows the diff/patch content with syntax highlighting
type DiffViewer struct {
	BasePanel
	repo     *jj.Repo
	repoPath string
	viewport viewport.Model
	content  string
	ready    bool
}

// NewDiffViewer creates a new diff viewer panel
func NewDiffViewer(repo *jj.Repo) *DiffViewer {
	d := &DiffViewer{
		BasePanel: NewBasePanel("0 Diff", "changes"),
		repo:      repo,
		repoPath:  ".", // Default to current directory
	}
	d.loadDiff()
	return d
}

// SetRepoPath sets the repository path for CLI operations
func (d *DiffViewer) SetRepoPath(path string) {
	d.repoPath = path
}

func (d *DiffViewer) loadDiff() {
	// Get diff from jj-lib
	diff, err := d.repo.Diff()
	if err != nil {
		d.content = ""
		return
	}
	d.content = diff
}

// LoadChange loads the diff for a specific change ID
func (d *DiffViewer) LoadChange(changeID string) {
	diff, err := jj.DiffForChange(d.repoPath, changeID)
	if err != nil {
		d.content = "Error loading diff: " + err.Error()
	} else {
		d.content = diff
	}

	if d.ready {
		d.viewport.SetContent(d.renderDiff())
		d.viewport.GotoTop()
	}
}

// LoadFileInChange loads the diff for a specific file within a change
func (d *DiffViewer) LoadFileInChange(changeID, filePath string) {
	diff, err := jj.DiffForChangeFile(d.repoPath, changeID, filePath)
	if err != nil {
		d.content = "Error loading diff: " + err.Error()
	} else {
		d.content = diff
	}

	if d.ready {
		d.viewport.SetContent(d.renderDiff())
		d.viewport.GotoTop()
	}
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
	case tea.MouseMsg:
		// Handle scroll wheel for viewport
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			d.viewport.LineUp(3)
		case tea.MouseButtonWheelDown:
			d.viewport.LineDown(3)
		}

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

	if strings.TrimSpace(d.content) == "" {
		return d.RenderFrame(theme.DimmedStyle.Render("No changes"))
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
		d.viewport.SetContent(d.renderDiff())
	}
}

// renderDiff applies syntax highlighting to the diff content
func (d *DiffViewer) renderDiff() string {
	var lines []string
	// Use contentWidth - 1 to add a safety margin and prevent overflow
	maxWidth := d.ContentWidth()
	if maxWidth > 0 {
		maxWidth = maxWidth - 1
	}

	for _, line := range strings.Split(d.content, "\n") {
		var styled string

		switch {
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			// File headers
			styled = theme.DimmedStyle.Bold(true).MaxWidth(maxWidth).Render(line)
		case strings.HasPrefix(line, "@@"):
			// Hunk headers
			styled = theme.DiffHunkHeader.MaxWidth(maxWidth).Render(line)
		case strings.HasPrefix(line, "+"):
			// Added lines
			styled = theme.DiffAddLine.MaxWidth(maxWidth).Render(line)
		case strings.HasPrefix(line, "-"):
			// Removed lines
			styled = theme.DiffRemoveLine.MaxWidth(maxWidth).Render(line)
		case strings.HasPrefix(line, "diff --git"):
			// Diff header
			styled = theme.DimmedStyle.Bold(true).MaxWidth(maxWidth).Render(line)
		case strings.HasPrefix(line, "index "):
			// Index line
			styled = theme.DimmedStyle.MaxWidth(maxWidth).Render(line)
		default:
			// Context lines
			styled = theme.DiffContextLine.MaxWidth(maxWidth).Render(line)
		}

		lines = append(lines, styled)
	}

	return strings.Join(lines, "\n")
}

// RenderFrame overrides to use titled border for the main diff panel
func (d *DiffViewer) RenderFrame(content string) string {
	// Build title with scroll percentage if applicable
	title := d.title
	if d.ready && d.viewport.TotalLineCount() > d.viewport.Height {
		scrollPercent := int(d.viewport.ScrollPercent() * 100)
		title = fmt.Sprintf("%s (%d%%)", d.title, scrollPercent)
	}

	return borders.RenderTitledBorder(content, title, d.width, d.height, d.focused)
}

// Ensure DiffViewer implements Panel
var _ Panel = (*DiffViewer)(nil)
