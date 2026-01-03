package floating

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gerund/jayz/jj"
	"github.com/gerund/jayz/ui/borders"
	"github.com/gerund/jayz/ui/fixtures"
	"github.com/gerund/jayz/ui/graph"
	"github.com/gerund/jayz/ui/messages"
	"github.com/gerund/jayz/ui/prefix"
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

	// Unique prefix highlighting
	changeIDPrefixes   *prefix.IDSet
	revisionIDPrefixes *prefix.IDSet
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
	changeIDs := make([]string, len(revs))
	revisionIDs := make([]string, len(revs))

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
			Bookmarks:     rev.Bookmarks,
			GitHead:       rev.GitHead,
			IsWorkingCopy: rev.IsWorkingCopy,
			WorkspaceName: wsName,
			IsRoot:        rev.IsRoot,
			Parents:       rev.Parents,
		}
		// Store truncated IDs (8 chars) for unique prefix calculation
		changeID := rev.ChangeID
		if len(changeID) > 8 {
			changeID = changeID[:8]
		}
		revID := rev.ID
		if len(revID) > 8 {
			revID = revID[:8]
		}
		changeIDs[i] = changeID
		revisionIDs[i] = revID
	}

	// Compute unique prefixes for ID highlighting (on truncated 8-char IDs)
	l.changeIDPrefixes = prefix.NewIDSet(changeIDs)
	l.revisionIDPrefixes = prefix.NewIDSet(revisionIDs)
}

func (l *LogOverlay) Init() tea.Cmd {
	return nil
}

func (l *LogOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Handle scroll wheel for viewport
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			l.viewport.LineUp(3)
		case tea.MouseButtonWheelDown:
			l.viewport.LineDown(3)
		}

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
		case "enter":
			// Select revision and emit message
			if rev := l.SelectedRevision(); rev != nil {
				return l, func() tea.Msg {
					return messages.RevisionSelectedMsg{RevisionID: rev.ID}
				}
			}
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
	// Each revision takes 2 lines (graph+metadata line, description line)
	linePos := l.cursor * 2
	if linePos < l.viewport.YOffset {
		l.viewport.SetYOffset(linePos)
	} else if linePos >= l.viewport.YOffset+l.viewport.Height-2 {
		l.viewport.SetYOffset(linePos - l.viewport.Height + 2)
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

	// With titled borders: just top and bottom borders (title is in top border)
	contentWidth := width - 2
	contentHeight := height - 2

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

	for i, rev := range l.revisions {
		isLast := i == len(l.revisions)-1

		// Root commit: just show "~" with no other info
		if rev.IsRoot {
			lines = append(lines, theme.DimmedStyle.Render("~"))
			continue
		}

		// Use graph package for consistent symbols
		revInfo := graph.RevisionInfo{
			ID:            rev.ID,
			Parents:       rev.Parents,
			IsWorkingCopy: rev.IsWorkingCopy,
			IsRoot:        rev.IsRoot,
		}
		graphChar, connector := graph.Simple(revInfo, isLast,
			theme.WorkingCopyStyle,
			theme.DimmedStyle, // Use dimmed for normal commits (○)
			theme.DimmedStyle, // Use dimmed for root (◆)
			theme.DimmedStyle, // Connector line style
		)

		// Override symbol to ◆ (blue) if this revision has the "main" bookmark
		hasMain := false
		for _, bookmark := range rev.Bookmarks {
			if bookmark == "main" {
				hasMain = true
				break
			}
		}
		if hasMain {
			mainStyle := lipgloss.NewStyle().Foreground(theme.ColorBlue)
			graphChar = mainStyle.Render("◆")
		}

		// Line 1: graph + changeID(8 chars) + email + timestamp + bookmarks + revID(8 chars) [+ workspace marker]
		// Use unique prefix highlighting for IDs
		var changeID, revID string

		// Truncate change ID to 8 characters
		shortChangeID := rev.ChangeID
		if len(shortChangeID) > 8 {
			shortChangeID = shortChangeID[:8]
		}
		// Apply bold to styles if working copy
		changeIDPrefixStyle := theme.ChangeIDPrefixStyle
		changeIDRestStyle := theme.ChangeIDRestStyle
		changeIDBaseStyle := theme.ChangeIDStyle
		if rev.IsWorkingCopy {
			changeIDPrefixStyle = changeIDPrefixStyle.Bold(true)
			changeIDRestStyle = changeIDRestStyle.Bold(true)
			changeIDBaseStyle = changeIDBaseStyle.Bold(true)
		}
		if l.changeIDPrefixes != nil {
			changeID = l.changeIDPrefixes.Format(shortChangeID, changeIDPrefixStyle, changeIDRestStyle)
		} else {
			changeID = changeIDBaseStyle.Render(shortChangeID)
		}

		// Truncate revision ID to 8 characters
		shortRevID := rev.ID
		if len(shortRevID) > 8 {
			shortRevID = shortRevID[:8]
		}
		revIDPrefixStyle := theme.RevisionIDPrefixStyle
		revIDRestStyle := theme.RevisionIDRestStyle
		revIDBaseStyle := theme.RevisionIDStyle
		if rev.IsWorkingCopy {
			revIDPrefixStyle = revIDPrefixStyle.Bold(true)
			revIDRestStyle = revIDRestStyle.Bold(true)
			revIDBaseStyle = revIDBaseStyle.Bold(true)
		}
		if l.revisionIDPrefixes != nil {
			revID = l.revisionIDPrefixes.Format(shortRevID, revIDPrefixStyle, revIDRestStyle)
		} else {
			revID = revIDBaseStyle.Render(shortRevID)
		}

		authorStyle := theme.AuthorStyle
		timestampStyle := theme.TimestampStyle
		if rev.IsWorkingCopy {
			authorStyle = authorStyle.Bold(true)
			timestampStyle = timestampStyle.Bold(true)
		}
		email := authorStyle.Render(rev.Author)
		timestamp := timestampStyle.Render(rev.Timestamp)

		// Format bookmarks and git_head
		var bookmarksStr string
		if rev.GitHead || len(rev.Bookmarks) > 0 {
			var parts []string
			if rev.GitHead {
				gitHeadStyle := lipgloss.NewStyle().Foreground(theme.ColorGreen)
				if rev.IsWorkingCopy {
					gitHeadStyle = gitHeadStyle.Bold(true)
				}
				parts = append(parts, gitHeadStyle.Render("git_head()"))
			}
			if len(rev.Bookmarks) > 0 {
				bookmarkStyle := lipgloss.NewStyle().Foreground(theme.ColorMagenta)
				if rev.IsWorkingCopy {
					bookmarkStyle = bookmarkStyle.Bold(true)
				}
				parts = append(parts, bookmarkStyle.Render(strings.Join(rev.Bookmarks, " ")))
			}
			bookmarksStr = " " + strings.Join(parts, " ")
		}

		var wsMarker string
		if rev.IsWorkingCopy && rev.WorkspaceName != "" {
			wsMarker = " " + theme.WorkingCopyStyle.Render(rev.WorkspaceName+"@")
		}

		line1 := fmt.Sprintf("%s  %s %s %s%s %s%s", graphChar, changeID, email, timestamp, bookmarksStr, revID, wsMarker)

		// Apply selection style if this is the cursor line
		if i == l.cursor {
			line1 = theme.SelectedItemStyle.Render(line1)
		}

		lines = append(lines, line1)

		// Line 2: connector + description (first line only, like jj log)
		var styledDesc string
		if rev.Description == "" {
			descStyle := theme.DimmedStyle.Italic(true)
			if rev.IsWorkingCopy {
				descStyle = descStyle.Bold(true)
			}
			styledDesc = descStyle.Render("(no description)")
		} else {
			// Only show first line of description
			desc := rev.Description
			if idx := strings.Index(desc, "\n"); idx >= 0 {
				desc = desc[:idx]
			}
			descStyle := theme.NormalItemStyle
			if rev.IsWorkingCopy {
				descStyle = descStyle.Bold(true)
			}
			styledDesc = descStyle.Render(desc)
		}

		line2 := connector + " " + styledDesc
		lines = append(lines, line2)
	}

	return strings.Join(lines, "\n")
}

func (l *LogOverlay) renderFrame(content string) string {
	// Use lipgloss native border rendering to avoid ANSI escape sequence corruption
	// See: https://github.com/charmbracelet/lipgloss/issues/498

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorYellow).
		Width(l.width - 2).  // Account for border width
		Height(l.height - 2) // Account for border height

	// Render content with border
	bordered := borderStyle.Render(content)

	// Add title to top border
	lines := strings.Split(bordered, "\n")
	if len(lines) > 0 {
		// Build custom top border with title using a styled approach
		borderColorStyle := lipgloss.NewStyle().Foreground(theme.ColorYellow)
		styledTitle := theme.FloatingTitleStyle.Render(" Log ")

		titleWidth := lipgloss.Width(styledTitle)
		// Total: TopLeft(1) + Horizontal(1) + Title + Horizontal*(N) + TopRight(1) = width
		// So: 3 + titleWidth + N = width => N = width - 3 - titleWidth
		remainingWidth := l.width - 3 - titleWidth
		if remainingWidth < 0 {
			remainingWidth = 0
		}

		topBorder := borderColorStyle.Render(borders.TopLeft+borders.Horizontal) +
			styledTitle +
			borderColorStyle.Render(strings.Repeat(borders.Horizontal, remainingWidth)+borders.TopRight)

		lines[0] = topBorder
	}

	return strings.Join(lines, "\n")
}

// SelectedRevision returns the currently selected revision
func (l *LogOverlay) SelectedRevision() *fixtures.Revision {
	if l.cursor >= 0 && l.cursor < len(l.revisions) {
		return &l.revisions[l.cursor]
	}
	return nil
}

// ContentHeight returns the height needed to display all content (not including borders)
func (l *LogOverlay) ContentHeight() int {
	// Each revision takes 2 lines
	return len(l.revisions) * 2
}
