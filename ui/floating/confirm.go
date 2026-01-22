package floating

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerunddev/jjazy/ui/borders"
	"github.com/gerunddev/jjazy/ui/theme"
)

// ConfirmOverlay is a floating Yes/No confirmation dialog
type ConfirmOverlay struct {
	title    string
	message  string
	width    int
	height   int
	ready    bool
	selected int // 0 = Yes, 1 = No
}

// NewConfirmOverlay creates a new confirmation dialog
func NewConfirmOverlay(title, message string) *ConfirmOverlay {
	return &ConfirmOverlay{
		title:    title,
		message:  message,
		selected: 1, // Default to "No" for safety
	}
}

func (c *ConfirmOverlay) Init() tea.Cmd {
	return nil
}

func (c *ConfirmOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			c.selected = 0 // Yes
		case "right", "l":
			c.selected = 1 // No
		case "tab":
			c.selected = (c.selected + 1) % 2
		case "y", "Y":
			c.selected = 0 // Yes
		case "n", "N":
			c.selected = 1 // No
		}
	}
	return c, nil
}

func (c *ConfirmOverlay) View() string {
	if !c.ready {
		return c.renderFrame("Initializing...")
	}

	// Build content
	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  "+c.message)
	lines = append(lines, "")

	// Build Yes/No buttons
	yesStyle := theme.HelpDescStyle
	noStyle := theme.HelpDescStyle
	if c.selected == 0 {
		yesStyle = theme.SelectedItemStyle
	}
	if c.selected == 1 {
		noStyle = theme.SelectedItemStyle
	}

	buttons := "        " + yesStyle.Render("[ Yes ]") + "    " + noStyle.Render("[ No ]")
	lines = append(lines, buttons)
	lines = append(lines, "")

	content := strings.Join(lines, "\n")

	return c.renderFrame(content)
}

func (c *ConfirmOverlay) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.ready = true
}

// Confirmed returns true if Yes is selected
func (c *ConfirmOverlay) Confirmed() bool {
	return c.selected == 0
}

func (c *ConfirmOverlay) renderFrame(content string) string {
	// Calculate centered window dimensions
	windowWidth := min(60, c.width-4)
	windowHeight := 8

	// Center the window
	x := (c.width - windowWidth) / 2
	y := (c.height - windowHeight) / 2

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
		styledTitle := theme.FloatingTitleStyle.Render(" " + c.title + " ")

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
