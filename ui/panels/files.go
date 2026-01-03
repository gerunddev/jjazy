package panels

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jayz/jj"
	"github.com/gerund/jayz/ui/fixtures"
	"github.com/gerund/jayz/ui/theme"
)

// FilesPanel shows files changed in the current revision
type FilesPanel struct {
	BasePanel
	repo  *jj.Repo
	files []fixtures.FileChange
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
	if !p.focused {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			p.CursorUp(len(p.files))
		case "down", "j":
			p.CursorDown(len(p.files))
		case "g", "home":
			p.CursorHome()
		case "G", "end":
			p.CursorEnd(len(p.files))
		}
	}

	return p, nil
}

func (p *FilesPanel) View() string {
	var lines []string

	contentHeight := p.ContentHeight()
	contentWidth := p.ContentWidth()

	for i, file := range p.files {
		if i >= contentHeight {
			break
		}

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

		path := file.Path
		if i == p.cursor && p.focused {
			path = theme.SelectedItemStyle.Render(path)
		} else {
			path = theme.NormalItemStyle.Render(path)
		}

		// Truncate path if needed
		maxPathLen := contentWidth - 3 // status + space
		if len(file.Path) > maxPathLen {
			path = theme.NormalItemStyle.Render(truncate(file.Path, maxPathLen))
			if i == p.cursor && p.focused {
				path = theme.SelectedItemStyle.Render(truncate(file.Path, maxPathLen))
			}
		}

		line := status + " " + path
		lines = append(lines, line)
	}

	// Pad remaining space
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return p.RenderFrame(content)
}

// SelectedFile returns the currently selected file
func (p *FilesPanel) SelectedFile() *fixtures.FileChange {
	if p.cursor >= 0 && p.cursor < len(p.files) {
		return &p.files[p.cursor]
	}
	return nil
}

// Ensure FilesPanel implements Panel
var _ Panel = (*FilesPanel)(nil)
