package floating

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jjazy/ui/borders"
	"github.com/gerund/jjazy/ui/theme"
)

// HelpOverlay is a floating window showing help information
type HelpOverlay struct {
	viewport viewport.Model
	help     help.Model
	keymap   help.KeyMap
	width    int
	height   int
	ready    bool
}

// NewHelpOverlay creates a new floating help window
func NewHelpOverlay(keymap help.KeyMap) *HelpOverlay {
	return &HelpOverlay{
		help:   help.New(),
		keymap: keymap,
	}
}

func (h *HelpOverlay) Init() tea.Cmd {
	return nil
}

func (h *HelpOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Handle scroll wheel for viewport
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			h.viewport.LineUp(3)
		case tea.MouseButtonWheelDown:
			h.viewport.LineDown(3)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			h.viewport.LineUp(1)
		case "down", "j":
			h.viewport.LineDown(1)
		case "pgup", "ctrl+u":
			h.viewport.HalfViewUp()
		case "pgdown", "ctrl+d":
			h.viewport.HalfViewDown()
		case "g", "home":
			h.viewport.GotoTop()
		case "G", "end":
			h.viewport.GotoBottom()
		}
	}

	h.viewport, cmd = h.viewport.Update(msg)
	return h, cmd
}

func (h *HelpOverlay) View() string {
	if !h.ready {
		return h.renderFrame("Initializing...")
	}

	return h.renderFrame(h.viewport.View())
}

func (h *HelpOverlay) SetSize(width, height int) {
	h.width = width
	h.height = height

	// With titled borders: just top and bottom borders (title is in top border)
	contentWidth := width - 2
	contentHeight := height - 2

	if !h.ready {
		h.viewport = viewport.New(contentWidth, contentHeight)
		h.viewport.SetContent(h.renderHelp())
		h.ready = true
	} else {
		h.viewport.Width = contentWidth
		h.viewport.Height = contentHeight
		h.viewport.SetContent(h.renderHelp())
	}
}

func (h *HelpOverlay) renderHelp() string {
	var sections []string

	// Title section
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorBlue).
		MarginBottom(1)
	sections = append(sections, titleStyle.Render("JJazy - Jujutsu TUI"))

	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		MarginBottom(2)
	sections = append(sections, descStyle.Render("A terminal user interface for Jujutsu version control"))

	// Render help sections using bubbles help
	h.help.Width = h.viewport.Width
	helpView := h.help.View(h.keymap)
	sections = append(sections, helpView)

	// Additional help text
	sectionTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorYellow).
		MarginTop(2).
		MarginBottom(1)

	sections = append(sections, sectionTitleStyle.Render("Navigation"))
	navHelp := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Render("• Use tab/shift+tab to navigate between panels\n" +
		"• Use number keys (0-4) to jump directly to a panel:\n" +
		"  0: Diff viewer  1: Status  2: Files  3: Bookmarks  4: Operations\n" +
		"• Use arrow keys or j/k to move within lists\n" +
		"• Press enter to select an item")
	sections = append(sections, navHelp)

	sections = append(sections, sectionTitleStyle.Render("Panels"))
	panelsHelp := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Render("• Status: Shows the current working copy revision\n" +
		"• Files: Lists changed files in the working copy\n" +
		"• Bookmarks: Shows local and remote bookmarks\n" +
		"• Operations: Shows recent Jujutsu operations\n" +
		"• Diff: Displays file diffs and revision changes")
	sections = append(sections, panelsHelp)

	sections = append(sections, sectionTitleStyle.Render("Special Views"))
	specialHelp := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Render("• Log (L): View full revision history\n" +
		"• Help (?): Show this help screen\n" +
		"• Press esc or ? to close overlays")
	sections = append(sections, specialHelp)

	sections = append(sections, sectionTitleStyle.Render("File Operations"))
	fileOpsHelp := lipgloss.NewStyle().
		Foreground(theme.ColorWhite).
		Render("When viewing files in working copy (@):\n" +
		"• del/backspace: Discard file changes (uses jj restore)\n" +
		"• s: Squash file changes to parent commit\n" +
		"\nNote: All operations are undoable with 'jj undo'")
	sections = append(sections, fileOpsHelp)

	return strings.Join(sections, "\n")
}

func (h *HelpOverlay) renderFrame(content string) string {
	// Use lipgloss native border rendering
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBlue).
		Width(h.width - 2).  // Account for border width
		Height(h.height - 2) // Account for border height

	// Render content with border
	bordered := borderStyle.Render(content)

	// Add title to top border
	lines := strings.Split(bordered, "\n")
	if len(lines) > 0 {
		// Build custom top border with title using a styled approach
		borderColorStyle := lipgloss.NewStyle().Foreground(theme.ColorBlue)
		styledTitle := theme.FloatingTitleStyle.Render(" Help ")

		titleWidth := lipgloss.Width(styledTitle)
		remainingWidth := h.width - 3 - titleWidth
		if remainingWidth < 0 {
			remainingWidth = 0
		}

		topBorder := borderColorStyle.Render(borders.TopLeft+borders.Horizontal) +
			styledTitle +
			borderColorStyle.Render(strings.Repeat(borders.Horizontal, remainingWidth)+borders.TopRight)

		lines[0] = topBorder
	}

	return strings.Join(lines, "\n")
}
