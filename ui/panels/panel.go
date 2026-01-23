package panels

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerunddev/jjazy/ui/borders"
)

// Panel defines the interface for all sidebar panels
type Panel interface {
	tea.Model
	Title() string
	ShortHelp() string
	SetFocused(bool)
	IsFocused() bool
	SetEntered(bool)
	IsEntered() bool
	SetSize(width, height int)
}

// BasePanel provides common functionality for all panels
type BasePanel struct {
	title     string
	shortHelp string
	focused   bool
	entered   bool // Whether cursor is visible and navigable (two-level focus)
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

func (b *BasePanel) SetEntered(entered bool) {
	b.entered = entered
	if entered {
		// Reset cursor to top when entering cursor mode
		b.cursor = 0
	}
}

func (b *BasePanel) IsEntered() bool {
	return b.entered
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

// ContentHeight returns the height available for content (minus borders)
func (b *BasePanel) ContentHeight() int {
	// With titled borders, title is IN the border, so just subtract top + bottom borders
	return b.height - 2
}

// ContentWidth returns the width available for content (minus borders)
func (b *BasePanel) ContentWidth() int {
	// Account for left and right borders
	return b.width - 2
}

// RenderFrame renders the panel frame with title embedded in border
func (b *BasePanel) RenderFrame(content string) string {
	return borders.RenderTitledBorder(content, b.title, b.width, b.height, b.focused)
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

// HandleCursorKeys handles common cursor key navigation.
// Returns true if the key was handled.
func (b *BasePanel) HandleCursorKeys(msg tea.KeyMsg, itemCount int, vp *viewport.Model) bool {
	switch msg.String() {
	case "up", "k":
		b.CursorUp(itemCount)
		b.EnsureCursorVisible(vp)
		return true
	case "down", "j":
		b.CursorDown(itemCount)
		b.EnsureCursorVisible(vp)
		return true
	case "g", "home":
		b.CursorHome()
		vp.GotoTop()
		return true
	case "G", "end":
		b.CursorEnd(itemCount)
		vp.GotoBottom()
		return true
	case "ctrl+u", "pgup":
		vp.HalfPageUp()
		return true
	case "ctrl+d", "pgdown":
		vp.HalfPageDown()
		return true
	}
	return false
}

// EnsureCursorVisible scrolls the viewport to keep the cursor in view.
func (b *BasePanel) EnsureCursorVisible(vp *viewport.Model) {
	if b.cursor < vp.YOffset {
		vp.SetYOffset(b.cursor)
	} else if b.cursor >= vp.YOffset+vp.Height {
		vp.SetYOffset(b.cursor - vp.Height + 1)
	}
}

// HandleMouseClick handles mouse click for item selection.
// Returns true if an item was selected.
func (b *BasePanel) HandleMouseClick(msg tea.MouseMsg, vp *viewport.Model, itemCount int) bool {
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		itemIndex := msg.Y - 1 + vp.YOffset
		if itemIndex >= 0 && itemIndex < itemCount {
			b.cursor = itemIndex
			b.EnsureCursorVisible(vp)
			return true
		}
	}
	return false
}

// HandleMouseScroll handles scroll wheel events.
func (b *BasePanel) HandleMouseScroll(msg tea.MouseMsg, vp *viewport.Model) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		vp.ScrollUp(3)
	case tea.MouseButtonWheelDown:
		vp.ScrollDown(3)
	}
}
