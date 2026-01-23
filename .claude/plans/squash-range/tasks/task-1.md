# Task 1: Add Ancestor Detection to Navigation

## Objective
Add utilities to the Navigation module for determining ancestor/descendant relationships between commits.

## Files to Modify
- `app/navigation.go`
- `app/navigation_test.go` (new file)

## Implementation

### 1. Add IsAncestor Method

```go
// IsAncestor returns true if potentialAncestor is an ancestor of potentialDescendant.
// Uses BFS walking from descendant up through parents.
func (n *Navigation) IsAncestor(potentialAncestor, potentialDescendant string) bool {
    if potentialAncestor == potentialDescendant {
        return false // A commit is not its own ancestor
    }

    visited := make(map[string]bool)
    queue := []string{potentialDescendant}

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

        // Check if any parent is the potential ancestor
        for _, parentID := range rev.Parents {
            if parentID == potentialAncestor {
                return true
            }
            if !visited[parentID] {
                queue = append(queue, parentID)
            }
        }
    }

    return false
}
```

### 2. Add OrderByAncestry Method

```go
// OrderByAncestry determines which of two commits is the ancestor (lower) and which
// is the descendant (higher). Returns error if commits are on divergent branches.
func (n *Navigation) OrderByAncestry(changeA, changeB string) (lower, higher string, err error) {
    if changeA == changeB {
        return "", "", fmt.Errorf("cannot order: same commit selected twice")
    }

    // Check if A is ancestor of B
    if n.IsAncestor(changeA, changeB) {
        return changeA, changeB, nil
    }

    // Check if B is ancestor of A
    if n.IsAncestor(changeB, changeA) {
        return changeB, changeA, nil
    }

    // Neither is ancestor of the other - divergent branches
    return "", "", fmt.Errorf("commits are on divergent branches: %s and %s are not in a direct ancestor/descendant relationship", changeA, changeB)
}
```

### 3. Add GetParent Method (for "lower-" computation)

```go
// GetParentChangeID returns the change ID of the first parent of a revision.
// Returns empty string if revision has no parent (root).
func (n *Navigation) GetParentChangeID(changeID string) string {
    // Find revision by change ID
    for _, rev := range n.revisions {
        // Match by either short ID or full change ID
        if rev.ID == changeID || rev.ChangeID == changeID ||
           strings.HasPrefix(rev.ChangeID, changeID) {
            if len(rev.Parents) > 0 {
                // Return the parent's change ID, not commit ID
                parentRev := n.revMap[rev.Parents[0]]
                if parentRev != nil {
                    return parentRev.ChangeID[:8] // Return short form
                }
            }
            return ""
        }
    }
    return ""
}
```

### 4. Add FindRevisionByChangeID Helper

```go
// FindRevisionByChangeID finds a revision by change ID (short or full).
func (n *Navigation) FindRevisionByChangeID(changeID string) *jj.Revision {
    for i := range n.revisions {
        if n.revisions[i].ChangeID == changeID ||
           strings.HasPrefix(n.revisions[i].ChangeID, changeID) {
            return &n.revisions[i]
        }
    }
    return nil
}
```

## Unit Tests

Create `app/navigation_test.go`:

```go
package app

import (
    "testing"
    "github.com/gerunddev/jjazy/jj"
)

func TestIsAncestor(t *testing.T) {
    // Build test DAG:
    //   A (root)
    //   |
    //   B
    //   |
    //   C (working copy)
    revisions := []jj.Revision{
        {ID: "c123", ChangeID: "cccccccc", Parents: []string{"b123"}},
        {ID: "b123", ChangeID: "bbbbbbbb", Parents: []string{"a123"}},
        {ID: "a123", ChangeID: "aaaaaaaa", Parents: []string{}},
    }

    nav := NewNavigation("/test", revisions)

    tests := []struct {
        name       string
        ancestor   string
        descendant string
        want       bool
    }{
        {"A is ancestor of C", "a123", "c123", true},
        {"B is ancestor of C", "b123", "c123", true},
        {"A is ancestor of B", "a123", "b123", true},
        {"C is not ancestor of A", "c123", "a123", false},
        {"Same commit", "b123", "b123", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := nav.IsAncestor(tt.ancestor, tt.descendant)
            if got != tt.want {
                t.Errorf("IsAncestor(%s, %s) = %v, want %v",
                    tt.ancestor, tt.descendant, got, tt.want)
            }
        })
    }
}

func TestOrderByAncestry(t *testing.T) {
    revisions := []jj.Revision{
        {ID: "c123", ChangeID: "cccccccc", Parents: []string{"b123"}},
        {ID: "b123", ChangeID: "bbbbbbbb", Parents: []string{"a123"}},
        {ID: "a123", ChangeID: "aaaaaaaa", Parents: []string{}},
    }

    nav := NewNavigation("/test", revisions)

    // Test normal ordering
    lower, higher, err := nav.OrderByAncestry("a123", "c123")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if lower != "a123" || higher != "c123" {
        t.Errorf("OrderByAncestry(a,c) = (%s,%s), want (a123,c123)", lower, higher)
    }

    // Test reversed input
    lower, higher, err = nav.OrderByAncestry("c123", "a123")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if lower != "a123" || higher != "c123" {
        t.Errorf("OrderByAncestry(c,a) = (%s,%s), want (a123,c123)", lower, higher)
    }

    // Test same commit error
    _, _, err = nav.OrderByAncestry("b123", "b123")
    if err == nil {
        t.Error("expected error for same commit, got nil")
    }
}

func TestOrderByAncestry_DivergentBranches(t *testing.T) {
    // Build test DAG with divergent branches:
    //     A (root)
    //    / \
    //   B   C
    revisions := []jj.Revision{
        {ID: "b123", ChangeID: "bbbbbbbb", Parents: []string{"a123"}},
        {ID: "c123", ChangeID: "cccccccc", Parents: []string{"a123"}},
        {ID: "a123", ChangeID: "aaaaaaaa", Parents: []string{}},
    }

    nav := NewNavigation("/test", revisions)

    _, _, err := nav.OrderByAncestry("b123", "c123")
    if err == nil {
        t.Error("expected error for divergent branches, got nil")
    }
}
```

## Acceptance Criteria

- [ ] `IsAncestor` correctly identifies ancestor relationships
- [ ] `OrderByAncestry` returns commits in correct order (ancestor, descendant)
- [ ] `OrderByAncestry` handles reversed input correctly
- [ ] `OrderByAncestry` returns error for same commit
- [ ] `OrderByAncestry` returns error for divergent branches
- [ ] All unit tests pass
- [ ] Methods handle edge cases (missing commits, root commit)

## Dependencies
- None (independent task)

## Estimated Effort
1-2 hours
