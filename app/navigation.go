package app

import (
	"github.com/gerund/jjazy/jj"
)

// Navigation handles bookmark/revision navigation logic.
// This abstracts the app-specific traversal logic from both UI and jj-lib.
type Navigation struct {
	repoPath  string
	revisions []jj.Revision
	revMap    map[string]*jj.Revision // ID -> Revision
	children  map[string][]string     // ID -> child IDs
}

// NewNavigation creates a Navigation from log revisions.
func NewNavigation(repoPath string, revisions []jj.Revision) *Navigation {
	n := &Navigation{
		repoPath:  repoPath,
		revisions: revisions,
		revMap:    make(map[string]*jj.Revision),
		children:  make(map[string][]string),
	}

	// Build revision map
	for i := range revisions {
		n.revMap[revisions[i].ID] = &revisions[i]
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
