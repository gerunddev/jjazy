package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jjazy/app"
	"github.com/gerund/jjazy/jj"
	"github.com/gerund/jjazy/ui/floating"
	"github.com/gerund/jjazy/ui/messages"
	"github.com/gerund/jjazy/ui/panels"
)

// Experience represents the current view mode of the application
type Experience int

const (
	ExperienceLog    Experience = iota // Main log view
	ExperienceChange                   // Change detail view (files + diff)
)

// PanelBound defines the screen coordinates of a panel for mouse detection
type PanelBound struct {
	X1, Y1, X2, Y2 int
	PanelIndex     int
}

// App is the main application model
type App struct {
	// Repository
	repo     *jj.Repo
	repoPath string

	// Experience state
	currentExperience Experience
	selectedChangeID  string // Change ID being viewed in ExperienceChange

	// Panels - Log Experience (Exp 1)
	workspacePanel *panels.WorkspacePanel
	bookmarksPanel *panels.BookmarksPanel
	logPanel       *panels.LogPanel

	// Panels - Change Experience (Exp 2)
	filesPanel *panels.FilesPanel
	diffPanel  *panels.DiffViewer

	// Floating windows
	helpOverlay *floating.HelpOverlay
	showHelp    bool

	// State
	focusedPanel int // Experience-relative: 0=main, 1=sidebar1, 2=sidebar2
	keys         KeyMap
	help         help.Model
	width        int
	height       int
	ready        bool

	// Panel bounds for mouse coordinate mapping
	panelBounds []PanelBound
}

// NewApp creates a new application
func NewApp(repo *jj.Repo, repoPath string) *App {
	keys := DefaultKeyMap()

	// Create panels
	filesPanel := panels.NewFilesPanel(repo)
	filesPanel.SetRepoPath(repoPath)

	diffPanel := panels.NewDiffViewer(repo)
	diffPanel.SetRepoPath(repoPath)

	app := &App{
		repo:              repo,
		repoPath:          repoPath,
		currentExperience: ExperienceLog,
		// Log Experience panels
		workspacePanel: panels.NewWorkspacePanel(repo),
		bookmarksPanel: panels.NewBookmarksPanel(repo, repoPath),
		logPanel:       panels.NewLogPanel(repoPath),
		// Change Experience panels
		filesPanel:   filesPanel,
		diffPanel:    diffPanel,
		helpOverlay:  floating.NewHelpOverlay(&keys),
		focusedPanel: 0, // Main panel (log in Exp1, diff in Exp2)
		keys:         keys,
		help:         help.New(),
	}

	// Set initial focus to Log panel
	app.logPanel.SetFocused(true)

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

	case tea.MouseMsg:
		return a.handleMouse(msg)

	case messages.FileSelectedMsg:
		// When a file is selected in Change experience, update the diff view
		if a.currentExperience == ExperienceChange && msg.Path != "" {
			a.diffPanel.LoadFileInChange(a.selectedChangeID, msg.Path)
		}
		return a, nil

	case tea.KeyMsg:
		// Handle floating help first if visible
		if a.showHelp {
			switch {
			case key.Matches(msg, a.keys.Escape), key.Matches(msg, a.keys.Help):
				a.showHelp = false
				return a, nil
			case key.Matches(msg, a.keys.Quit):
				return a, tea.Quit
			default:
				_, cmd := a.helpOverlay.Update(msg)
				return a, cmd
			}
		}

		// Global keys
		switch {
		case key.Matches(msg, a.keys.Quit):
			return a, tea.Quit

		case key.Matches(msg, a.keys.Help):
			a.showHelp = true
			return a, nil

		case key.Matches(msg, a.keys.Escape),
			msg.Type == tea.KeyEscape,
			msg.Type == tea.KeyEsc,
			msg.String() == "esc",
			msg.String() == "escape",
			msg.String() == "ctrl+g",
			msg.Type == tea.KeyCtrlG:
			// First check if we need to exit cursor mode in browsable panels
			if a.currentExperience == ExperienceLog && a.focusedPanel == 1 && a.workspacePanel.IsEntered() {
				a.workspacePanel.SetEntered(false)
				return a, nil
			}
			if a.currentExperience == ExperienceLog && a.focusedPanel == 2 && a.bookmarksPanel.IsEntered() {
				a.bookmarksPanel.SetEntered(false)
				return a, nil
			}
			// Go back to previous experience
			if a.currentExperience == ExperienceChange {
				a.exitChangeExperience()
			}
			return a, nil

		case msg.Type == tea.KeyLeft, msg.String() == "left":
			// Left arrow navigation
			// First check if we need to exit entered mode
			if a.currentExperience == ExperienceLog && a.focusedPanel == 1 && a.workspacePanel.IsEntered() {
				a.workspacePanel.SetEntered(false)
				return a, nil
			}
			if a.currentExperience == ExperienceLog && a.focusedPanel == 2 && a.bookmarksPanel.IsEntered() {
				a.bookmarksPanel.SetEntered(false)
				return a, nil
			}
			if a.currentExperience == ExperienceChange {
				// Go back to Log experience
				a.exitChangeExperience()
				return a, nil
			} else if a.currentExperience == ExperienceLog && a.focusedPanel == 0 {
				// From Log panel → Workspace
				a.setFocus(1)
				return a, nil
			}

		case msg.Type == tea.KeyRight, msg.String() == "right":
			// Right arrow navigation
			// First check if we need to exit entered mode and move to log
			if a.currentExperience == ExperienceLog && a.focusedPanel == 1 && a.workspacePanel.IsEntered() {
				a.workspacePanel.SetEntered(false)
				a.setFocus(0) // Move to log
				return a, nil
			}
			if a.currentExperience == ExperienceLog && a.focusedPanel == 2 && a.bookmarksPanel.IsEntered() {
				a.bookmarksPanel.SetEntered(false)
				a.setFocus(0) // Move to log
				return a, nil
			}
			if a.currentExperience == ExperienceLog {
				if a.focusedPanel == 0 {
					// From Log panel → drill into change
					if change := a.logPanel.SelectedChange(); change != nil {
						a.enterChangeExperience(change.ChangeID)
						return a, nil
					}
				} else if a.focusedPanel == 1 || a.focusedPanel == 2 {
					// From Workspace or Bookmarks → Log panel
					a.setFocus(0)
					return a, nil
				}
			}

		case key.Matches(msg, a.keys.Enter):
			if a.currentExperience == ExperienceLog {
				switch a.focusedPanel {
				case 0: // Log panel - jj edit
					if change := a.logPanel.SelectedChange(); change != nil {
						_ = jj.Edit(a.repoPath, change.ChangeID)
						a.logPanel.Refresh()
						a.workspacePanel.Refresh()
						a.bookmarksPanel.Refresh()
					}
					return a, nil
				case 1: // Workspace panel
					if !a.workspacePanel.IsEntered() {
						a.workspacePanel.SetEntered(true)
					} else if ws := a.workspacePanel.SelectedWorkspace(); ws != nil {
						_ = jj.WorkspaceSwitch(a.repoPath, ws.Name)
						a.workspacePanel.SetEntered(false)
						a.setFocus(0) // Return to log view
						a.logPanel.Refresh()
						a.workspacePanel.Refresh()
						a.bookmarksPanel.Refresh()
					}
					return a, nil
				case 2: // Bookmarks panel
					if !a.bookmarksPanel.IsEntered() {
						a.bookmarksPanel.SetEntered(true)
					} else if bm := a.bookmarksPanel.SelectedBookmark(); bm != nil {
						// Use Navigation to find edit target (tip of branch or boundary)
						if revisions, err := a.repo.Log(); err == nil {
							nav := app.NewNavigation(a.repoPath, revisions)
							if target := nav.FindBookmarkEditTarget(bm.Name); target != nil {
								_ = nav.EditRevision(target.ChangeID)
							}
						}
						a.bookmarksPanel.SetEntered(false)
						a.setFocus(0) // Return to log view
						a.logPanel.Refresh()
						a.workspacePanel.Refresh()
						a.bookmarksPanel.Refresh()
					}
					return a, nil
				}
			}

		case msg.Type == tea.KeyDown, msg.String() == "down":
			// Down from Workspace → Bookmarks (in Log experience)
			// Only move between panels if not in entered mode
			if a.currentExperience == ExperienceLog && a.focusedPanel == 1 && !a.workspacePanel.IsEntered() {
				a.setFocus(2)
				return a, nil
			}

		case msg.Type == tea.KeyUp, msg.String() == "up":
			// Up from Bookmarks → Workspace (in Log experience)
			// Only move between panels if not in entered mode
			if a.currentExperience == ExperienceLog && a.focusedPanel == 2 && !a.bookmarksPanel.IsEntered() {
				a.setFocus(1)
				return a, nil
			}

		case key.Matches(msg, a.keys.Panel0):
			a.setFocus(0) // Main panel (log or diff)
			return a, nil

		case key.Matches(msg, a.keys.Panel1):
			a.setFocus(1) // First sidebar panel
			return a, nil

		case key.Matches(msg, a.keys.Panel2):
			// Only valid in Log experience (bookmarks)
			if a.currentExperience == ExperienceLog {
				a.setFocus(2)
			}
			return a, nil

		case key.Matches(msg, a.keys.NextPanel):
			// If in cursor mode in a browsable panel, exit to focus mode
			if a.currentExperience == ExperienceLog && a.focusedPanel == 1 && a.workspacePanel.IsEntered() {
				a.workspacePanel.SetEntered(false)
				return a, nil
			}
			if a.currentExperience == ExperienceLog && a.focusedPanel == 2 && a.bookmarksPanel.IsEntered() {
				a.bookmarksPanel.SetEntered(false)
				return a, nil
			}
			maxPanels := a.maxPanelsForExperience()
			a.setFocus((a.focusedPanel + 1) % maxPanels)
			return a, nil

		case key.Matches(msg, a.keys.PrevPanel):
			// If in cursor mode in a browsable panel, exit to focus mode
			if a.currentExperience == ExperienceLog && a.focusedPanel == 1 && a.workspacePanel.IsEntered() {
				a.workspacePanel.SetEntered(false)
				return a, nil
			}
			if a.currentExperience == ExperienceLog && a.focusedPanel == 2 && a.bookmarksPanel.IsEntered() {
				a.bookmarksPanel.SetEntered(false)
				return a, nil
			}
			maxPanels := a.maxPanelsForExperience()
			a.setFocus((a.focusedPanel + maxPanels - 1) % maxPanels)
			return a, nil
		}

		// Route to focused panel based on current experience
		var cmd tea.Cmd
		switch a.currentExperience {
		case ExperienceLog:
			switch a.focusedPanel {
			case 0:
				_, cmd = a.logPanel.Update(msg)
			case 1:
				_, cmd = a.workspacePanel.Update(msg)
			case 2:
				_, cmd = a.bookmarksPanel.Update(msg)
			}
		case ExperienceChange:
			switch a.focusedPanel {
			case 0:
				_, cmd = a.diffPanel.Update(msg)
			case 1:
				_, cmd = a.filesPanel.Update(msg)
			}
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

	var sidebar, mainPanel string

	// Build layout based on current experience
	switch a.currentExperience {
	case ExperienceLog:
		// Log experience: Workspace + Bookmarks sidebar, Log main
		sidebar = lipgloss.JoinVertical(lipgloss.Left,
			a.workspacePanel.View(),
			a.bookmarksPanel.View(),
		)
		mainPanel = a.logPanel.View()

	case ExperienceChange:
		// Change experience: Files sidebar, Diff main
		sidebar = a.filesPanel.View()
		mainPanel = a.diffPanel.View()
	}

	// Build main layout
	main := lipgloss.JoinHorizontal(lipgloss.Top,
		sidebar,
		mainPanel,
	)

	// Add space at top of main content
	mainWithSpacing := "\n" + main

	// Wrap main in border with breadcrumb tabs
	borderedMain := a.renderMainFrame(mainWithSpacing)

	// Build help bar
	helpBar := a.renderHelpBar()

	// Combine bordered main + help
	fullView := lipgloss.JoinVertical(lipgloss.Left, borderedMain, helpBar)

	// Overlay floating help if visible
	if a.showHelp {
		fullView = a.overlayHelp(fullView)
	}

	return fullView
}

// clearAllFocus clears focus from all panels
func (a *App) clearAllFocus() {
	a.logPanel.SetFocused(false)
	a.workspacePanel.SetFocused(false)
	a.bookmarksPanel.SetFocused(false)
	a.filesPanel.SetFocused(false)
	a.diffPanel.SetFocused(false)
}

// setFocusForExperience sets default focus for the current experience
func (a *App) setFocusForExperience() {
	a.clearAllFocus()

	switch a.currentExperience {
	case ExperienceLog:
		a.focusedPanel = 0
		a.logPanel.SetFocused(true)
	case ExperienceChange:
		a.focusedPanel = 1 // Files panel is default focus
		a.filesPanel.SetFocused(true)
	}
}

// maxPanelsForExperience returns the number of panels in the current experience
func (a *App) maxPanelsForExperience() int {
	switch a.currentExperience {
	case ExperienceLog:
		return 3 // log, workspace, bookmarks
	case ExperienceChange:
		return 2 // diff, files
	}
	return 3
}

func (a *App) setFocus(panel int) {
	a.clearAllFocus()

	// Clear entered state when changing focus
	a.workspacePanel.SetEntered(false)
	a.bookmarksPanel.SetEntered(false)

	// Clamp panel index to valid range for current experience
	maxPanels := a.maxPanelsForExperience()
	if panel >= maxPanels {
		panel = maxPanels - 1
	}
	if panel < 0 {
		panel = 0
	}

	a.focusedPanel = panel

	switch a.currentExperience {
	case ExperienceLog:
		// 0=log, 1=workspace, 2=bookmarks
		switch panel {
		case 0:
			a.logPanel.SetFocused(true)
		case 1:
			a.workspacePanel.SetFocused(true)
		case 2:
			a.bookmarksPanel.SetFocused(true)
		}
	case ExperienceChange:
		// 0=diff, 1=files
		switch panel {
		case 0:
			a.diffPanel.SetFocused(true)
		case 1:
			a.filesPanel.SetFocused(true)
		}
	}
}

// enterChangeExperience transitions to the Change experience for a specific change
func (a *App) enterChangeExperience(changeID string) {
	a.currentExperience = ExperienceChange
	a.selectedChangeID = changeID

	// Set focus first so files panel renders with correct highlight
	a.setFocusForExperience()

	// Load files for this change (will render with first file highlighted)
	a.filesPanel.LoadForChange(changeID)

	// Load diff for the first file (if any), otherwise load full change diff
	if file := a.filesPanel.SelectedFile(); file != nil {
		a.diffPanel.LoadFileInChange(changeID, file.Path)
	} else {
		a.diffPanel.LoadChange(changeID)
	}

	// Recalculate layout for new experience
	a.updateLayout()
}

// exitChangeExperience returns to the Log experience
func (a *App) exitChangeExperience() {
	a.currentExperience = ExperienceLog
	a.selectedChangeID = ""

	// Recalculate layout for new experience
	a.updateLayout()

	// Set focus to log panel
	a.setFocusForExperience()
}

func (a *App) updateLayout() {
	// Calculate dimensions
	// Account for outer border (2 chars width, 2 chars height), help bar (1 line), and top spacing (1 line)
	availableWidth := a.width - 2  // Border takes 2 chars
	availableHeight := a.height - 4 // Border (2) + help bar (1) + top spacing (1)

	sidebarWidth := SidebarWidth
	if a.width < 100 {
		sidebarWidth = SidebarMinWidth
	} else if a.width > 200 {
		sidebarWidth = SidebarMaxWidth
	}

	mainWidth := availableWidth - sidebarWidth
	contentHeight := availableHeight

	switch a.currentExperience {
	case ExperienceLog:
		// Log Experience: Workspace + Bookmarks sidebar, Log main
		workspaceHeight := 3
		bookmarksHeight := contentHeight - workspaceHeight
		if bookmarksHeight < 3 {
			bookmarksHeight = 3
		}

		a.workspacePanel.SetSize(sidebarWidth, workspaceHeight)
		a.bookmarksPanel.SetSize(sidebarWidth, bookmarksHeight)
		a.logPanel.SetSize(mainWidth, contentHeight)

		// Panel bounds: 0=log, 1=workspace, 2=bookmarks
		a.panelBounds = []PanelBound{
			{X1: sidebarWidth, Y1: 0, X2: a.width - 1, Y2: contentHeight - 1, PanelIndex: 0},                    // Log
			{X1: 0, Y1: 0, X2: sidebarWidth - 1, Y2: workspaceHeight - 1, PanelIndex: 1},                        // Workspace
			{X1: 0, Y1: workspaceHeight, X2: sidebarWidth - 1, Y2: contentHeight - 1, PanelIndex: 2},            // Bookmarks
		}

	case ExperienceChange:
		// Change Experience: Files sidebar, Diff main
		a.filesPanel.SetSize(sidebarWidth, contentHeight)
		a.diffPanel.SetSize(mainWidth, contentHeight)

		// Panel bounds: 0=diff, 1=files
		a.panelBounds = []PanelBound{
			{X1: sidebarWidth, Y1: 0, X2: a.width - 1, Y2: contentHeight - 1, PanelIndex: 0}, // Diff
			{X1: 0, Y1: 0, X2: sidebarWidth - 1, Y2: contentHeight - 1, PanelIndex: 1},       // Files
		}
	}

	// Set overlay sizes to full screen
	overlayWidth := a.width
	overlayHeight := a.height - 1 // Leave room for help bar (1 line)
	a.helpOverlay.SetSize(overlayWidth, overlayHeight)
}

// renderBreadcrumbs builds the styled breadcrumb tabs based on current experience
func (a *App) renderBreadcrumbs() string {
	// Get the folder name from the repo path
	folderName := filepath.Base(a.repoPath)
	if folderName == "." || folderName == "" {
		if absPath, err := filepath.Abs(a.repoPath); err == nil {
			folderName = filepath.Base(absPath)
		} else {
			folderName = "repo"
		}
	}

	// Orange tab style for folder name
	orangeTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FCFCFA")). // White text
		Background(lipgloss.Color("#FC9867")). // Monokai orange
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1)

	folderTab := orangeTabStyle.Render(" " + folderName + " ")

	if a.currentExperience == ExperienceLog {
		return folderTab
	}

	// Blue tab style for change ID (Change experience)
	blueTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FCFCFA")). // White text
		Background(lipgloss.Color("#78DCE8")). // Monokai blue/cyan
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1)

	changeTab := blueTabStyle.Render(" " + a.selectedChangeID + " ")

	return folderTab + " " + changeTab
}

func (a *App) renderMainFrame(content string) string {
	// Create border style (similar to help overlay)
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#939293")). // Dimmed color for border
		Width(a.width - 2).                          // Account for border width
		Height(a.height - 3)                         // Account for border height and help bar

	// Render content with border
	bordered := borderStyle.Render(content)

	// Add breadcrumb tabs to top border
	lines := strings.Split(bordered, "\n")
	if len(lines) > 0 {
		borderColorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#939293"))

		styledBreadcrumbs := a.renderBreadcrumbs()
		breadcrumbWidth := lipgloss.Width(styledBreadcrumbs)
		remainingWidth := a.width - 3 - breadcrumbWidth
		if remainingWidth < 0 {
			remainingWidth = 0
		}

		topBorder := borderColorStyle.Render("╭─") +
			styledBreadcrumbs +
			borderColorStyle.Render(strings.Repeat("─", remainingWidth)+"╮")

		lines[0] = topBorder
	}

	return strings.Join(lines, "\n")
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

func (a *App) overlayHelp(background string) string {
	// Render help overlay at full screen
	helpView := a.helpOverlay.View()

	// Replace background with help view (full overlay)
	bgLines := strings.Split(background, "\n")
	helpLines := strings.Split(helpView, "\n")

	// Replace background lines with help lines starting from the top
	for i, helpLine := range helpLines {
		if i >= 0 && i < len(bgLines) {
			bgLines[i] = helpLine
		}
	}

	return strings.Join(bgLines, "\n")
}

// handleMouse processes mouse events for panel focus and interaction
func (a *App) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// If help overlay is visible, handle mouse there first
	if a.showHelp {
		// Check if click is outside help overlay to dismiss it
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// For now, any click while help is visible dismisses it
			// Could be improved to check if click is inside overlay
			a.showHelp = false
			return a, nil
		}
		// Forward scroll events to help overlay
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			_, cmd := a.helpOverlay.Update(msg)
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
	// Adjust coordinates to be panel-relative
	for _, bound := range a.panelBounds {
		if bound.PanelIndex == panelIndex {
			msg.Y = msg.Y - bound.Y1
			msg.X = msg.X - bound.X1
			break
		}
	}

	var cmd tea.Cmd
	switch a.currentExperience {
	case ExperienceLog:
		switch panelIndex {
		case 0:
			_, cmd = a.logPanel.Update(msg)
		case 1:
			_, cmd = a.workspacePanel.Update(msg)
		case 2:
			_, cmd = a.bookmarksPanel.Update(msg)
		}
	case ExperienceChange:
		switch panelIndex {
		case 0:
			_, cmd = a.diffPanel.Update(msg)
		case 1:
			_, cmd = a.filesPanel.Update(msg)
		}
	}
	return a, cmd
}
