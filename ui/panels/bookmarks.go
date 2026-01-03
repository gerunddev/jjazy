package panels

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerund/jjazy/jj"
	"github.com/gerund/jjazy/ui/fixtures"
	"github.com/gerund/jjazy/ui/theme"
)

// BookmarksPanel shows bookmarks (branches)
type BookmarksPanel struct {
	BasePanel
	repo      *jj.Repo
	bookmarks []fixtures.Bookmark
	viewport  viewport.Model
	ready     bool
}

// NewBookmarksPanel creates a new bookmarks panel
func NewBookmarksPanel(repo *jj.Repo) *BookmarksPanel {
	p := &BookmarksPanel{
		BasePanel: NewBasePanel("3 Bookmarks", "branches"),
		repo:      repo,
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

	// Convert jj.Branch to fixtures.Bookmark
	p.bookmarks = make([]fixtures.Bookmark, len(branches))
	for i, b := range branches {
		p.bookmarks[i] = fixtures.Bookmark{
			Name:    b.Name,
			IsLocal: b.IsLocal,
			// TODO: Get these from jj-lib when available
			RevisionID: "",
			IsCurrent:  false,
		}
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
		if !p.focused {
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
		// Build the line with indicator for current bookmark
		indicator := "  "
		if bm.IsCurrent {
			indicator = theme.WorkingCopyStyle.Render("● ")
		}

		// Truncate if needed
		name := bm.Name
		if len(name)+2 > contentWidth && contentWidth > 3 {
			name = truncate(name, contentWidth-3)
		}

		// Style the name based on local vs remote
		var styledName string
		if i == p.cursor && p.focused {
			styledName = theme.SelectedItemStyle.Render(name)
		} else if bm.IsLocal {
			styledName = theme.NormalItemStyle.Render(name)
		} else {
			styledName = theme.DimmedStyle.Render(name)
		}

		line := indicator + styledName
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
