package ui

import (
	"testing"
)

// TestHelpBarContextFields verifies the HelpBarContext structure has all required fields
func TestHelpBarContextFields(t *testing.T) {
	ctx := HelpBarContext{
		Experience:    ExperienceChange,
		FocusedPanel:  1,
		Entered:       false,
		IsWorkingCopy: true,
	}

	if ctx.Experience != ExperienceChange {
		t.Errorf("Expected Experience to be ExperienceChange, got %v", ctx.Experience)
	}
	if ctx.FocusedPanel != 1 {
		t.Errorf("Expected FocusedPanel to be 1, got %d", ctx.FocusedPanel)
	}
	if ctx.Entered {
		t.Errorf("Expected Entered to be false, got %v", ctx.Entered)
	}
	if !ctx.IsWorkingCopy {
		t.Errorf("Expected IsWorkingCopy to be true, got %v", ctx.IsWorkingCopy)
	}
}

// TestGetActionHintsExperienceChangeFilesPanel tests file operation hints in Change experience
func TestGetActionHintsExperienceChangeFilesPanel(t *testing.T) {
	tests := []struct {
		name          string
		ctx           HelpBarContext
		expectHints   bool
		expectedCount int
		expectedKeys  []string
	}{
		{
			name: "Working copy with files panel focused",
			ctx: HelpBarContext{
				Experience:    ExperienceChange,
				FocusedPanel:  1,
				Entered:       false,
				IsWorkingCopy: true,
			},
			expectHints:   true,
			expectedCount: 2,
			expectedKeys:  []string{"del", "s"},
		},
		{
			name: "Non-working copy with files panel focused",
			ctx: HelpBarContext{
				Experience:    ExperienceChange,
				FocusedPanel:  1,
				Entered:       false,
				IsWorkingCopy: false,
			},
			expectHints:   false,
			expectedCount: 0,
		},
		{
			name: "Working copy with diff panel focused",
			ctx: HelpBarContext{
				Experience:    ExperienceChange,
				FocusedPanel:  0,
				Entered:       false,
				IsWorkingCopy: true,
			},
			expectHints:   false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := getActionHints(tt.ctx)

			if len(hints) != tt.expectedCount {
				t.Errorf("Expected %d hints, got %d", tt.expectedCount, len(hints))
			}

			if tt.expectHints {
				if len(hints) != len(tt.expectedKeys) {
					t.Errorf("Expected %d keys, got %d", len(tt.expectedKeys), len(hints))
				}

				for i, expectedKey := range tt.expectedKeys {
					if i < len(hints) && hints[i].Key != expectedKey {
						t.Errorf("Expected key %s, got %s", expectedKey, hints[i].Key)
					}
				}

				// Verify descriptions
				if len(hints) >= 1 && hints[0].Desc != "discard" {
					t.Errorf("Expected first hint description 'discard', got %s", hints[0].Desc)
				}
				if len(hints) >= 2 && hints[1].Desc != "squash" {
					t.Errorf("Expected second hint description 'squash', got %s", hints[1].Desc)
				}
			}
		})
	}
}

// TestGetActionHintsExperienceLog tests action hints for Log experience
func TestGetActionHintsExperienceLog(t *testing.T) {
	tests := []struct {
		name          string
		ctx           HelpBarContext
		expectedCount int
	}{
		{
			name: "Log panel in ExperienceLog",
			ctx: HelpBarContext{
				Experience:    ExperienceLog,
				FocusedPanel:  0,
				Entered:       false,
				IsWorkingCopy: false,
			},
			expectedCount: 1,
		},
		{
			name: "Workspace panel not entered",
			ctx: HelpBarContext{
				Experience:    ExperienceLog,
				FocusedPanel:  1,
				Entered:       false,
				IsWorkingCopy: false,
			},
			expectedCount: 0,
		},
		{
			name: "Workspace panel entered",
			ctx: HelpBarContext{
				Experience:    ExperienceLog,
				FocusedPanel:  1,
				Entered:       true,
				IsWorkingCopy: false,
			},
			expectedCount: 1,
		},
		{
			name: "Bookmarks panel not entered",
			ctx: HelpBarContext{
				Experience:    ExperienceLog,
				FocusedPanel:  2,
				Entered:       false,
				IsWorkingCopy: false,
			},
			expectedCount: 0,
		},
		{
			name: "Bookmarks panel entered",
			ctx: HelpBarContext{
				Experience:    ExperienceLog,
				FocusedPanel:  2,
				Entered:       true,
				IsWorkingCopy: false,
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := getActionHints(tt.ctx)
			if len(hints) != tt.expectedCount {
				t.Errorf("Expected %d hints, got %d", tt.expectedCount, len(hints))
			}
		})
	}
}

// TestGetNavigationHints tests navigation hints for different experiences
func TestGetNavigationHints(t *testing.T) {
	tests := []struct {
		name          string
		ctx           HelpBarContext
		expectedCount int
	}{
		{
			name: "Log panel navigation",
			ctx: HelpBarContext{
				Experience:    ExperienceLog,
				FocusedPanel:  0,
				Entered:       false,
				IsWorkingCopy: false,
			},
			expectedCount: 1,
		},
		{
			name: "Workspace not entered",
			ctx: HelpBarContext{
				Experience:    ExperienceLog,
				FocusedPanel:  1,
				Entered:       false,
				IsWorkingCopy: false,
			},
			expectedCount: 2,
		},
		{
			name: "Workspace entered",
			ctx: HelpBarContext{
				Experience:    ExperienceLog,
				FocusedPanel:  1,
				Entered:       true,
				IsWorkingCopy: false,
			},
			expectedCount: 1,
		},
		{
			name: "Diff panel in Change experience",
			ctx: HelpBarContext{
				Experience:    ExperienceChange,
				FocusedPanel:  0,
				Entered:       false,
				IsWorkingCopy: false,
			},
			expectedCount: 2,
		},
		{
			name: "Files panel in Change experience",
			ctx: HelpBarContext{
				Experience:    ExperienceChange,
				FocusedPanel:  1,
				Entered:       false,
				IsWorkingCopy: false,
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := getNavigationHints(tt.ctx)
			if len(hints) != tt.expectedCount {
				t.Errorf("Expected %d hints, got %d", tt.expectedCount, len(hints))
			}
		})
	}
}

// TestGetAlwaysHints tests the always-visible hints
func TestGetAlwaysHints(t *testing.T) {
	hints := getAlwaysHints()

	expectedCount := 3
	if len(hints) != expectedCount {
		t.Errorf("Expected %d always hints, got %d", expectedCount, len(hints))
	}

	expectedKeys := []string{"tab", "?", "q"}
	for i, expectedKey := range expectedKeys {
		if i < len(hints) && hints[i].Key != expectedKey {
			t.Errorf("Expected key %s, got %s", expectedKey, hints[i].Key)
		}
	}
}

// TestHelpHintFormat tests the Format method
func TestHelpHintFormat(t *testing.T) {
	hint := HelpHint{
		Key:  "del",
		Desc: "discard",
	}

	formatted := hint.Format()
	if formatted == "" {
		t.Error("Expected formatted hint to not be empty")
	}

	// The formatted string should contain the key and description
	// Note: It will include ANSI styling from theme, so we just verify it's not empty
	if len(formatted) < len("del discard") {
		t.Errorf("Formatted hint seems too short: %q", formatted)
	}
}

// TestFormatHints tests the formatHints function
func TestFormatHints(t *testing.T) {
	tests := []struct {
		name     string
		hints    []HelpHint
		expected string // We'll just check it's not empty or is empty as appropriate
	}{
		{
			name:     "Empty hints",
			hints:    []HelpHint{},
			expected: "",
		},
		{
			name: "Single hint",
			hints: []HelpHint{
				{Key: "del", Desc: "discard"},
			},
			expected: "non-empty",
		},
		{
			name: "Multiple hints",
			hints: []HelpHint{
				{Key: "del", Desc: "discard"},
				{Key: "s", Desc: "squash"},
			},
			expected: "non-empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatHints(tt.hints)
			if tt.expected == "" {
				if result != "" {
					t.Errorf("Expected empty result, got %q", result)
				}
			} else {
				if result == "" {
					t.Error("Expected non-empty result")
				}
			}
		})
	}
}

// TestRenderContextualHelpBar tests the main help bar rendering function
func TestRenderContextualHelpBar(t *testing.T) {
	tests := []struct {
		name   string
		ctx    HelpBarContext
		width  int
		expect bool
	}{
		{
			name: "Change experience with working copy",
			ctx: HelpBarContext{
				Experience:    ExperienceChange,
				FocusedPanel:  1,
				Entered:       false,
				IsWorkingCopy: true,
			},
			width:  80,
			expect: true,
		},
		{
			name: "Log experience",
			ctx: HelpBarContext{
				Experience:    ExperienceLog,
				FocusedPanel:  0,
				Entered:       false,
				IsWorkingCopy: false,
			},
			width:  80,
			expect: true,
		},
		{
			name: "Very narrow width",
			ctx: HelpBarContext{
				Experience:    ExperienceChange,
				FocusedPanel:  1,
				Entered:       false,
				IsWorkingCopy: true,
			},
			width:  20,
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderContextualHelpBar(tt.ctx, tt.width)
			if tt.expect && result == "" {
				t.Error("Expected non-empty help bar")
			}
		})
	}
}

// TestWorkingCopyDetection tests that IsWorkingCopy is correctly passed through
// Note: IsWorkingCopy is now determined by jj.ChangeInfo.IsWorkingCopy flag,
// NOT by checking if changeID == "@". The flag is set when parsing jj log output.
func TestWorkingCopyDetection(t *testing.T) {
	// The HelpBarContext.IsWorkingCopy field is now set directly from
	// App.selectedChangeIsWorking, which is populated from jj.ChangeInfo.IsWorkingCopy
	// during enterChangeExperience(). This test verifies the context field works correctly.
	tests := []struct {
		name          string
		isWorkingCopy bool
		expectHints   bool
	}{
		{
			name:          "Working copy should show file operation hints",
			isWorkingCopy: true,
			expectHints:   true,
		},
		{
			name:          "Non-working copy should not show file operation hints",
			isWorkingCopy: false,
			expectHints:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := HelpBarContext{
				Experience:    ExperienceChange,
				FocusedPanel:  1, // Files panel
				Entered:       false,
				IsWorkingCopy: tt.isWorkingCopy,
			}
			hints := getActionHints(ctx)
			hasHints := len(hints) > 0
			if hasHints != tt.expectHints {
				t.Errorf("Expected hints=%v, got hints=%v (count=%d)", tt.expectHints, hasHints, len(hints))
			}
		})
	}
}
