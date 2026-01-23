package panels

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerunddev/jjazy/jj"
	"github.com/gerunddev/jjazy/ui/fixtures"
	"github.com/gerunddev/jjazy/ui/messages"
	"github.com/gerunddev/jjazy/ui/theme"
)

// FilesPanel shows files changed in the current revision
type FilesPanel struct {
	BasePanel
	repo     *jj.Repo
	repoPath string
	files    []fixtures.FileChange
	loadErr  error
	viewport viewport.Model
	ready    bool
}

// NewFilesPanel creates a new files panel
func NewFilesPanel(repo *jj.Repo) *FilesPanel {
	p := &FilesPanel{
		BasePanel: NewBasePanel("1 Files", "changes"),
		repo:      repo,
		repoPath:  ".", // Default to current directory
	}
	p.loadFiles()
	return p
}

// SetRepoPath sets the repository path for CLI operations
func (p *FilesPanel) SetRepoPath(path string) {
	p.repoPath = path
}

func (p *FilesPanel) loadFiles() {
	// Get file changes from jj-lib
	changes, err := p.repo.WorkingCopyChanges()
	if err != nil {
		p.files = nil
		p.loadErr = err
		return
	}
	p.loadErr = nil

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

// LoadForChange loads files changed in a specific change ID
func (p *FilesPanel) LoadForChange(changeID string) {
	cliFiles, err := jj.FilesForChange(p.repoPath, changeID)
	if err != nil {
		p.files = nil
		p.loadErr = err
		return
	}
	p.loadErr = nil

	p.files = make([]fixtures.FileChange, len(cliFiles))
	for i, cf := range cliFiles {
		var status fixtures.FileStatus
		switch cf.Status {
		case "A":
			status = fixtures.StatusAdded
		case "D":
			status = fixtures.StatusDeleted
		case "M":
			status = fixtures.StatusModified
		default:
			status = fixtures.StatusModified
		}
		p.files[i] = fixtures.FileChange{
			Path:   cf.Path,
			Status: status,
		}
	}

	p.cursor = 0
	if p.ready {
		p.viewport.SetContent(p.renderContent())
		p.viewport.GotoTop()
	}
}

func (p *FilesPanel) Init() tea.Cmd {
	return nil
}

func (p *FilesPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	prevCursor := p.cursor

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft {
			p.HandleMouseClick(msg, &p.viewport, len(p.files))
		} else {
			p.HandleMouseScroll(msg, &p.viewport)
		}

	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		p.HandleCursorKeys(msg, len(p.files), &p.viewport)
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

func (p *FilesPanel) View() string {
	if !p.ready {
		return p.RenderFrame("Loading...")
	}
	if len(p.files) == 0 {
		if p.loadErr != nil {
			return p.RenderFrame(theme.DimmedStyle.Render("Error: " + p.loadErr.Error()))
		}
		return p.RenderFrame(theme.DimmedStyle.Render("No files changed"))
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

		if i == p.cursor {
			// Always show selected item in yellow (tracked cursor)
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
