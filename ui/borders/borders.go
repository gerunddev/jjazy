package borders

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jjazy/ui/theme"
)

// Rounded border characters
const (
	TopLeft     = "╭"
	TopRight    = "╮"
	BottomLeft  = "╰"
	BottomRight = "╯"
	Horizontal  = "─"
	Vertical    = "│"
)

// RenderTitledBorder creates a box with title embedded in top border.
// Example: ╭─ 1 Status ─────────────╮
func RenderTitledBorder(content, title string, width, height int, focused bool) string {
	if width < 4 || height < 2 {
		return content
	}

	var borderStyle, titleStyle lipgloss.Style
	if focused {
		borderStyle = lipgloss.NewStyle().Foreground(theme.ColorYellow)
		titleStyle = lipgloss.NewStyle().Foreground(theme.ColorYellow)
	} else {
		borderStyle = lipgloss.NewStyle().Foreground(theme.ColorDimWhite)
		titleStyle = lipgloss.NewStyle().Foreground(theme.ColorWhite)
	}

	// Build top border with title
	topBorder := buildTopBorder(title, width, borderStyle, titleStyle)

	// Build bottom border
	bottomBorder := buildBottomBorder(width, borderStyle)

	// Split content into lines and build side borders
	contentLines := strings.Split(content, "\n")
	contentWidth := width - 2 // Account for left and right borders

	var lines []string
	lines = append(lines, topBorder)

	// Calculate content height (total height minus top and bottom borders)
	contentHeight := height - 2

	for i := 0; i < contentHeight; i++ {
		var line string
		if i < len(contentLines) {
			line = contentLines[i]
		} else {
			line = ""
		}

		// Pad or truncate line to fit content width
		line = padOrTruncate(line, contentWidth)

		// Add side borders
		borderedLine := borderStyle.Render(Vertical) + line + borderStyle.Render(Vertical)
		lines = append(lines, borderedLine)
	}

	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// buildTopBorder creates: ╭─ Title ─────────╮
func buildTopBorder(title string, width int, borderStyle, titleStyle lipgloss.Style) string {
	if width < 4 {
		return ""
	}

	// Calculate available space for horizontal lines
	// Format: ╭─ Title ─...─╮
	titleLen := lipgloss.Width(title)
	minPadding := 4 // "╭─ " before title and " ─╮" after (minimum)

	var topLine string
	if titleLen+minPadding > width {
		// Title too long, truncate
		maxTitleLen := width - minPadding
		if maxTitleLen > 0 {
			title = truncateString(title, maxTitleLen)
			titleLen = lipgloss.Width(title)
		} else {
			title = ""
			titleLen = 0
		}
	}

	// Build: ╭─ Title ─...─╮
	// Total: "╭─ "(3) + title(titleLen) + " ─"(2) + repeated(X) + "╮"(1) = width
	// So: 6 + titleLen + X = width => X = width - 6 - titleLen
	remainingWidth := width - 6 - titleLen
	if remainingWidth < 0 {
		remainingWidth = 0
	}

	topLine = borderStyle.Render(TopLeft+Horizontal+" ") +
		titleStyle.Render(title) +
		borderStyle.Render(" "+Horizontal+strings.Repeat(Horizontal, remainingWidth)+TopRight)

	return topLine
}

// buildBottomBorder creates: ╰─────────────────╯
func buildBottomBorder(width int, borderStyle lipgloss.Style) string {
	if width < 2 {
		return ""
	}

	innerWidth := width - 2 // Minus corners
	return borderStyle.Render(BottomLeft + strings.Repeat(Horizontal, innerWidth) + BottomRight)
}

// padOrTruncate ensures a string is exactly the given width
func padOrTruncate(s string, width int) string {
	currentWidth := lipgloss.Width(s)

	if currentWidth > width {
		return truncateString(s, width)
	}

	if currentWidth < width {
		return s + strings.Repeat(" ", width-currentWidth)
	}

	return s
}

// truncateString truncates a string to the given width, handling ANSI codes
func truncateString(s string, width int) string {
	if width <= 0 {
		return ""
	}

	// Use lipgloss to handle ANSI-aware truncation
	style := lipgloss.NewStyle().MaxWidth(width)
	return style.Render(s)
}

// RenderFloatingBorder creates a floating window border with title
func RenderFloatingBorder(content, title string, width, height int) string {
	borderStyle := lipgloss.NewStyle().Foreground(theme.ColorYellow)
	titleStyle := theme.FloatingTitleStyle

	// Build top border with styled title
	topBorder := buildFloatingTopBorder(title, width, borderStyle, titleStyle)

	// Build bottom border
	bottomBorder := buildBottomBorder(width, borderStyle)

	// Split content into lines
	contentLines := strings.Split(content, "\n")
	contentWidth := width - 2

	var lines []string
	lines = append(lines, topBorder)

	contentHeight := height - 2
	for i := 0; i < contentHeight; i++ {
		var line string
		if i < len(contentLines) {
			line = contentLines[i]
		} else {
			// Empty line - fill with spaces
			line = strings.Repeat(" ", contentWidth)
		}

		// Pad if needed (but don't truncate styled content)
		currentWidth := lipgloss.Width(line)
		if currentWidth < contentWidth {
			line = line + strings.Repeat(" ", contentWidth-currentWidth)
		}

		// Add side borders
		borderedLine := borderStyle.Render(Vertical) + line + borderStyle.Render(Vertical)
		lines = append(lines, borderedLine)
	}

	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// buildFloatingTopBorder creates top border for floating windows
func buildFloatingTopBorder(title string, width int, borderStyle lipgloss.Style, titleStyle lipgloss.Style) string {
	if width < 4 {
		return ""
	}

	styledTitle := titleStyle.Render(" " + title + " ")
	titleLen := lipgloss.Width(styledTitle)

	// Build: ╭─Title─...─╮
	// Total: "╭─"(2) + title(titleLen) + repeated(X) + "╮"(1) = width
	// So: 3 + titleLen + X = width => X = width - 3 - titleLen
	remainingWidth := width - 3 - titleLen
	if remainingWidth < 0 {
		remainingWidth = 0
	}

	return borderStyle.Render(TopLeft+Horizontal) +
		styledTitle +
		borderStyle.Render(strings.Repeat(Horizontal, remainingWidth)+TopRight)
}
