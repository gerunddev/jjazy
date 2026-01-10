package floating

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jjazy/ui/borders"
	"github.com/gerund/jjazy/ui/theme"
)

// TextInputOverlay is a floating window for text input
type TextInputOverlay struct {
	textInput textinput.Model
	title     string
	width     int
	height    int
	ready     bool
}

// NewTextInputOverlay creates a new floating text input window
func NewTextInputOverlay(title, placeholder, initialValue string) *TextInputOverlay {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(initialValue)
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60

	return &TextInputOverlay{
		textInput: ti,
		title:     title,
	}
}

func (t *TextInputOverlay) Init() tea.Cmd {
	return textinput.Blink
}

func (t *TextInputOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	t.textInput, cmd = t.textInput.Update(msg)
	return t, cmd
}

func (t *TextInputOverlay) View() string {
	if !t.ready {
		return t.renderFrame("Initializing...")
	}

	// Build content
	var lines []string
	lines = append(lines, "")
	lines = append(lines, t.textInput.View())
	lines = append(lines, "")
	lines = append(lines, theme.HelpDescStyle.Render("  ctrl+s save • ctrl+x cancel"))

	content := strings.Join(lines, "\n")

	return t.renderFrame(content)
}

func (t *TextInputOverlay) SetSize(width, height int) {
	t.width = width
	t.height = height
	t.ready = true

	// Update text input width to fit within the window
	inputWidth := min(60, width-8)
	t.textInput.Width = inputWidth
}

func (t *TextInputOverlay) Value() string {
	return t.textInput.Value()
}

func (t *TextInputOverlay) renderFrame(content string) string {
	// Calculate centered window dimensions
	windowWidth := min(70, t.width-4)
	windowHeight := 8

	// Center the window
	x := (t.width - windowWidth) / 2
	y := (t.height - windowHeight) / 2

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
		styledTitle := theme.FloatingTitleStyle.Render(" " + t.title + " ")

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
	for i := range windowLines {
		windowLines[i] = paddingLeft + windowLines[i]
	}

	return paddingTop + strings.Join(windowLines, "\n")
}
