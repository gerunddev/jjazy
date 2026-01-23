package app

import (
	"errors"
	"testing"

	"github.com/gerunddev/jjazy/jj"
)

// Helper to create a test revision with specified ID and parents
func testRevision(id string, parents ...string) jj.Revision {
	return jj.Revision{
		ID:      id,
		Parents: parents,
	}
}

// Helper to create a test revision with specified ID, ChangeID, and parents
func testRevisionWithChangeID(id, changeID string, parents ...string) jj.Revision {
	return jj.Revision{
		ID:       id,
		ChangeID: changeID,
		Parents:  parents,
	}
}

func TestIsAncestor_LinearChain(t *testing.T) {
	// Linear chain: A <- B <- C <- D (D is newest, A is oldest/root)
	revisions := []jj.Revision{
		testRevision("D", "C"),
		testRevision("C", "B"),
		testRevision("B", "A"),
		testRevision("A"),
	}
	nav := NewNavigation("", revisions)

	tests := []struct {
		name       string
		ancestorID string
		descendant string
		want       bool
	}{
		{"A is ancestor of D", "A", "D", true},
		{"A is ancestor of C", "A", "C", true},
		{"A is ancestor of B", "A", "B", true},
		{"B is ancestor of D", "B", "D", true},
		{"B is ancestor of C", "B", "C", true},
		{"C is ancestor of D", "C", "D", true},
		{"D is not ancestor of A", "D", "A", false},
		{"D is not ancestor of B", "D", "B", false},
		{"C is not ancestor of A", "C", "A", false},
		{"A is not its own ancestor", "A", "A", false},
		{"D is not its own ancestor", "D", "D", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nav.IsAncestor(tt.ancestorID, tt.descendant)
			if got != tt.want {
				t.Errorf("IsAncestor(%q, %q) = %v, want %v", tt.ancestorID, tt.descendant, got, tt.want)
			}
		})
	}
}

func TestIsAncestor_BranchingScenario(t *testing.T) {
	// Branching:
	//     C
	//    /
	// A-B
	//    \
	//     D
	revisions := []jj.Revision{
		testRevision("C", "B"),
		testRevision("D", "B"),
		testRevision("B", "A"),
		testRevision("A"),
	}
	nav := NewNavigation("", revisions)

	tests := []struct {
		name       string
		ancestorID string
		descendant string
		want       bool
	}{
		{"A is ancestor of C", "A", "C", true},
		{"A is ancestor of D", "A", "D", true},
		{"B is ancestor of C", "B", "C", true},
		{"B is ancestor of D", "B", "D", true},
		{"C is not ancestor of D (divergent)", "C", "D", false},
		{"D is not ancestor of C (divergent)", "D", "C", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nav.IsAncestor(tt.ancestorID, tt.descendant)
			if got != tt.want {
				t.Errorf("IsAncestor(%q, %q) = %v, want %v", tt.ancestorID, tt.descendant, got, tt.want)
			}
		})
	}
}

func TestIsAncestor_MergeCommit(t *testing.T) {
	// Merge scenario:
	//   B   C
	//    \ /
	//     D (merge commit with two parents)
	//     |
	//     A
	revisions := []jj.Revision{
		testRevision("D", "B", "C"),
		testRevision("B", "A"),
		testRevision("C", "A"),
		testRevision("A"),
	}
	nav := NewNavigation("", revisions)

	tests := []struct {
		name       string
		ancestorID string
		descendant string
		want       bool
	}{
		{"A is ancestor of D", "A", "D", true},
		{"B is ancestor of D", "B", "D", true},
		{"C is ancestor of D", "C", "D", true},
		{"D is not ancestor of A", "D", "A", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nav.IsAncestor(tt.ancestorID, tt.descendant)
			if got != tt.want {
				t.Errorf("IsAncestor(%q, %q) = %v, want %v", tt.ancestorID, tt.descendant, got, tt.want)
			}
		})
	}
}

func TestOrderByAncestry_LinearChain(t *testing.T) {
	// Linear chain: A <- B <- C
	revisions := []jj.Revision{
		testRevision("C", "B"),
		testRevision("B", "A"),
		testRevision("A"),
	}
	nav := NewNavigation("", revisions)

	tests := []struct {
		name             string
		id1              string
		id2              string
		wantAncestor     string
		wantDescendant   string
	}{
		{"A and C - A first", "A", "C", "A", "C"},
		{"C and A - C first", "C", "A", "A", "C"},
		{"A and B", "A", "B", "A", "B"},
		{"B and A", "B", "A", "A", "B"},
		{"B and C", "B", "C", "B", "C"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ancestor, descendant, err := nav.OrderByAncestry(tt.id1, tt.id2)
			if err != nil {
				t.Fatalf("OrderByAncestry(%q, %q) unexpected error: %v", tt.id1, tt.id2, err)
			}
			if ancestor != tt.wantAncestor || descendant != tt.wantDescendant {
				t.Errorf("OrderByAncestry(%q, %q) = (%q, %q), want (%q, %q)",
					tt.id1, tt.id2, ancestor, descendant, tt.wantAncestor, tt.wantDescendant)
			}
		})
	}
}

func TestOrderByAncestry_SameCommitError(t *testing.T) {
	revisions := []jj.Revision{
		testRevision("A"),
	}
	nav := NewNavigation("", revisions)

	_, _, err := nav.OrderByAncestry("A", "A")
	if !errors.Is(err, ErrSameCommit) {
		t.Errorf("OrderByAncestry(A, A) error = %v, want ErrSameCommit", err)
	}
}

func TestOrderByAncestry_DivergentBranchesError(t *testing.T) {
	// Branching:
	//     C
	//    /
	// A-B
	//    \
	//     D
	revisions := []jj.Revision{
		testRevision("C", "B"),
		testRevision("D", "B"),
		testRevision("B", "A"),
		testRevision("A"),
	}
	nav := NewNavigation("", revisions)

	_, _, err := nav.OrderByAncestry("C", "D")
	if !errors.Is(err, ErrDivergentBranches) {
		t.Errorf("OrderByAncestry(C, D) error = %v, want ErrDivergentBranches", err)
	}

	_, _, err = nav.OrderByAncestry("D", "C")
	if !errors.Is(err, ErrDivergentBranches) {
		t.Errorf("OrderByAncestry(D, C) error = %v, want ErrDivergentBranches", err)
	}
}

func TestOrderByAncestry_BranchingWithCommonAncestor(t *testing.T) {
	// Branching:
	//     C
	//    /
	// A-B
	//    \
	//     D
	revisions := []jj.Revision{
		testRevision("C", "B"),
		testRevision("D", "B"),
		testRevision("B", "A"),
		testRevision("A"),
	}
	nav := NewNavigation("", revisions)

	// B is ancestor of both C and D
	tests := []struct {
		name             string
		id1              string
		id2              string
		wantAncestor     string
		wantDescendant   string
	}{
		{"B and C", "B", "C", "B", "C"},
		{"B and D", "B", "D", "B", "D"},
		{"A and C", "A", "C", "A", "C"},
		{"A and D", "A", "D", "A", "D"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ancestor, descendant, err := nav.OrderByAncestry(tt.id1, tt.id2)
			if err != nil {
				t.Fatalf("OrderByAncestry(%q, %q) unexpected error: %v", tt.id1, tt.id2, err)
			}
			if ancestor != tt.wantAncestor || descendant != tt.wantDescendant {
				t.Errorf("OrderByAncestry(%q, %q) = (%q, %q), want (%q, %q)",
					tt.id1, tt.id2, ancestor, descendant, tt.wantAncestor, tt.wantDescendant)
			}
		})
	}
}

func TestIsAncestor_UnknownRevision(t *testing.T) {
	revisions := []jj.Revision{
		testRevision("A"),
	}
	nav := NewNavigation("", revisions)

	// Testing with unknown revision should return false (not panic)
	got := nav.IsAncestor("unknown", "A")
	if got != false {
		t.Errorf("IsAncestor(unknown, A) = %v, want false", got)
	}

	got = nav.IsAncestor("A", "unknown")
	if got != false {
		t.Errorf("IsAncestor(A, unknown) = %v, want false", got)
	}
}

func TestFindByChangeID_ExactMatch(t *testing.T) {
	revisions := []jj.Revision{
		testRevisionWithChangeID("commit1", "abcdef12", "root"),
		testRevisionWithChangeID("commit2", "xyz98765", "commit1"),
		testRevisionWithChangeID("root", "rootchan"),
	}
	nav := NewNavigation("", revisions)

	tests := []struct {
		name     string
		changeID string
		wantID   string
		wantOK   bool
	}{
		{"exact match first", "abcdef12", "commit1", true},
		{"exact match second", "xyz98765", "commit2", true},
		{"exact match root", "rootchan", "root", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rev, ok := nav.FindByChangeID(tt.changeID)
			if ok != tt.wantOK {
				t.Errorf("FindByChangeID(%q) ok = %v, want %v", tt.changeID, ok, tt.wantOK)
				return
			}
			if ok && rev.ID != tt.wantID {
				t.Errorf("FindByChangeID(%q) ID = %q, want %q", tt.changeID, rev.ID, tt.wantID)
			}
		})
	}
}

func TestFindByChangeID_PrefixMatch(t *testing.T) {
	revisions := []jj.Revision{
		testRevisionWithChangeID("commit1", "abcdef123456", "root"),
		testRevisionWithChangeID("commit2", "xyz987654321", "commit1"),
		testRevisionWithChangeID("root", "rootchangeid"),
	}
	nav := NewNavigation("", revisions)

	tests := []struct {
		name     string
		changeID string
		wantID   string
		wantOK   bool
	}{
		{"short prefix abc", "abc", "commit1", true},
		{"medium prefix abcdef", "abcdef", "commit1", true},
		{"longer prefix abcdef12", "abcdef12", "commit1", true},
		{"full change ID", "abcdef123456", "commit1", true},
		{"prefix xyz", "xyz", "commit2", true},
		{"prefix xyz98", "xyz98", "commit2", true},
		{"prefix root", "root", "root", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rev, ok := nav.FindByChangeID(tt.changeID)
			if ok != tt.wantOK {
				t.Errorf("FindByChangeID(%q) ok = %v, want %v", tt.changeID, ok, tt.wantOK)
				return
			}
			if ok && rev.ID != tt.wantID {
				t.Errorf("FindByChangeID(%q) ID = %q, want %q", tt.changeID, rev.ID, tt.wantID)
			}
		})
	}
}

func TestFindByChangeID_NoMatch(t *testing.T) {
	revisions := []jj.Revision{
		testRevisionWithChangeID("commit1", "abcdef12", "root"),
		testRevisionWithChangeID("root", "rootchan"),
	}
	nav := NewNavigation("", revisions)

	tests := []struct {
		name     string
		changeID string
	}{
		{"completely unknown", "zzzzzzz"},
		{"partial mismatch", "abd"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rev, ok := nav.FindByChangeID(tt.changeID)
			if ok {
				t.Errorf("FindByChangeID(%q) = (%v, true), want (nil, false)", tt.changeID, rev)
			}
		})
	}
}

func TestFindByChangeID_AmbiguousPrefix(t *testing.T) {
	// Two revisions with change IDs that share a common prefix
	revisions := []jj.Revision{
		testRevisionWithChangeID("commit1", "abc123def", "root"),
		testRevisionWithChangeID("commit2", "abc456ghi", "commit1"),
		testRevisionWithChangeID("root", "rootchan"),
	}
	nav := NewNavigation("", revisions)

	tests := []struct {
		name     string
		changeID string
		wantOK   bool
		wantID   string
	}{
		{"ambiguous prefix abc", "abc", false, ""},
		{"unique prefix abc1", "abc1", true, "commit1"},
		{"unique prefix abc4", "abc4", true, "commit2"},
		{"full ID abc123def", "abc123def", true, "commit1"},
		{"full ID abc456ghi", "abc456ghi", true, "commit2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rev, ok := nav.FindByChangeID(tt.changeID)
			if ok != tt.wantOK {
				t.Errorf("FindByChangeID(%q) ok = %v, want %v", tt.changeID, ok, tt.wantOK)
				return
			}
			if ok && rev.ID != tt.wantID {
				t.Errorf("FindByChangeID(%q) ID = %q, want %q", tt.changeID, rev.ID, tt.wantID)
			}
		})
	}
}

func TestFindByChangeID_EmptyChangeID(t *testing.T) {
	// Revisions without change IDs should not cause issues
	revisions := []jj.Revision{
		testRevision("commit1", "root"), // No ChangeID
		testRevisionWithChangeID("commit2", "validchange", "commit1"),
		testRevision("root"), // No ChangeID
	}
	nav := NewNavigation("", revisions)

	// Should find revision with valid change ID
	rev, ok := nav.FindByChangeID("valid")
	if !ok {
		t.Error("FindByChangeID(\"valid\") should find a match")
		return
	}
	if rev.ID != "commit2" {
		t.Errorf("FindByChangeID(\"valid\") ID = %q, want \"commit2\"", rev.ID)
	}

	// Should not find anything for empty input
	_, ok = nav.FindByChangeID("")
	if ok {
		t.Error("FindByChangeID(\"\") should return false for empty input")
	}
}
