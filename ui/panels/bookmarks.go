package panels

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerunddev/jjazy/app"
	"github.com/gerunddev/jjazy/jj"
	"github.com/gerunddev/jjazy/ui/borders"
	"github.com/gerunddev/jjazy/ui/fixtures"
	"github.com/gerunddev/jjazy/ui/theme"
)

// BookmarksPanel shows bookmarks (branches).
// This is a "browsable" panel that requires Enter to show cursor.
type BookmarksPanel struct {
	BasePanel
	repo      *jj.Repo
	repoPath  string
	bookmarks []fixtures.Bookmark
	viewport  viewport.Model
	ready     bool
}

// SetEntered overrides BasePanel to also reset viewport and re-render
func (p *BookmarksPanel) SetEntered(entered bool) {
	p.BasePanel.SetEntered(entered)
	if p.ready {
		if entered {
			p.viewport.GotoTop()
		}
		// Re-render to show/hide cursor styling
		p.viewport.SetContent(p.renderContent())
	}
}

// NewBookmarksPanel creates a new bookmarks panel
func NewBookmarksPanel(repo *jj.Repo, repoPath string) *BookmarksPanel {
	p := &BookmarksPanel{
		BasePanel: NewBasePanel("2 Bookmarks", "branches"),
		repo:      repo,
		repoPath:  repoPath,
	}
	p.loadBookmarks()
	return p
}

func (p *BookmarksPanel) loadBookmarks() {
	// Get branches from jj-lib
	branches, err := p.repo.Branches()
	if err != nil {
		// Fall back to empty list on error
		p.bookmarks = nil
		return
	}

	// Use Navigation to find current bookmark (closest to working copy)
	var currentBookmark string
	if revisions, err := p.repo.Log(); err == nil {
		nav := app.NewNavigation(p.repoPath, revisions)
		currentBookmark = nav.FindCurrentBookmark()
	}

	// Convert jj.Branch to fixtures.Bookmark
	p.bookmarks = make([]fixtures.Bookmark, len(branches))
	for i, b := range branches {
		p.bookmarks[i] = fixtures.Bookmark{
			Name:      b.Name,
			IsLocal:   b.IsLocal,
			IsCurrent: b.Name == currentBookmark,
		}
	}
}

// Refresh reloads bookmark data and re-renders.
func (p *BookmarksPanel) Refresh() {
	p.loadBookmarks()
	if p.ready {
		p.viewport.SetContent(p.renderContent())
	}
}

func (p *BookmarksPanel) Init() tea.Cmd {
	return nil
}

func (p *BookmarksPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress {
				itemIndex := msg.Y - 1 + p.viewport.YOffset
				if itemIndex >= 0 && itemIndex < len(p.bookmarks) {
					p.cursor = itemIndex
					p.ensureCursorVisible()
				}
			}
		case tea.MouseButtonWheelUp:
			p.viewport.LineUp(3)
		case tea.MouseButtonWheelDown:
			p.viewport.LineDown(3)
		}

	case tea.KeyMsg:
		// Only process cursor keys when focused AND entered
		if !p.focused || !p.entered {
			return p, nil
		}
		switch msg.String() {
		case "up", "k":
			p.CursorUp(len(p.bookmarks))
			p.ensureCursorVisible()
		case "down", "j":
			p.CursorDown(len(p.bookmarks))
			p.ensureCursorVisible()
		case "g", "home":
			p.CursorHome()
			p.viewport.GotoTop()
		case "G", "end":
			p.CursorEnd(len(p.bookmarks))
			p.viewport.GotoBottom()
		case "ctrl+u", "pgup":
			p.viewport.HalfViewUp()
		case "ctrl+d", "pgdown":
			p.viewport.HalfViewDown()
		}
	}

	if p.ready {
		p.viewport.SetContent(p.renderContent())
	}

	return p, nil
}

func (p *BookmarksPanel) ensureCursorVisible() {
	if p.cursor < p.viewport.YOffset {
		p.viewport.SetYOffset(p.cursor)
	} else if p.cursor >= p.viewport.YOffset+p.viewport.Height {
		p.viewport.SetYOffset(p.cursor - p.viewport.Height + 1)
	}
}

func (p *BookmarksPanel) View() string {
	if !p.ready {
		return p.RenderFrame("Loading...")
	}
	return p.RenderFrame(p.viewport.View())
}

// RenderFrame overrides BasePanel to show white border in cursor mode
func (p *BookmarksPanel) RenderFrame(content string) string {
	// Focus mode: yellow border (focused && !entered)
	// Cursor mode: white border (entered)
	showFocusBorder := p.focused && !p.entered
	return borders.RenderTitledBorder(content, p.title, p.width, p.height, showFocusBorder)
}

// SetSize initializes or resizes the viewport
func (p *BookmarksPanel) SetSize(width, height int) {
	p.BasePanel.SetSize(width, height)

	contentWidth := p.ContentWidth()
	contentHeight := p.ContentHeight()

	if !p.ready {
		p.viewport = viewport.New(contentWidth, contentHeight)
		p.viewport.SetContent(p.renderContent())
		p.ready = true
	} else {
		p.viewport.Width = contentWidth
		p.viewport.Height = contentHeight
		p.viewport.SetContent(p.renderContent())
	}
}

func (p *BookmarksPanel) renderContent() string {
	var lines []string
	contentWidth := p.ContentWidth()

	for i, bm := range p.bookmarks {
		// Truncate if needed
		name := bm.Name
		if len(name)+2 > contentWidth && contentWidth > 3 {
			name = truncate(name, contentWidth-3)
		}

		// Style the name based on current/selected state
		// Cursor (yellow) takes priority when entered
		var styledName string
		if i == p.cursor && p.focused && p.entered {
			// Selected + entered: YELLOW (overrides current color)
			styledName = theme.SelectedItemStyle.Render(name)
		} else if bm.IsCurrent {
			// Current bookmark is PURPLE
			styledName = theme.CurrentBookmarkStyle.Render(name)
		} else if bm.IsLocal {
			styledName = theme.NormalItemStyle.Render(name)
		} else {
			styledName = theme.DimmedStyle.Render(name)
		}

		line := styledName
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// SelectedBookmark returns the currently selected bookmark
func (p *BookmarksPanel) SelectedBookmark() *fixtures.Bookmark {
	if p.cursor >= 0 && p.cursor < len(p.bookmarks) {
		return &p.bookmarks[p.cursor]
	}
	return nil
}

// Count returns the number of bookmarks
func (p *BookmarksPanel) Count() int {
	return len(p.bookmarks)
}

// Ensure BookmarksPanel implements Panel
var _ Panel = (*BookmarksPanel)(nil)
