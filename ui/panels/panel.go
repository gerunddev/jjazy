package panels

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jayz/ui/theme"
)

// Panel defines the interface for all sidebar panels
type Panel interface {
	tea.Model
	Title() string
	ShortHelp() string
	SetFocused(bool)
	IsFocused() bool
	SetSize(width, height int)
}

// BasePanel provides common functionality for all panels
type BasePanel struct {
	title     string
	shortHelp string
	focused   bool
	width     int
	height    int
	cursor    int
}

// NewBasePanel creates a new base panel
func NewBasePanel(title, shortHelp string) BasePanel {
	return BasePanel{
		title:     title,
		shortHelp: shortHelp,
	}
}

func (b *BasePanel) Title() string {
	return b.title
}

func (b *BasePanel) ShortHelp() string {
	return b.shortHelp
}

func (b *BasePanel) SetFocused(focused bool) {
	b.focused = focused
}

func (b *BasePanel) IsFocused() bool {
	return b.focused
}

func (b *BasePanel) SetSize(width, height int) {
	b.width = width
	b.height = height
}

func (b *BasePanel) Width() int {
	return b.width
}

func (b *BasePanel) Height() int {
	return b.height
}

func (b *BasePanel) Cursor() int {
	return b.cursor
}

func (b *BasePanel) SetCursor(c int) {
	b.cursor = c
}

// ContentHeight returns the height available for content (minus borders and title)
func (b *BasePanel) ContentHeight() int {
	// Account for top border, title line, and bottom border
	return b.height - 3
}

// ContentWidth returns the width available for content (minus borders)
func (b *BasePanel) ContentWidth() int {
	// Account for left and right borders
	return b.width - 2
}

// RenderFrame renders the panel frame with title and content
func (b *BasePanel) RenderFrame(content string) string {
	var style lipgloss.Style
	var titleStyle lipgloss.Style

	if b.focused {
		style = theme.FocusedBorder.
			Width(b.width).
			Height(b.height)
		titleStyle = theme.FocusedTitleStyle
	} else {
		style = theme.UnfocusedBorder.
			Width(b.width).
			Height(b.height)
		titleStyle = theme.TitleStyle
	}

	// Render title
	title := titleStyle.Render(b.title)

	// Combine title and content
	fullContent := lipgloss.JoinVertical(lipgloss.Left, title, content)

	return style.Render(fullContent)
}

// CursorUp moves the cursor up within bounds
func (b *BasePanel) CursorUp(itemCount int) {
	if b.cursor > 0 {
		b.cursor--
	}
}

// CursorDown moves the cursor down within bounds
func (b *BasePanel) CursorDown(itemCount int) {
	if b.cursor < itemCount-1 {
		b.cursor++
	}
}

// CursorHome moves the cursor to the first item
func (b *BasePanel) CursorHome() {
	b.cursor = 0
}

// CursorEnd moves the cursor to the last item
func (b *BasePanel) CursorEnd(itemCount int) {
	if itemCount > 0 {
		b.cursor = itemCount - 1
	}
}
