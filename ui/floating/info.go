package floating

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jjazy/ui/borders"
	"github.com/gerund/jjazy/ui/theme"
)

// InfoOverlay is a floating information/error dialog with OK button
type InfoOverlay struct {
	title   string
	message string
	width   int
	height  int
	ready   bool
}

// NewInfoOverlay creates a new information dialog
func NewInfoOverlay(title, message string) *InfoOverlay {
	return &InfoOverlay{
		title:   title,
		message: message,
	}
}

func (i *InfoOverlay) Init() tea.Cmd {
	return nil
}

func (i *InfoOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Any key dismisses the info overlay (handled by caller)
	return i, nil
}

func (i *InfoOverlay) View() string {
	if !i.ready {
		return i.renderFrame("Initializing...")
	}

	// Build content
	var lines []string
	lines = append(lines, "")

	// Word wrap message if too long
	maxWidth := min(56, i.width-8)
	wrappedLines := wrapText(i.message, maxWidth)
	for _, line := range wrappedLines {
		lines = append(lines, "  "+line)
	}

	lines = append(lines, "")

	// OK button (always selected)
	okStyle := theme.SelectedItemStyle
	button := "        " + okStyle.Render("[ OK ]")
	lines = append(lines, button)
	lines = append(lines, "")

	content := strings.Join(lines, "\n")

	return i.renderFrame(content)
}

func (i *InfoOverlay) SetSize(width, height int) {
	i.width = width
	i.height = height
	i.ready = true
}

func (i *InfoOverlay) renderFrame(content string) string {
	// Calculate centered window dimensions
	windowWidth := min(60, i.width-4)
	windowHeight := min(12, i.height-4)

	// Center the window
	x := (i.width - windowWidth) / 2
	y := (i.height - windowHeight) / 2

	// Use lipgloss native border rendering
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorYellow).
		Width(windowWidth - 2).
		Height(windowHeight - 2)

	// Render content with border
	bordered := borderStyle.Render(content)

	// Add title to top border
	lines := strings.Split(bordered, "\n")
	if len(lines) > 0 {
		borderColorStyle := lipgloss.NewStyle().Foreground(theme.ColorYellow)
		styledTitle := theme.FloatingTitleStyle.Render(" " + i.title + " ")

		titleWidth := lipgloss.Width(styledTitle)
		remainingWidth := windowWidth - 3 - titleWidth
		if remainingWidth < 0 {
			remainingWidth = 0
		}

		topBorder := borderColorStyle.Render(borders.TopLeft+borders.Horizontal) +
			styledTitle +
			borderColorStyle.Render(strings.Repeat(borders.Horizontal, remainingWidth)+borders.TopRight)

		lines[0] = topBorder
	}

	centeredWindow := strings.Join(lines, "\n")

	// Add vertical padding to center the window
	paddingTop := strings.Repeat("\n", y)
	paddingLeft := strings.Repeat(" ", x)

	// Apply horizontal padding to each line
	windowLines := strings.Split(centeredWindow, "\n")
	for idx := range windowLines {
		windowLines[idx] = paddingLeft + windowLines[idx]
	}

	return paddingTop + strings.Join(windowLines, "\n")
}

// wrapText wraps text to a maximum width
func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentLine.Len()+1+len(word) <= maxWidth {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}
