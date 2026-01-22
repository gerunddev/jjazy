package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerunddev/jjazy/jj"
	"github.com/gerunddev/jjazy/ui/borders"
)

// ANSI escape codes for selection highlighting
const (
	// Dark gray background (ANSI 256 color 238)
	selectionBgStart = "\x1b[48;5;238m"
	selectionBgEnd   = "\x1b[49m" // Reset background only
)

// LogPanel displays the jj log with CLI-style output and selection.
type LogPanel struct {
	BasePanel
	repoPath      string
	viewport      viewport.Model
	logOutput     *jj.LogOutput
	selectedIndex int // Index into logOutput.Changes
	ready         bool
}

// NewLogPanel creates a new log panel.
func NewLogPanel(repoPath string) *LogPanel {
	l := &LogPanel{
		BasePanel: NewBasePanel("0 Log", "log"),
		repoPath:  repoPath,
	}
	l.loadLog()
	return l
}

// SetTitle changes the panel title
func (l *LogPanel) SetTitle(title string) {
	l.title = title
}

func (l *LogPanel) loadLog() {
	output, err := jj.LogCLI(l.repoPath)
	if err != nil {
		// Create empty output on error
		l.logOutput = &jj.LogOutput{
			RawANSI:      "Error loading log: " + err.Error(),
			LineToChange: []string{},
			Changes:      []jj.ChangeInfo{},
		}
		return
	}
	l.logOutput = output

	// Ensure selected index is valid
	if l.selectedIndex >= len(l.logOutput.Changes) {
		l.selectedIndex = 0
	}
}

// Refresh reloads the log from the CLI.
func (l *LogPanel) Refresh() {
	l.loadLog()
	if l.ready {
		l.viewport.SetContent(l.renderLog())
	}
}

// SelectedChange returns the currently selected change, or nil if none.
func (l *LogPanel) SelectedChange() *jj.ChangeInfo {
	if l.logOutput == nil || len(l.logOutput.Changes) == 0 {
		return nil
	}
	if l.selectedIndex < 0 || l.selectedIndex >= len(l.logOutput.Changes) {
		return nil
	}
	return &l.logOutput.Changes[l.selectedIndex]
}

// GetChanges returns all changes in the log
func (l *LogPanel) GetChanges() []jj.ChangeInfo {
	if l.logOutput == nil {
		return nil
	}
	return l.logOutput.Changes
}

// SelectByChangeID selects the change with the given change ID
func (l *LogPanel) SelectByChangeID(changeID string) {
	if l.logOutput == nil {
		return
	}
	for i, change := range l.logOutput.Changes {
		if change.ChangeID == changeID {
			l.selectedIndex = i
			l.ensureSelectedVisible()
			if l.ready {
				l.viewport.SetContent(l.renderLog())
			}
			return
		}
	}
}

func (l *LogPanel) Init() tea.Cmd {
	return nil
}

func (l *LogPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Handle scroll wheel
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			l.viewport.LineUp(3)
		case tea.MouseButtonWheelDown:
			l.viewport.LineDown(3)
		}

	case tea.KeyMsg:
		if l.focused {
			changeCount := len(l.logOutput.Changes)

			switch msg.String() {
			// Emacs-style navigation
			case "up", "ctrl+p":
				l.selectPrev()
			case "down", "ctrl+n":
				l.selectNext()
			case "alt+v": // Page up
				l.viewport.HalfViewUp()
			case "ctrl+v": // Page down
				l.viewport.HalfViewDown()
			case "alt+<", "home": // Beginning of buffer
				l.selectedIndex = 0
				l.ensureSelectedVisible()
			case "alt+>", "end": // End of buffer
				if changeCount > 0 {
					l.selectedIndex = changeCount - 1
				}
				l.ensureSelectedVisible()
			case "ctrl+l": // Center selected in viewport
				l.centerSelected()
			}

			// Re-render after selection change
			if l.ready && l.logOutput != nil {
				l.viewport.SetContent(l.renderLog())
			}
		}
	}

	l.viewport, cmd = l.viewport.Update(msg)
	return l, cmd
}

func (l *LogPanel) selectNext() {
	if l.logOutput == nil || len(l.logOutput.Changes) == 0 {
		return
	}
	if l.selectedIndex < len(l.logOutput.Changes)-1 {
		l.selectedIndex++
		l.ensureSelectedVisible()
	}
}

func (l *LogPanel) selectPrev() {
	if l.logOutput == nil || len(l.logOutput.Changes) == 0 {
		return
	}
	if l.selectedIndex > 0 {
		l.selectedIndex--
		l.ensureSelectedVisible()
	}
}

func (l *LogPanel) ensureSelectedVisible() {
	if l.logOutput == nil || len(l.logOutput.Changes) == 0 || !l.ready {
		return
	}

	change := l.logOutput.Changes[l.selectedIndex]
	viewTop := l.viewport.YOffset
	viewBottom := viewTop + l.viewport.Height

	// If selection is above viewport, scroll up
	if change.StartLine < viewTop {
		l.viewport.SetYOffset(change.StartLine)
	}

	// If selection is below viewport, scroll down
	if change.EndLine > viewBottom {
		// Scroll so the end of the change is at the bottom
		l.viewport.SetYOffset(change.EndLine - l.viewport.Height)
	}
}

func (l *LogPanel) centerSelected() {
	if l.logOutput == nil || len(l.logOutput.Changes) == 0 || !l.ready {
		return
	}

	change := l.logOutput.Changes[l.selectedIndex]
	// Center the start of the change in the viewport
	offset := change.StartLine - l.viewport.Height/2
	if offset < 0 {
		offset = 0
	}
	l.viewport.SetYOffset(offset)
}

func (l *LogPanel) View() string {
	if !l.ready {
		return l.RenderFrame("Loading log...")
	}

	return l.RenderFrame(l.viewport.View())
}

// SetSize resizes the panel and viewport.
func (l *LogPanel) SetSize(width, height int) {
	l.BasePanel.SetSize(width, height)

	contentWidth := l.ContentWidth()
	contentHeight := l.ContentHeight()

	if !l.ready {
		l.viewport = viewport.New(contentWidth, contentHeight)
		l.viewport.SetContent(l.renderLog())
		l.ready = true
	} else {
		l.viewport.Width = contentWidth
		l.viewport.Height = contentHeight
		l.viewport.SetContent(l.renderLog())
	}
}

// renderLog renders the log with selection highlighting.
func (l *LogPanel) renderLog() string {
	if l.logOutput == nil {
		return ""
	}

	lines := strings.Split(l.logOutput.RawANSI, "\n")
	var selectedChangeID string

	// Get the selected change ID
	if len(l.logOutput.Changes) > 0 && l.selectedIndex < len(l.logOutput.Changes) {
		selectedChangeID = l.logOutput.Changes[l.selectedIndex].ChangeID
	}

	// Apply selection highlighting
	var result []string
	for i, line := range lines {
		// Check if this line belongs to the selected change
		if i < len(l.logOutput.LineToChange) && l.logOutput.LineToChange[i] == selectedChangeID && selectedChangeID != "" {
			// Add background highlight, preserving existing ANSI codes
			line = selectionBgStart + line + selectionBgEnd
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// RenderFrame renders the panel with titled border.
func (l *LogPanel) RenderFrame(content string) string {
	title := l.title
	if l.ready && l.viewport.TotalLineCount() > l.viewport.Height {
		scrollPercent := int(l.viewport.ScrollPercent() * 100)
		title = fmt.Sprintf("%s (%d%%)", l.title, scrollPercent)
	}

	return borders.RenderTitledBorder(content, title, l.width, l.height, l.focused)
}

// Ensure LogPanel implements Panel
var _ Panel = (*LogPanel)(nil)
