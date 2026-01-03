package graph

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Symbols used for graph rendering (jj-style)
const (
	SymbolWorkingCopy = "@"
	SymbolCommit      = "○"
	SymbolRoot        = "◆"
	SymbolVertical    = "│"
	SymbolMergeLeft   = "├"
	SymbolMergeRight  = "┤"
	SymbolBranch      = "┬"
	SymbolJoin        = "┴"
	SymbolCross       = "┼"
	SymbolHorizontal  = "─"
	SymbolSpace       = " "
)

// RevisionInfo contains the information needed to render a revision in the graph.
type RevisionInfo struct {
	ID            string
	Parents       []string
	IsWorkingCopy bool
	IsRoot        bool
}

// GraphLine represents one line of the graph output.
type GraphLine struct {
	Prefix string // The graph characters (e.g., "│ @ ")
	Column int    // Which column this revision is in
}

// Renderer generates ASCII graph lines for a revision log.
type Renderer struct {
	activeColumns []string         // Currently active columns (waiting for their commit to appear)
	columnMap     map[string]int   // Maps commit ID to column position
	workingStyle  lipgloss.Style   // Style for working copy symbol
	commitStyle   lipgloss.Style   // Style for normal commit symbol
	rootStyle     lipgloss.Style   // Style for root symbol
	lineStyle     lipgloss.Style   // Style for connecting lines
}

// NewRenderer creates a new graph renderer with the given styles.
func NewRenderer(workingStyle, commitStyle, rootStyle, lineStyle lipgloss.Style) *Renderer {
	return &Renderer{
		activeColumns: nil,
		columnMap:     make(map[string]int),
		workingStyle:  workingStyle,
		commitStyle:   commitStyle,
		rootStyle:     rootStyle,
		lineStyle:     lineStyle,
	}
}

// Reset clears the renderer state for a new graph.
func (r *Renderer) Reset() {
	r.activeColumns = nil
	r.columnMap = make(map[string]int)
}

// RenderRevision generates the graph prefix for a single revision.
// Call this for each revision in order (topological order, newest first).
func (r *Renderer) RenderRevision(rev RevisionInfo) GraphLine {
	// Find the column for this revision
	column := r.findColumn(rev.ID)

	// Build the graph line
	var parts []string

	// Draw columns before this one
	for i := 0; i < column; i++ {
		if i < len(r.activeColumns) && r.activeColumns[i] != "" {
			parts = append(parts, r.lineStyle.Render(SymbolVertical))
		} else {
			parts = append(parts, SymbolSpace)
		}
		parts = append(parts, SymbolSpace)
	}

	// Draw the commit symbol
	var symbol string
	if rev.IsWorkingCopy {
		symbol = r.workingStyle.Render(SymbolWorkingCopy)
	} else if rev.IsRoot {
		symbol = r.rootStyle.Render(SymbolRoot)
	} else {
		symbol = r.commitStyle.Render(SymbolCommit)
	}
	parts = append(parts, symbol)
	parts = append(parts, SymbolSpace)

	// Draw columns after this one
	for i := column + 1; i < len(r.activeColumns); i++ {
		if r.activeColumns[i] != "" {
			parts = append(parts, r.lineStyle.Render(SymbolVertical))
		} else {
			parts = append(parts, SymbolSpace)
		}
		parts = append(parts, SymbolSpace)
	}

	// Update active columns: remove this commit, add parents
	r.updateColumns(rev.ID, rev.Parents, column)

	return GraphLine{
		Prefix: strings.Join(parts, ""),
		Column: column,
	}
}

// RenderConnector generates the connector line between revisions.
func (r *Renderer) RenderConnector() string {
	var parts []string

	for i := 0; i < len(r.activeColumns); i++ {
		if r.activeColumns[i] != "" {
			parts = append(parts, r.lineStyle.Render(SymbolVertical))
		} else {
			parts = append(parts, SymbolSpace)
		}
		parts = append(parts, SymbolSpace)
	}

	return strings.Join(parts, "")
}

// findColumn determines which column a revision should appear in.
func (r *Renderer) findColumn(id string) int {
	// Check if we're expecting this commit in a specific column
	for i, colID := range r.activeColumns {
		if colID == id {
			return i
		}
	}

	// This commit wasn't expected (could be a root or new branch)
	// Find an empty column or add a new one
	for i, colID := range r.activeColumns {
		if colID == "" {
			return i
		}
	}

	// No empty columns, add a new one
	return len(r.activeColumns)
}

// updateColumns updates the active columns after processing a revision.
func (r *Renderer) updateColumns(id string, parents []string, column int) {
	// Ensure we have enough columns
	for len(r.activeColumns) <= column {
		r.activeColumns = append(r.activeColumns, "")
	}

	// Clear this column (we just drew this commit)
	r.activeColumns[column] = ""

	// Add parents to columns
	if len(parents) == 0 {
		// No parents (root commit), column stays empty
		return
	}

	// First parent takes this column
	r.activeColumns[column] = parents[0]

	// Additional parents need new columns
	for i := 1; i < len(parents); i++ {
		parentID := parents[i]
		// Find empty column or add new one
		placed := false
		for j := range r.activeColumns {
			if r.activeColumns[j] == "" {
				r.activeColumns[j] = parentID
				placed = true
				break
			}
		}
		if !placed {
			r.activeColumns = append(r.activeColumns, parentID)
		}
	}

	// Compact: remove trailing empty columns
	for len(r.activeColumns) > 0 && r.activeColumns[len(r.activeColumns)-1] == "" {
		r.activeColumns = r.activeColumns[:len(r.activeColumns)-1]
	}
}

// Simple renders a simple single-column graph (no merges/branches).
// This is useful when parent information isn't reliable.
func Simple(rev RevisionInfo, isLast bool, workingStyle, commitStyle, rootStyle, lineStyle lipgloss.Style) (graphChar, connector string) {
	if rev.IsWorkingCopy {
		graphChar = workingStyle.Render(SymbolWorkingCopy)
	} else if rev.IsRoot {
		graphChar = rootStyle.Render(SymbolRoot)
	} else {
		graphChar = commitStyle.Render(SymbolCommit)
	}

	if isLast {
		connector = SymbolSpace
	} else {
		connector = lineStyle.Render(SymbolVertical)
	}

	return graphChar, connector
}
