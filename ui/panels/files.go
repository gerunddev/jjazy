package panels

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jjazy/jj"
	"github.com/gerund/jjazy/ui/fixtures"
	"github.com/gerund/jjazy/ui/messages"
	"github.com/gerund/jjazy/ui/theme"
)

// FilesPanel shows files changed in the current revision
type FilesPanel struct {
	BasePanel
	repo     *jj.Repo
	files    []fixtures.FileChange
	viewport viewport.Model
	ready    bool
}

// NewFilesPanel creates a new files panel
func NewFilesPanel(repo *jj.Repo) *FilesPanel {
	p := &FilesPanel{
		BasePanel: NewBasePanel("2 Files", "changes"),
		repo:      repo,
	}
	p.loadFiles()
	return p
}

func (p *FilesPanel) loadFiles() {
	// Get file changes from jj-lib
	changes, err := p.repo.WorkingCopyChanges()
	if err != nil {
		// Fall back to empty list on error
		p.files = nil
		return
	}

	// Convert jj.FileChange to fixtures.FileChange
	p.files = make([]fixtures.FileChange, len(changes))
	for i, fc := range changes {
		var status fixtures.FileStatus
		switch fc.Status {
		case "added":
			status = fixtures.StatusAdded
		case "deleted":
			status = fixtures.StatusDeleted
		default:
			status = fixtures.StatusModified
		}
		p.files[i] = fixtures.FileChange{
			Path:   fc.Path,
			Status: status,
		}
	}
}

func (p *FilesPanel) Init() tea.Cmd {
	return nil
}

func (p *FilesPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	prevCursor := p.cursor

	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Handle mouse events even when not focused
		switch msg.Button {
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress {
				// Convert Y to item index (subtract 1 for top border, add viewport offset)
				itemIndex := msg.Y - 1 + p.viewport.YOffset
				if itemIndex >= 0 && itemIndex < len(p.files) {
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
			p.CursorUp(len(p.files))
			p.ensureCursorVisible()
		case "down", "j":
			p.CursorDown(len(p.files))
			p.ensureCursorVisible()
		case "g", "home":
			p.CursorHome()
			p.viewport.GotoTop()
		case "G", "end":
			p.CursorEnd(len(p.files))
			p.viewport.GotoBottom()
		case "ctrl+u", "pgup":
			p.viewport.HalfViewUp()
		case "ctrl+d", "pgdown":
			p.viewport.HalfViewDown()
		}
	}

	// Update viewport content when cursor changes
	if p.ready {
		p.viewport.SetContent(p.renderContent())
	}

	// Emit selection message if cursor changed
	if p.cursor != prevCursor {
		if file := p.SelectedFile(); file != nil {
			return p, func() tea.Msg {
				return messages.FileSelectedMsg{Path: file.Path}
			}
		}
	}

	return p, nil
}

func (p *FilesPanel) ensureCursorVisible() {
	if p.cursor < p.viewport.YOffset {
		p.viewport.SetYOffset(p.cursor)
	} else if p.cursor >= p.viewport.YOffset+p.viewport.Height {
		p.viewport.SetYOffset(p.cursor - p.viewport.Height + 1)
	}
}

func (p *FilesPanel) View() string {
	if !p.ready {
		return p.RenderFrame("Loading...")
	}
	return p.RenderFrame(p.viewport.View())
}

// SetSize initializes or resizes the viewport
func (p *FilesPanel) SetSize(width, height int) {
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

func (p *FilesPanel) renderContent() string {
	var lines []string
	contentWidth := p.ContentWidth()

	for i, file := range p.files {
		// Style the status indicator based on file status
		var statusStyle lipgloss.Style
		switch file.Status {
		case fixtures.StatusModified:
			statusStyle = theme.ModifiedStyle
		case fixtures.StatusAdded:
			statusStyle = theme.AddedStyle
		case fixtures.StatusDeleted:
			statusStyle = theme.DeletedStyle
		case fixtures.StatusRenamed:
			statusStyle = theme.RenamedStyle
		case fixtures.StatusConflict:
			statusStyle = theme.ConflictStyle
		default:
			statusStyle = theme.NormalItemStyle
		}

		status := statusStyle.Render(file.Status.String())

		// Truncate path if needed
		maxPathLen := contentWidth - 3 // status + space
		path := file.Path
		if len(path) > maxPathLen && maxPathLen > 0 {
			path = truncate(path, maxPathLen)
		}

		if i == p.cursor && p.focused {
			path = theme.SelectedItemStyle.Render(path)
		} else {
			path = theme.NormalItemStyle.Render(path)
		}

		line := status + " " + path
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// SelectedFile returns the currently selected file
func (p *FilesPanel) SelectedFile() *fixtures.FileChange {
	if p.cursor >= 0 && p.cursor < len(p.files) {
		return &p.files[p.cursor]
	}
	return nil
}

// Count returns the number of files
func (p *FilesPanel) Count() int {
	return len(p.files)
}

// Ensure FilesPanel implements Panel
var _ Panel = (*FilesPanel)(nil)
