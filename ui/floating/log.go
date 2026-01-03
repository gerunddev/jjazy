package floating

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jayz/jj"
	"github.com/gerund/jayz/ui/fixtures"
	"github.com/gerund/jayz/ui/theme"
)

// LogOverlay is a floating window showing the revision graph
type LogOverlay struct {
	repo      *jj.Repo
	viewport  viewport.Model
	revisions []fixtures.Revision
	cursor    int
	width     int
	height    int
	ready     bool
}

// NewLogOverlay creates a new floating log window
func NewLogOverlay(repo *jj.Repo) *LogOverlay {
	l := &LogOverlay{repo: repo}
	l.loadRevisions()
	return l
}

func (l *LogOverlay) loadRevisions() {
	// Get revisions from jj-lib
	revs, err := l.repo.Log()
	if err != nil {
		// Fall back to empty list on error
		l.revisions = nil
		return
	}

	// Convert jj.Revision to fixtures.Revision
	l.revisions = make([]fixtures.Revision, len(revs))
	for i, rev := range revs {
		wsName := ""
		if rev.WorkspaceName != nil {
			wsName = *rev.WorkspaceName
		}
		l.revisions[i] = fixtures.Revision{
			ID:            rev.ID,
			ChangeID:      rev.ChangeID,
			Description:   rev.Description,
			Author:        rev.Author,
			Timestamp:     rev.Timestamp, // Unix timestamp (TODO: format as relative time)
			IsWorkingCopy: rev.IsWorkingCopy,
			WorkspaceName: wsName,
			IsRoot:        rev.IsRoot,
			Parents:       rev.Parents,
		}
	}
}

func (l *LogOverlay) Init() tea.Cmd {
	return nil
}

func (l *LogOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if l.cursor > 0 {
				l.cursor--
				l.ensureCursorVisible()
			}
		case "down", "j":
			if l.cursor < len(l.revisions)-1 {
				l.cursor++
				l.ensureCursorVisible()
			}
		case "g", "home":
			l.cursor = 0
			l.viewport.GotoTop()
		case "G", "end":
			if len(l.revisions) > 0 {
				l.cursor = len(l.revisions) - 1
				l.viewport.GotoBottom()
			}
		case "pgup", "ctrl+u":
			l.viewport.HalfViewUp()
		case "pgdown", "ctrl+d":
			l.viewport.HalfViewDown()
		}
	}

	// Re-render content when cursor changes
	if l.ready {
		l.viewport.SetContent(l.renderLog())
	}

	l.viewport, cmd = l.viewport.Update(msg)
	return l, cmd
}

func (l *LogOverlay) ensureCursorVisible() {
	// Each revision takes 2-3 lines, estimate position
	linePos := l.cursor * 2
	if linePos < l.viewport.YOffset {
		l.viewport.SetYOffset(linePos)
	} else if linePos >= l.viewport.YOffset+l.viewport.Height-2 {
		l.viewport.SetYOffset(linePos - l.viewport.Height + 3)
	}
}

func (l *LogOverlay) View() string {
	if !l.ready {
		return l.renderFrame("Initializing...")
	}

	return l.renderFrame(l.viewport.View())
}

func (l *LogOverlay) SetSize(width, height int) {
	l.width = width
	l.height = height

	contentWidth := width - 4  // borders + padding
	contentHeight := height - 4 // borders + title + padding

	if !l.ready {
		l.viewport = viewport.New(contentWidth, contentHeight)
		l.viewport.SetContent(l.renderLog())
		l.ready = true
	} else {
		l.viewport.Width = contentWidth
		l.viewport.Height = contentHeight
		l.viewport.SetContent(l.renderLog())
	}
}

func (l *LogOverlay) renderLog() string {
	var lines []string
	contentWidth := l.width - 6 // borders + padding + graph margin

	for i, rev := range l.revisions {
		// Build graph characters
		var graphChar string
		if rev.IsWorkingCopy {
			graphChar = theme.WorkingCopyStyle.Render("@")
		} else if rev.IsRoot {
			graphChar = theme.DimmedStyle.Render("◆")
		} else {
			graphChar = theme.DimmedStyle.Render("○")
		}

		// Connect lines
		connector := "│"
		if i == len(l.revisions)-1 {
			connector = " "
		}

		// First line: graph + change ID + workspace marker
		changeID := theme.ChangeIDStyle.Render(rev.ChangeID)
		revID := theme.RevisionIDStyle.Render(rev.ID)

		var wsMarker string
		if rev.IsWorkingCopy && rev.WorkspaceName != "" {
			wsMarker = " " + theme.WorkingCopyStyle.Render(rev.WorkspaceName+"@")
		}

		line1 := fmt.Sprintf("%s %s %s%s", graphChar, changeID, revID, wsMarker)

		// Highlight selected line
		if i == l.cursor {
			line1 = theme.SelectedItemStyle.Render(line1)
		}

		lines = append(lines, line1)

		// Second line: connector + description
		desc := rev.Description
		if len(desc) > contentWidth-4 {
			desc = desc[:contentWidth-7] + "..."
		}

		var styledDesc string
		if rev.Description == "" {
			styledDesc = theme.DimmedStyle.Italic(true).Render("(no description)")
		} else {
			styledDesc = theme.NormalItemStyle.Render(desc)
		}

		line2 := theme.DimmedStyle.Render(connector) + " " + styledDesc
		lines = append(lines, line2)

		// Third line: connector + author + timestamp
		meta := fmt.Sprintf("%s %s",
			theme.AuthorStyle.Render(rev.Author),
			theme.TimestampStyle.Render(rev.Timestamp),
		)
		line3 := theme.DimmedStyle.Render(connector) + " " + meta
		lines = append(lines, line3)

		// Empty line between revisions (except last)
		if i < len(l.revisions)-1 {
			lines = append(lines, theme.DimmedStyle.Render(connector))
		}
	}

	return strings.Join(lines, "\n")
}

func (l *LogOverlay) renderFrame(content string) string {
	title := theme.FloatingTitleStyle.Render(" Log ")

	// Create styled content area
	contentStyle := lipgloss.NewStyle().
		Width(l.width - 2).
		Height(l.height - 3).
		Padding(0, 1)

	styledContent := contentStyle.Render(content)

	// Combine title and content
	inner := lipgloss.JoinVertical(lipgloss.Left, title, styledContent)

	// Apply floating window style
	return theme.FloatingWindowStyle.
		Width(l.width).
		Height(l.height).
		Render(inner)
}

// SelectedRevision returns the currently selected revision
func (l *LogOverlay) SelectedRevision() *fixtures.Revision {
	if l.cursor >= 0 && l.cursor < len(l.revisions) {
		return &l.revisions[l.cursor]
	}
	return nil
}
