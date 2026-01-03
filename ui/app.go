package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jayz/jj"
	"github.com/gerund/jayz/ui/floating"
	"github.com/gerund/jayz/ui/messages"
	"github.com/gerund/jayz/ui/panels"
)

// PanelBound defines the screen coordinates of a panel for mouse detection
type PanelBound struct {
	X1, Y1, X2, Y2 int
	PanelIndex     int
}

// App is the main application model
type App struct {
	// Repository
	repo *jj.Repo

	// Panels
	statusPanel     *panels.StatusPanel
	filesPanel      *panels.FilesPanel
	bookmarksPanel  *panels.BookmarksPanel
	operationsPanel *panels.OperationsPanel
	diffViewer      *panels.DiffViewer

	// Floating windows
	logOverlay *floating.LogOverlay
	showLog    bool

	// State
	focusedPanel int // 0=diff, 1=status, 2=files, 3=bookmarks, 4=operations
	keys         KeyMap
	help         help.Model
	width        int
	height       int
	ready        bool

	// Panel bounds for mouse coordinate mapping
	panelBounds []PanelBound
}

// NewApp creates a new application
func NewApp(repo *jj.Repo) *App {
	app := &App{
		repo:            repo,
		statusPanel:     panels.NewStatusPanel(repo),
		filesPanel:      panels.NewFilesPanel(repo),
		bookmarksPanel:  panels.NewBookmarksPanel(repo),
		operationsPanel: panels.NewOperationsPanel(repo),
		diffViewer:      panels.NewDiffViewer(repo),
		logOverlay:      floating.NewLogOverlay(repo),
		focusedPanel:    2, // Files panel
		keys:            DefaultKeyMap(),
		help:            help.New(),
	}

	// Set initial focus to Files panel
	app.filesPanel.SetFocused(true)

	return app
}

func (a *App) Init() tea.Cmd {
	// Fetch initial diff for the first selected file
	if file := a.filesPanel.SelectedFile(); file != nil {
		return a.fetchFileDiff(file.Path)
	}
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateLayout()
		a.ready = true
		return a, nil

	case messages.FileSelectedMsg:
		// Fetch diff for selected file
		return a, a.fetchFileDiff(msg.Path)

	case messages.RevisionSelectedMsg:
		// Fetch diff for selected revision
		return a, a.fetchRevisionDiff(msg.RevisionID)

	case messages.DiffContentMsg:
		// Update DiffViewer with new content
		a.diffViewer.SetContent(msg.Content)
		return a, nil

	case tea.MouseMsg:
		return a.handleMouse(msg)

	case tea.KeyMsg:
		// Handle floating log first if visible
		if a.showLog {
			switch {
			case key.Matches(msg, a.keys.Escape), key.Matches(msg, a.keys.ToggleLog):
				a.showLog = false
				return a, nil
			case key.Matches(msg, a.keys.Quit):
				return a, tea.Quit
			default:
				_, cmd := a.logOverlay.Update(msg)
				return a, cmd
			}
		}

		// Global keys
		switch {
		case key.Matches(msg, a.keys.Quit):
			return a, tea.Quit

		case key.Matches(msg, a.keys.ToggleLog):
			a.showLog = true
			return a, nil

		case key.Matches(msg, a.keys.Help):
			a.help.ShowAll = !a.help.ShowAll
			return a, nil

		case key.Matches(msg, a.keys.Panel0):
			a.setFocus(0) // Diff panel
			return a, nil

		case key.Matches(msg, a.keys.Panel1):
			a.setFocus(1) // Status panel
			return a, nil

		case key.Matches(msg, a.keys.Panel2):
			a.setFocus(2) // Files panel
			return a, nil

		case key.Matches(msg, a.keys.Panel3):
			a.setFocus(3) // Bookmarks panel
			return a, nil

		case key.Matches(msg, a.keys.Panel4):
			a.setFocus(4) // Operations panel
			return a, nil

		case key.Matches(msg, a.keys.NextPanel):
			a.setFocus((a.focusedPanel + 1) % 5)
			return a, nil

		case key.Matches(msg, a.keys.PrevPanel):
			a.setFocus((a.focusedPanel + 4) % 5)
			return a, nil
		}

		// Route to focused panel (0=diff, 1=status, 2=files, 3=bookmarks, 4=operations)
		var cmd tea.Cmd
		switch a.focusedPanel {
		case 0:
			_, cmd = a.diffViewer.Update(msg)
		case 1:
			_, cmd = a.statusPanel.Update(msg)
		case 2:
			_, cmd = a.filesPanel.Update(msg)
		case 3:
			_, cmd = a.bookmarksPanel.Update(msg)
		case 4:
			_, cmd = a.operationsPanel.Update(msg)
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return a, tea.Batch(cmds...)
}

func (a *App) View() string {
	if !a.ready {
		return "Initializing..."
	}

	// Build sidebar (stacked panels)
	sidebar := lipgloss.JoinVertical(lipgloss.Left,
		a.statusPanel.View(),
		a.filesPanel.View(),
		a.bookmarksPanel.View(),
		a.operationsPanel.View(),
	)

	// Build main layout
	main := lipgloss.JoinHorizontal(lipgloss.Top,
		sidebar,
		a.diffViewer.View(),
	)

	// Build help bar
	helpBar := a.renderHelpBar()

	// Combine main + help
	fullView := lipgloss.JoinVertical(lipgloss.Left, main, helpBar)

	// Overlay floating log if visible
	if a.showLog {
		fullView = a.overlayLog(fullView)
	}

	return fullView
}

func (a *App) setFocus(panel int) {
	// Clear all focus
	a.diffViewer.SetFocused(false)
	a.statusPanel.SetFocused(false)
	a.filesPanel.SetFocused(false)
	a.bookmarksPanel.SetFocused(false)
	a.operationsPanel.SetFocused(false)

	// Set new focus: 0=diff, 1=status, 2=files, 3=bookmarks, 4=operations
	a.focusedPanel = panel
	switch panel {
	case 0:
		a.diffViewer.SetFocused(true)
	case 1:
		a.statusPanel.SetFocused(true)
	case 2:
		a.filesPanel.SetFocused(true)
	case 3:
		a.bookmarksPanel.SetFocused(true)
	case 4:
		a.operationsPanel.SetFocused(true)
	}
}

func (a *App) updateLayout() {
	// Calculate dimensions
	sidebarWidth := SidebarWidth
	if a.width < 100 {
		sidebarWidth = SidebarMinWidth
	} else if a.width > 200 {
		sidebarWidth = SidebarMaxWidth
	}

	diffWidth := a.width - sidebarWidth
	contentHeight := a.height - 1 // Leave room for help bar

	// Smart dynamic panel heights - allocate space based on content needs:
	// 1. Status: always 1 content line + 2 borders = 3 total
	statusHeight := 3
	remainingHeight := contentHeight - statusHeight

	// 2. Files: take as much as needed (up to remaining space)
	filesCount := a.filesPanel.Count()
	filesContentLines := min(filesCount, remainingHeight-2) // Need room for borders
	if filesContentLines < 1 {
		filesContentLines = 1 // Minimum 1 line
	}
	filesHeight := filesContentLines + 2
	remainingHeight -= filesHeight

	// 3. Bookmarks: take as much as needed (up to remaining space)
	bookmarkCount := a.bookmarksPanel.Count()
	bookmarksContentLines := min(bookmarkCount, remainingHeight-2)
	if bookmarksContentLines < 1 {
		bookmarksContentLines = 1
	}
	bookmarksHeight := bookmarksContentLines + 2
	remainingHeight -= bookmarksHeight

	// 4. Operations: take remaining space (or as much as needed)
	operationsCount := a.operationsPanel.Count()
	operationsContentLines := min(operationsCount, remainingHeight-2)
	if operationsContentLines < 1 {
		operationsContentLines = 1
	}
	operationsHeight := operationsContentLines + 2

	// Set panel sizes
	a.statusPanel.SetSize(sidebarWidth, statusHeight)
	a.filesPanel.SetSize(sidebarWidth, filesHeight)
	a.bookmarksPanel.SetSize(sidebarWidth, bookmarksHeight)
	a.operationsPanel.SetSize(sidebarWidth, operationsHeight)
	a.diffViewer.SetSize(diffWidth, contentHeight)

	// Calculate Y positions for panel bounds
	statusY := 0
	filesY := statusY + statusHeight
	bookmarksY := filesY + filesHeight
	operationsY := bookmarksY + bookmarksHeight

	// Calculate panel bounds for mouse detection
	// Panel indices: 0=diff, 1=status, 2=files, 3=bookmarks, 4=operations
	a.panelBounds = []PanelBound{
		{X1: sidebarWidth, Y1: 0, X2: a.width - 1, Y2: contentHeight - 1, PanelIndex: 0},                      // DiffViewer
		{X1: 0, Y1: statusY, X2: sidebarWidth - 1, Y2: statusY + statusHeight - 1, PanelIndex: 1},              // Status
		{X1: 0, Y1: filesY, X2: sidebarWidth - 1, Y2: filesY + filesHeight - 1, PanelIndex: 2},                 // Files
		{X1: 0, Y1: bookmarksY, X2: sidebarWidth - 1, Y2: bookmarksY + bookmarksHeight - 1, PanelIndex: 3},     // Bookmarks
		{X1: 0, Y1: operationsY, X2: sidebarWidth - 1, Y2: operationsY + operationsHeight - 1, PanelIndex: 4},  // Operations
	}

	// Set log overlay size to full screen
	logWidth := a.width
	logHeight := a.height - 1 // Leave room for help bar
	a.logOverlay.SetSize(logWidth, logHeight)
}

func (a *App) renderHelpBar() string {
	var items []string

	// Build help items
	bindings := a.keys.ShortHelp()
	for _, b := range bindings {
		keyStyle := HelpKeyStyle.Render(b.Help().Key)
		descStyle := HelpDescStyle.Render(b.Help().Desc)
		items = append(items, keyStyle+" "+descStyle)
	}

	helpText := strings.Join(items, "  ")
	return HelpBarStyle.Width(a.width).Render(helpText)
}

func (a *App) overlayLog(background string) string {
	// Render log overlay at full screen
	logView := a.logOverlay.View()

	// Replace background with log view (full overlay)
	bgLines := strings.Split(background, "\n")
	logLines := strings.Split(logView, "\n")

	// Replace background lines with log lines starting from the top
	for i, logLine := range logLines {
		if i >= 0 && i < len(bgLines) {
			bgLines[i] = logLine
		}
	}

	return strings.Join(bgLines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// fetchFileDiff fetches the diff for a specific file
func (a *App) fetchFileDiff(path string) tea.Cmd {
	return func() tea.Msg {
		if path == "" {
			return nil
		}
		diff, err := a.repo.FileDiff(path)
		if err != nil {
			diff = "Error: " + err.Error()
		}
		return messages.DiffContentMsg{Content: diff, Title: "Diff: " + path}
	}
}

// fetchRevisionDiff fetches the diff for a revision compared to its parent
func (a *App) fetchRevisionDiff(revisionID string) tea.Cmd {
	return func() tea.Msg {
		if revisionID == "" {
			return nil
		}
		diff, err := a.repo.RevisionDiff(revisionID)
		if err != nil {
			diff = "Error: " + err.Error()
		}
		return messages.DiffContentMsg{Content: diff, Title: "Diff: " + revisionID}
	}
}

// handleMouse processes mouse events for panel focus and interaction
func (a *App) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// If log overlay is visible, handle mouse there first
	if a.showLog {
		// Check if click is outside log overlay to dismiss it
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// For now, any click while log is visible dismisses it
			// Could be improved to check if click is inside overlay
			a.showLog = false
			return a, nil
		}
		// Forward scroll events to log overlay
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			_, cmd := a.logOverlay.Update(msg)
			return a, cmd
		}
		return a, nil
	}

	// Find which panel was clicked
	panelIndex := a.panelAtPoint(msg.X, msg.Y)

	switch msg.Button {
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionPress {
			// Focus the clicked panel
			if panelIndex >= 0 && panelIndex != a.focusedPanel {
				a.setFocus(panelIndex)
			}
			// Forward click to panel for item selection
			return a.forwardMouseToPanel(panelIndex, msg)
		}

	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		// Forward scroll to the panel under cursor
		return a.forwardMouseToPanel(panelIndex, msg)
	}

	return a, nil
}

// panelAtPoint returns the panel index at the given screen coordinates
func (a *App) panelAtPoint(x, y int) int {
	for _, bound := range a.panelBounds {
		if x >= bound.X1 && x <= bound.X2 && y >= bound.Y1 && y <= bound.Y2 {
			return bound.PanelIndex
		}
	}
	return -1
}

// forwardMouseToPanel forwards a mouse event to the appropriate panel
func (a *App) forwardMouseToPanel(panelIndex int, msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Adjust Y coordinate to be panel-relative
	for _, bound := range a.panelBounds {
		if bound.PanelIndex == panelIndex {
			msg.Y = msg.Y - bound.Y1
			msg.X = msg.X - bound.X1
			break
		}
	}

	var cmd tea.Cmd
	switch panelIndex {
	case 0:
		_, cmd = a.diffViewer.Update(msg)
	case 1:
		_, cmd = a.statusPanel.Update(msg)
	case 2:
		_, cmd = a.filesPanel.Update(msg)
	case 3:
		_, cmd = a.bookmarksPanel.Update(msg)
	case 4:
		_, cmd = a.operationsPanel.Update(msg)
	}
	return a, cmd
}
