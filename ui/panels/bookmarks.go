package panels

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gerund/jayz/jj"
	"github.com/gerund/jayz/ui/fixtures"
	"github.com/gerund/jayz/ui/theme"
)

// BookmarksPanel shows bookmarks (branches)
type BookmarksPanel struct {
	BasePanel
	repo      *jj.Repo
	bookmarks []fixtures.Bookmark
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
	if !p.focused {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			p.CursorUp(len(p.bookmarks))
		case "down", "j":
			p.CursorDown(len(p.bookmarks))
		case "g", "home":
			p.CursorHome()
		case "G", "end":
			p.CursorEnd(len(p.bookmarks))
		}
	}

	return p, nil
}

func (p *BookmarksPanel) View() string {
	var lines []string

	contentHeight := p.ContentHeight()
	contentWidth := p.ContentWidth()

	for i, bm := range p.bookmarks {
		if i >= contentHeight {
			break
		}

		// Build the line with indicator for current bookmark
		indicator := "  "
		if bm.IsCurrent {
			indicator = theme.WorkingCopyStyle.Render("● ")
		}

		// Style the name based on local vs remote
		name := bm.Name
		var styledName string
		if i == p.cursor && p.focused {
			styledName = theme.SelectedItemStyle.Render(name)
		} else if bm.IsLocal {
			styledName = theme.NormalItemStyle.Render(name)
		} else {
			// Remote bookmarks are dimmed
			styledName = theme.DimmedStyle.Render(name)
		}

		line := indicator + styledName

		// Truncate if needed
		if len(bm.Name)+2 > contentWidth {
			name = truncate(bm.Name, contentWidth-2)
			if i == p.cursor && p.focused {
				styledName = theme.SelectedItemStyle.Render(name)
			} else if bm.IsLocal {
				styledName = theme.NormalItemStyle.Render(name)
			} else {
				styledName = theme.DimmedStyle.Render(name)
			}
			line = indicator + styledName
		}

		lines = append(lines, line)
	}

	// Pad remaining space
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return p.RenderFrame(content)
}

// SelectedBookmark returns the currently selected bookmark
func (p *BookmarksPanel) SelectedBookmark() *fixtures.Bookmark {
	if p.cursor >= 0 && p.cursor < len(p.bookmarks) {
		return &p.bookmarks[p.cursor]
	}
	return nil
}

// Ensure BookmarksPanel implements Panel
var _ Panel = (*BookmarksPanel)(nil)
