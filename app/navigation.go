package app

import (
	"fmt"
	"strings"

	"github.com/gerunddev/jjazy/jj"
)

// Navigation handles bookmark/revision navigation logic.
// This abstracts the app-specific traversal logic from both UI and jj-lib.
type Navigation struct {
	repoPath    string
	revisions   []jj.Revision
	revMap      map[string]*jj.Revision // ID -> Revision
	changeIDMap map[string]*jj.Revision // ChangeID -> Revision
	children    map[string][]string     // ID -> child IDs
}

// NewNavigation creates a Navigation from log revisions.
func NewNavigation(repoPath string, revisions []jj.Revision) *Navigation {
	n := &Navigation{
		repoPath:    repoPath,
		revisions:   revisions,
		revMap:      make(map[string]*jj.Revision),
		changeIDMap: make(map[string]*jj.Revision),
		children:    make(map[string][]string),
	}

	// Build revision map and change ID map
	for i := range revisions {
		n.revMap[revisions[i].ID] = &revisions[i]
		if revisions[i].ChangeID != "" {
			n.changeIDMap[revisions[i].ChangeID] = &revisions[i]
		}
	}

	// Build children map (reverse of Parents)
	for i := range revisions {
		rev := &revisions[i]
		for _, parentID := range rev.Parents {
			n.children[parentID] = append(n.children[parentID], rev.ID)
		}
	}

	return n
}

// FindCurrentBookmark returns the closest bookmark to working copy by traversing
// down through ancestors. Returns empty string if no bookmark found.
func (n *Navigation) FindCurrentBookmark() string {
	// Find working copy
	var workingCopy *jj.Revision
	for i := range n.revisions {
		if n.revisions[i].IsWorkingCopy {
			workingCopy = &n.revisions[i]
			break
		}
	}
	if workingCopy == nil {
		return ""
	}

	// BFS through ancestors to find first bookmark
	visited := make(map[string]bool)
	queue := []string{workingCopy.ID}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		if visited[id] {
			continue
		}
		visited[id] = true

		rev := n.revMap[id]
		if rev == nil {
			continue
		}

		// Check if this revision has bookmarks
		if len(rev.Bookmarks) > 0 {
			return rev.Bookmarks[0] // Return first bookmark found
		}

		// Add parents to queue
		queue = append(queue, rev.Parents...)
	}

	return ""
}

// FindBookmarkEditTarget returns the revision to edit for a bookmark.
// It traverses UP from the bookmark toward the tip:
// - Returns the tip if no children exist
// - Returns the last revision before a different bookmark is encountered
func (n *Navigation) FindBookmarkEditTarget(bookmarkName string) *jj.Revision {
	// Find revision with this bookmark
	var startRev *jj.Revision
	for i := range n.revisions {
		for _, bm := range n.revisions[i].Bookmarks {
			if bm == bookmarkName {
				startRev = &n.revisions[i]
				break
			}
		}
		if startRev != nil {
			break
		}
	}
	if startRev == nil {
		return nil
	}

	// Traverse UP through children toward tip
	current := startRev
	for {
		childIDs := n.children[current.ID]

		if len(childIDs) == 0 {
			// No children = tip of branch
			return current
		}

		// Find child that continues this bookmark's lineage
		var nextChild *jj.Revision
		for _, childID := range childIDs {
			child := n.revMap[childID]
			if child == nil {
				continue
			}

			// Check if child has a different bookmark
			hasDifferentBookmark := false
			for _, bm := range child.Bookmarks {
				if bm != bookmarkName {
					hasDifferentBookmark = true
					break
				}
			}

			if hasDifferentBookmark {
				// Stop here - edit current (boundary before new bookmark)
				return current
			}

			// Continue with this child
			nextChild = child
			break // Take first valid child
		}

		if nextChild == nil {
			return current
		}
		current = nextChild
	}
}

// EditRevision executes jj edit for a revision.
func (n *Navigation) EditRevision(changeID string) error {
	return jj.Edit(n.repoPath, changeID)
}

// FindByChangeID looks up a revision by change ID, supporting both partial
// (prefix) and full change IDs. Returns the matching revision and true if
// exactly one match is found, or nil and false if no match or multiple
// matches exist.
func (n *Navigation) FindByChangeID(changeID string) (*jj.Revision, bool) {
	if changeID == "" {
		return nil, false
	}

	// Try exact match first
	if rev, ok := n.changeIDMap[changeID]; ok {
		return rev, true
	}

	// Try prefix match
	var match *jj.Revision
	for fullID, rev := range n.changeIDMap {
		if strings.HasPrefix(fullID, changeID) {
			if match != nil {
				// Multiple matches - ambiguous
				return nil, false
			}
			match = rev
		}
	}

	if match != nil {
		return match, true
	}

	return nil, false
}

// ErrSameCommit is returned when both commits are the same.
var ErrSameCommit = fmt.Errorf("cannot order: both commits are the same")

// ErrDivergentBranches is returned when commits are on divergent branches.
var ErrDivergentBranches = fmt.Errorf("cannot order: commits are on divergent branches")

// IsAncestor checks if ancestorID is an ancestor of descendantID by walking
// the parent chain using BFS.
func (n *Navigation) IsAncestor(ancestorID, descendantID string) bool {
	if ancestorID == descendantID {
		return false // A commit is not its own ancestor
	}

	// BFS through ancestors of descendantID looking for ancestorID
	visited := make(map[string]bool)
	queue := []string{descendantID}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		if visited[id] {
			continue
		}
		visited[id] = true

		rev := n.revMap[id]
		if rev == nil {
			continue
		}

		// Check parents
		for _, parentID := range rev.Parents {
			if parentID == ancestorID {
				return true
			}
			queue = append(queue, parentID)
		}
	}

	return false
}

// OrderByAncestry determines which of two commits is the ancestor and which
// is the descendant. Returns (ancestor, descendant) or an error if the commits
// are the same or on divergent branches with no ancestry relationship.
func (n *Navigation) OrderByAncestry(id1, id2 string) (ancestor, descendant string, err error) {
	if id1 == id2 {
		return "", "", ErrSameCommit
	}

	// Check if id1 is ancestor of id2
	if n.IsAncestor(id1, id2) {
		return id1, id2, nil
	}

	// Check if id2 is ancestor of id1
	if n.IsAncestor(id2, id1) {
		return id2, id1, nil
	}

	// Neither is an ancestor of the other - divergent branches
	return "", "", ErrDivergentBranches
}
