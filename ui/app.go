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
	focusedPanel int // 0-3 for sidebar panels, 4 for diff
	keys         KeyMap
	help         help.Model
	width        int
	height       int
	ready        bool
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
		focusedPanel:    0,
		keys:            DefaultKeyMap(),
		help:            help.New(),
	}

	// Set initial focus
	app.statusPanel.SetFocused(true)

	return app
}

func (a *App) Init() tea.Cmd {
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

		case key.Matches(msg, a.keys.Panel1):
			a.setFocus(0)
			return a, nil

		case key.Matches(msg, a.keys.Panel2):
			a.setFocus(1)
			return a, nil

		case key.Matches(msg, a.keys.Panel3):
			a.setFocus(2)
			return a, nil

		case key.Matches(msg, a.keys.Panel4):
			a.setFocus(3)
			return a, nil

		case key.Matches(msg, a.keys.NextPanel):
			a.setFocus((a.focusedPanel + 1) % 5)
			return a, nil

		case key.Matches(msg, a.keys.PrevPanel):
			a.setFocus((a.focusedPanel + 4) % 5)
			return a, nil
		}

		// Route to focused panel
		var cmd tea.Cmd
		switch a.focusedPanel {
		case 0:
			_, cmd = a.statusPanel.Update(msg)
		case 1:
			_, cmd = a.filesPanel.Update(msg)
		case 2:
			_, cmd = a.bookmarksPanel.Update(msg)
		case 3:
			_, cmd = a.operationsPanel.Update(msg)
		case 4:
			_, cmd = a.diffViewer.Update(msg)
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
	a.statusPanel.SetFocused(false)
	a.filesPanel.SetFocused(false)
	a.bookmarksPanel.SetFocused(false)
	a.operationsPanel.SetFocused(false)
	a.diffViewer.SetFocused(false)

	// Set new focus
	a.focusedPanel = panel
	switch panel {
	case 0:
		a.statusPanel.SetFocused(true)
	case 1:
		a.filesPanel.SetFocused(true)
	case 2:
		a.bookmarksPanel.SetFocused(true)
	case 3:
		a.operationsPanel.SetFocused(true)
	case 4:
		a.diffViewer.SetFocused(true)
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

	// Divide sidebar height among 4 panels
	panelHeight := contentHeight / 4
	lastPanelHeight := contentHeight - (panelHeight * 3)

	// Set panel sizes
	a.statusPanel.SetSize(sidebarWidth, panelHeight)
	a.filesPanel.SetSize(sidebarWidth, panelHeight)
	a.bookmarksPanel.SetSize(sidebarWidth, panelHeight)
	a.operationsPanel.SetSize(sidebarWidth, lastPanelHeight)
	a.diffViewer.SetSize(diffWidth, contentHeight)

	// Set log overlay size (centered, 80% width, 60% height)
	logWidth := min(FloatingLogWidth, a.width*8/10)
	logHeight := min(FloatingLogHeight, a.height*6/10)
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
	logView := a.logOverlay.View()

	// Get dimensions
	logWidth := lipgloss.Width(logView)
	logHeight := lipgloss.Height(logView)

	// Calculate center position
	x := (a.width - logWidth) / 2
	y := (a.height - logHeight) / 2

	// Split background into lines
	bgLines := strings.Split(background, "\n")

	// Split log into lines
	logLines := strings.Split(logView, "\n")

	// Overlay log onto background
	for i, logLine := range logLines {
		bgY := y + i
		if bgY >= 0 && bgY < len(bgLines) {
			bgLine := bgLines[bgY]
			// Pad background line if needed
			for len(bgLine) < a.width {
				bgLine += " "
			}

			// Convert to runes for proper handling
			bgRunes := []rune(bgLine)
			logRunes := []rune(logLine)

			// Insert log line at x position
			for j, r := range logRunes {
				pos := x + j
				if pos >= 0 && pos < len(bgRunes) {
					bgRunes[pos] = r
				}
			}

			bgLines[bgY] = string(bgRunes)
		}
	}

	return strings.Join(bgLines, "\n")
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
