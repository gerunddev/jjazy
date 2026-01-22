package interactive

import (
	"testing"

	"github.com/gerunddev/jjazy/jj"
)

func TestBuildRevisionOptions(t *testing.T) {
	tests := []struct {
		name     string
		changes  []jj.ChangeInfo
		wantLen  int
		wantLabels []string
		wantValues []string
	}{
		{
			name:     "empty changes",
			changes:  []jj.ChangeInfo{},
			wantLen:  0,
		},
		{
			name: "single change not working copy",
			changes: []jj.ChangeInfo{
				{ChangeID: "abcd1234", CommitID: "deadbeef", IsWorkingCopy: false},
			},
			wantLen:    1,
			wantLabels: []string{"abcd1234 (no description)"},
			wantValues: []string{"abcd1234"},
		},
		{
			name: "single change is working copy",
			changes: []jj.ChangeInfo{
				{ChangeID: "wxyz9876", CommitID: "cafebabe", IsWorkingCopy: true},
			},
			wantLen:    1,
			wantLabels: []string{"wxyz9876 @ (no description)"},
			wantValues: []string{"wxyz9876"},
		},
		{
			name: "multiple changes with working copy",
			changes: []jj.ChangeInfo{
				{ChangeID: "aaaaaaaa", CommitID: "11111111", IsWorkingCopy: true},
				{ChangeID: "bbbbbbbb", CommitID: "22222222", IsWorkingCopy: false},
				{ChangeID: "cccccccc", CommitID: "33333333", IsWorkingCopy: false},
			},
			wantLen:    3,
			wantLabels: []string{"aaaaaaaa @ (no description)", "bbbbbbbb (no description)", "cccccccc (no description)"},
			wantValues: []string{"aaaaaaaa", "bbbbbbbb", "cccccccc"},
		},
		{
			name: "change with description",
			changes: []jj.ChangeInfo{
				{ChangeID: "desctest", CommitID: "abcd1234", Description: "Fix the bug", IsWorkingCopy: false},
			},
			wantLen:    1,
			wantLabels: []string{"desctest Fix the bug"},
			wantValues: []string{"desctest"},
		},
		{
			name: "change with bookmarks",
			changes: []jj.ChangeInfo{
				{ChangeID: "bookmark1", CommitID: "aaaa1111", Bookmarks: []string{"main", "feature"}, IsWorkingCopy: false},
			},
			wantLen:    1,
			wantLabels: []string{"bookmark1 [main, feature] (no description)"},
			wantValues: []string{"bookmark1"},
		},
		{
			name: "change with description and bookmarks",
			changes: []jj.ChangeInfo{
				{ChangeID: "fullinfo", CommitID: "bbbb2222", Description: "Add feature", Bookmarks: []string{"dev"}, IsWorkingCopy: true},
			},
			wantLen:    1,
			wantLabels: []string{"fullinfo @ [dev] Add feature"},
			wantValues: []string{"fullinfo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := buildRevisionOptions(tt.changes)

			if len(options) != tt.wantLen {
				t.Errorf("buildRevisionOptions() returned %d options, want %d", len(options), tt.wantLen)
				return
			}

			for i, opt := range options {
				// Get the key (label) and value from the option
				// huh.Option stores Key as the display string and Value as the value
				gotLabel := opt.Key
				gotValue := opt.Value

				if i < len(tt.wantLabels) && gotLabel != tt.wantLabels[i] {
					t.Errorf("option[%d] label = %q, want %q", i, gotLabel, tt.wantLabels[i])
				}

				if i < len(tt.wantValues) && gotValue != tt.wantValues[i] {
					t.Errorf("option[%d] value = %q, want %q", i, gotValue, tt.wantValues[i])
				}
			}
		})
	}
}

func TestBuildRevisionOptionsPreservesOrder(t *testing.T) {
	changes := []jj.ChangeInfo{
		{ChangeID: "first", CommitID: "111", IsWorkingCopy: false},
		{ChangeID: "second", CommitID: "222", IsWorkingCopy: true},
		{ChangeID: "third", CommitID: "333", IsWorkingCopy: false},
	}

	options := buildRevisionOptions(changes)

	if len(options) != 3 {
		t.Fatalf("expected 3 options, got %d", len(options))
	}

	expectedOrder := []string{"first", "second", "third"}
	for i, opt := range options {
		if opt.Value != expectedOrder[i] {
			t.Errorf("option[%d] value = %q, want %q (order not preserved)", i, opt.Value, expectedOrder[i])
		}
	}
}
