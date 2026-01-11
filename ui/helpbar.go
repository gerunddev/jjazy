package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jjazy/ui/theme"
)

// HelpBarContext captures the current UI state for help bar rendering
type HelpBarContext struct {
	Experience      Experience
	FocusedPanel    int
	Entered         bool // True if current panel is in "entered" mode
	IsWorkingCopy   bool // True when viewing @ change
	BookmarkSetMode bool // True when in bookmark set flow
}

// HelpHint represents a single hint (key + description)
type HelpHint struct {
	Key  string
	Desc string
}

// Format renders a hint as "key desc" in uniform dim color
func (h HelpHint) Format() string {
	return theme.HelpDescStyle.Render(h.Key + " " + h.Desc)
}

// getActionHints returns context-specific action hints (left section)
func getActionHints(ctx HelpBarContext) []HelpHint {
	switch ctx.Experience {
	case ExperienceLog:
		switch ctx.FocusedPanel {
		case 0: // Log panel
			if ctx.BookmarkSetMode {
				return []HelpHint{
					{Key: "↵", Desc: "set"},
				}
			}
			return []HelpHint{
				{Key: "↵", Desc: "edit"},
				{Key: "n", Desc: "new"},
				{Key: "d", Desc: "describe"},
				{Key: "a", Desc: "abandon"},
				{Key: "s", Desc: "squash"},
			}
		case 1: // Workspace panel
			if ctx.Entered {
				return []HelpHint{{Key: "↵", Desc: "edit"}}
			}
			return nil
		case 2: // Bookmarks panel
			if ctx.Entered {
				return []HelpHint{
					{Key: "↵", Desc: "set"},
					{Key: "e", Desc: "edit"},
				}
			}
			return nil
		}
	case ExperienceChange:
		switch ctx.FocusedPanel {
		case 1: // Files panel
			if ctx.IsWorkingCopy {
				return []HelpHint{
					{Key: "del", Desc: "discard"},  // PM feedback: "discard" clearer than "restore"
					{Key: "s", Desc: "squash"},
				}
			}
			return nil
		default:
			return nil
		}
	}
	return nil
}

// getNavigationHints returns context-specific navigation hints (center section)
func getNavigationHints(ctx HelpBarContext) []HelpHint {
	switch ctx.Experience {
	case ExperienceLog:
		switch ctx.FocusedPanel {
		case 0: // Log panel
			if ctx.BookmarkSetMode {
				return []HelpHint{
					{Key: "←", Desc: "cancel"},
					{Key: "↑↓", Desc: "select"},
				}
			}
			return []HelpHint{
				{Key: "→", Desc: "view"},
			}
		case 1: // Workspace panel
			if ctx.Entered {
				return []HelpHint{
					{Key: "↑↓", Desc: "select"},
				}
			}
			return []HelpHint{
				{Key: "↑↓", Desc: "panels"},
				{Key: "↵", Desc: "enter"},
			}
		case 2: // Bookmarks panel
			if ctx.Entered {
				return []HelpHint{
					{Key: "↑↓", Desc: "select"},
				}
			}
			return []HelpHint{
				{Key: "↑↓", Desc: "panels"},
				{Key: "↵", Desc: "enter"},
			}
		}
	case ExperienceChange:
		switch ctx.FocusedPanel {
		case 0: // Diff panel
			return []HelpHint{
				{Key: "←", Desc: "files"},
				{Key: "↑↓", Desc: "scroll"},
			}
		case 1: // Files panel
			return []HelpHint{
				{Key: "←", Desc: "exit"},
				{Key: "→", Desc: "diff"},
				{Key: "↑↓", Desc: "select"},
			}
		}
	}
	return nil
}

// getAlwaysHints returns hints that are always shown (right section)
func getAlwaysHints() []HelpHint {
	return []HelpHint{
		{Key: "tab", Desc: "↻"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	}
}

// formatHints joins hints with double spaces
func formatHints(hints []HelpHint) string {
	if len(hints) == 0 {
		return ""
	}

	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = h.Format()
	}
	return strings.Join(parts, "  ")
}

// RenderContextualHelpBar renders the three-section help bar
func RenderContextualHelpBar(ctx HelpBarContext, width int) string {
	// Build each section
	actionHints := getActionHints(ctx)
	navHints := getNavigationHints(ctx)
	alwaysHints := getAlwaysHints()

	// Format sections (all uniform dim color)
	leftSection := formatHints(actionHints)
	centerSection := formatHints(navHints)
	rightSection := formatHints(alwaysHints)

	// Calculate widths (using lipgloss to handle ANSI sequences)
	leftWidth := lipgloss.Width(leftSection)
	centerWidth := lipgloss.Width(centerSection)
	rightWidth := lipgloss.Width(rightSection)

	// Calculate available space for padding
	totalContentWidth := leftWidth + centerWidth + rightWidth
	availableSpace := width - totalContentWidth

	if availableSpace < 6 {
		// Not enough space, just join everything with minimal spacing
		return theme.HelpBarStyle.Width(width).Render(
			leftSection + "  " + centerSection + "  " + rightSection,
		)
	}

	// Distribute space to center the navigation section
	// Layout: [left].....[center].....[right]
	// We want center to be roughly in the middle, right to be at the far right

	// Calculate spacing
	// Right section should be at the far right
	// Center section should be roughly centered
	// Left section is left-aligned

	midPoint := width / 2
	centerStart := midPoint - centerWidth/2

	// Space between left and center
	leftToCenter := max(centerStart-leftWidth, 2)

	// Space between center and right
	centerEnd := centerStart + centerWidth
	rightStart := width - rightWidth
	centerToRight := max(rightStart-centerEnd, 2)

	// Build the bar
	var bar string
	if leftWidth > 0 {
		bar = leftSection + strings.Repeat(" ", leftToCenter) + centerSection + strings.Repeat(" ", centerToRight) + rightSection
	} else {
		// No left section, adjust spacing
		// Put center section roughly in the middle-left area
		leftPadding := max(centerStart, 0)
		bar = strings.Repeat(" ", leftPadding) + centerSection + strings.Repeat(" ", centerToRight) + rightSection
	}

	return theme.HelpBarStyle.Width(width).Render(bar)
}
