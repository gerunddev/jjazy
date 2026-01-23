# Feature: Range Squash

Squash a range of commits with an intuitive two-selection UX.

## Requirements

### User Stories
- As a user, I want to select a destination commit and an endpoint commit, and squash all commits between them into the destination
- As a user, I want the selection order to not matter (system infers ancestor/descendant)
- As a user, I want to provide a description for the squashed result
- As a user, I want this available in both TUI and interactive (-i) modes

### Acceptance Criteria
- [ ] In TUI log panel: press 's' to start squash mode, navigate and select second commit, press enter
- [ ] Press ESC or left arrow to cancel squash mode
- [ ] Visual feedback during squash mode (title change, help bar update)
- [ ] Floating dialog prompts for new commit description (pre-populated with destination's current description)
- [ ] Works regardless of selection order (a then f, or f then a)
- [ ] Works with non-adjacent commits (squashes entire range)
- [ ] Interactive mode (-i) offers "Squash Range" action
- [ ] Error handling for invalid selections (same commit, divergent branches, root commit)

### jj Command Equivalent

User selects commits `a` and `f`. System determines `a` is ancestor, `f` is descendant:

```bash
jj squash --from "a::f ~ a" --into a -m "description"
```

Where:
- `a::f` = all commits from a to f inclusive
- `~ a` = excluding a itself (so the range becomes b,c,d,e,f)
- `--into a` = destination is a
- Result: changes from b,c,d,e,f are moved into a, those commits are abandoned

### Visual Example

```
Before:                          After:
  f  ← user selects                (abandoned)
  |                                (abandoned)
  e                                (abandoned)
  |                                (abandoned)
  d                       →        (abandoned)
  |                                (abandoned)
  c                                (abandoned)
  |                                (abandoned)
  b                                (abandoned)
  |                                   |
  a  ← user selects               a' (contains all changes from a+b+c+d+e+f)
  |                                   |
  ...                                ...
```

---

## Architecture

### UX Flow (TUI)

```
Log Panel (normal mode)
    |
    +-- User presses 's' on commit A
    |        |
    |        +-- Enter "Squash Mode"
    |        |   - Store first selection (A)
    |        |   - Title changes: "0 Squash"
    |        |   - Help bar shows: "↵ confirm | esc cancel"
    |        |
    |        +-- User navigates to commit F, presses Enter
    |        |       |
    |        |       +-- Validate: A ≠ F, A and F are on same lineage
    |        |       +-- Compute: lower=ancestor, higher=descendant
    |        |       +-- Show TextInputOverlay "Squash Description"
    |        |       |       (pre-filled with lower's current description)
    |        |       |
    |        |       +-- User enters description, Ctrl+S
    |        |       |       |
    |        |       |       +-- Execute: jj squash --from "lower::higher ~ lower" --into lower -m "desc"
    |        |       |       +-- Exit squash mode
    |        |       |       +-- Refresh all panels
    |        |       |
    |        |       +-- User presses Esc/Ctrl+X at description prompt
    |        |               |
    |        |               +-- Cancel, exit squash mode
    |        |
    |        +-- User presses 's' or Enter on SAME commit (A)
    |                |
    |                +-- Execute simple squash-to-parent (existing behavior)
    |                +-- jj squash -r A
    |
    +-- User presses Esc/Left during squash mode
            |
            +-- Exit squash mode, return to normal
```

### UX Flow (Interactive Mode)

```
Main Menu
    |
    +-- "Squash Range" option
            |
            +-- "Select destination commit" (huh select)
            |       - This is where changes will be squashed INTO
            |
            +-- "Select end of range" (huh select)
            |       - Everything between destination and this commit will be squashed
            |
            +-- Validate selection (same lineage, not same commit)
            |
            +-- "Enter description" (huh input, pre-filled)
            |
            +-- Execute squash
            |
            +-- Display result
```

### State Changes in App

```go
// New fields in ui/app.go App struct
type App struct {
    // ... existing fields ...

    // Squash mode state (follows bookmarkSetMode pattern)
    squashMode           bool   // True when in squash selection flow
    squashFirstChangeID  string // Change ID of first selection
    squashFirstCursor    int    // Preserved cursor position for cancel
}
```

### Component Changes

| File | Changes |
|------|---------|
| `app/navigation.go` | Add `IsAncestor()`, `OrderByAncestry()`, `GetDescription()` |
| `jj/cli.go` | Add `SquashRange()` function |
| `ui/app.go` | Add squash mode state, handlers, flow logic |
| `ui/helpbar.go` | Add squash mode hints |
| `interactive/interactive.go` | Add "Squash Range" menu option |
| `interactive/actions.go` | Add `runSquashRange()` implementation |

---

## Detailed Implementation

### 1. Navigation Utilities (`app/navigation.go`)

```go
import (
    "fmt"
    "github.com/gerunddev/jjazy/jj"
)

// IsAncestor returns true if potentialAncestor is an ancestor of potentialDescendant.
func (n *Navigation) IsAncestor(ancestorID, descendantID string) bool {
    if ancestorID == descendantID {
        return false
    }

    // BFS from descendant walking up through parents
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

        for _, parentID := range rev.Parents {
            if parentID == ancestorID {
                return true
            }
            queue = append(queue, parentID)
        }
    }
    return false
}

// OrderByAncestry returns (ancestor, descendant) for two commits.
// Error if same commit or divergent branches.
func (n *Navigation) OrderByAncestry(idA, idB string) (lower, higher string, err error) {
    if idA == idB {
        return "", "", fmt.Errorf("same commit selected twice")
    }

    if n.IsAncestor(idA, idB) {
        return idA, idB, nil
    }
    if n.IsAncestor(idB, idA) {
        return idB, idA, nil
    }

    return "", "", fmt.Errorf("commits are on divergent branches")
}

// FindByChangeID returns revision by change ID (short or full).
func (n *Navigation) FindByChangeID(changeID string) *jj.Revision {
    for i := range n.revisions {
        if n.revisions[i].ChangeID == changeID ||
           strings.HasPrefix(n.revisions[i].ChangeID, changeID) ||
           n.revisions[i].ID == changeID {
            return &n.revisions[i]
        }
    }
    return nil
}
```

### 2. CLI Squash Function (`jj/cli.go`)

```go
// SquashRange squashes a range of commits into a destination.
// fromRevset: the revset expression for source commits (e.g., "a::f ~ a")
// intoRev: the destination commit
// message: description for the resulting commit (empty to keep destination's)
func SquashRange(repoPath, fromRevset, intoRev, message string) error {
    args := []string{"squash", "--from", fromRevset, "--into", intoRev}
    if message != "" {
        args = append(args, "-m", message)
    }
    cmd := exec.Command("jj", args...)
    cmd.Dir = repoPath
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("squash failed: %s", string(output))
    }
    return nil
}

// BuildSquashRevset creates the revset for range squash.
// Returns "lower::higher ~ lower" which includes everything from lower to higher, excluding lower.
func BuildSquashRevset(lowerChangeID, higherChangeID string) string {
    return fmt.Sprintf("%s::%s ~ %s", lowerChangeID, higherChangeID, lowerChangeID)
}
```

### 3. TUI App Changes (`ui/app.go`)

#### New State Fields

```go
type App struct {
    // ... existing fields ...

    // Squash mode state
    squashMode          bool
    squashFirstChangeID string
    squashFirstCursor   int
}
```

#### Enter Squash Mode (on 's' key in log panel)

```go
// In the key handling section for ExperienceLog, focusedPanel == 0:
case key.Matches(msg, a.keys.SquashChange):
    if change := a.logPanel.SelectedChange(); change != nil {
        if a.squashMode {
            // Already in squash mode - this is the second selection
            a.handleSquashSecondSelection(change)
        } else {
            // Start squash mode
            a.enterSquashMode(change.ChangeID)
        }
    }
    return a, nil
```

#### Squash Mode Functions

```go
func (a *App) enterSquashMode(changeID string) {
    a.squashMode = true
    a.squashFirstChangeID = changeID
    a.squashFirstCursor = a.logPanel.selectedIndex // preserve for cancel

    // Update log panel title
    a.logPanel.SetTitle("0 Squash")
}

func (a *App) exitSquashMode() {
    a.squashMode = false
    a.squashFirstChangeID = ""
    a.logPanel.SetTitle("0 Log")
}

func (a *App) handleSquashSecondSelection(secondChange *jj.ChangeInfo) {
    firstChangeID := a.squashFirstChangeID
    secondChangeID := secondChange.ChangeID

    // Same commit = simple squash to parent (existing behavior)
    if firstChangeID == secondChangeID {
        _ = jj.Squash(a.repoPath, firstChangeID)
        a.exitSquashMode()
        a.refreshAllPanels()
        return
    }

    // Get revisions for ancestry check
    revisions, err := a.repo.Log()
    if err != nil {
        a.showInfoDialog("Error", "Failed to get log: "+err.Error())
        a.exitSquashMode()
        return
    }

    nav := app.NewNavigation(a.repoPath, revisions)

    // Determine order
    lower, higher, err := nav.OrderByAncestry(firstChangeID, secondChangeID)
    if err != nil {
        a.showInfoDialog("Error", err.Error())
        a.exitSquashMode()
        return
    }

    // Get current description of destination (lower) for pre-fill
    currentDesc, _ := jj.GetDescription(a.repoPath, lower)

    // Store squash params for after description input
    a.squashLower = lower
    a.squashHigher = higher

    // Show description input
    a.textInputOverlay = floating.NewTextInputOverlay(
        "Squash Description",
        "Enter description for squashed commit...",
        currentDesc,
    )
    a.textInputOverlay.SetSize(a.width, a.height-1)
    a.showTextInput = true
    a.textInputAction = "squash_range"
}

func (a *App) executeSquashRange(description string) {
    revset := jj.BuildSquashRevset(a.squashLower, a.squashHigher)

    err := jj.SquashRange(a.repoPath, revset, a.squashLower, description)
    if err != nil {
        a.showInfoDialog("Error", err.Error())
    }

    a.exitSquashMode()
    a.refreshAllPanels()
}

func (a *App) refreshAllPanels() {
    a.logPanel.Refresh()
    a.workspacePanel.Refresh()
    a.bookmarksPanel.Refresh()
}
```

#### Handle Enter Key in Squash Mode

```go
// In Enter key handling for log panel:
case key.Matches(msg, a.keys.Enter):
    if a.currentExperience == ExperienceLog && a.focusedPanel == 0 {
        if a.squashMode {
            // Confirm second selection
            if change := a.logPanel.SelectedChange(); change != nil {
                a.handleSquashSecondSelection(change)
            }
            return a, nil
        }
        // ... existing edit logic ...
    }
```

#### Handle Escape in Squash Mode

```go
// In escape handling:
case key.Matches(msg, a.keys.Escape):
    if a.squashMode {
        a.exitSquashMode()
        return a, nil
    }
    // ... existing escape logic ...
```

#### Handle Text Input Completion

```go
// In text input Ctrl+S handling (existing section):
case "squash_range":
    a.executeSquashRange(value)
```

### 4. Help Bar Updates (`ui/helpbar.go`)

```go
func getActionHints(ctx HelpBarContext) []HelpHint {
    switch ctx.Experience {
    case ExperienceLog:
        switch ctx.FocusedPanel {
        case 0: // Log panel
            if ctx.SquashMode {
                return []HelpHint{
                    {Key: "↵", Desc: "squash"},
                    {Key: "s", Desc: "squash"},
                }
            }
            if ctx.BookmarkSetMode {
                // ... existing ...
            }
            return []HelpHint{
                {Key: "↵", Desc: "edit"},
                {Key: "n", Desc: "new"},
                {Key: "d", Desc: "describe"},
                {Key: "a", Desc: "abandon"},
                {Key: "s", Desc: "squash"},
            }
        // ...
        }
    }
}

func getNavigationHints(ctx HelpBarContext) []HelpHint {
    switch ctx.Experience {
    case ExperienceLog:
        switch ctx.FocusedPanel {
        case 0: // Log panel
            if ctx.SquashMode {
                return []HelpHint{
                    {Key: "←", Desc: "cancel"},
                    {Key: "↑↓", Desc: "select"},
                }
            }
            // ... existing ...
        }
    }
}
```

Add to `HelpBarContext`:
```go
type HelpBarContext struct {
    // ... existing ...
    SquashMode bool
}
```

### 5. Interactive Mode (`interactive/interactive.go`)

```go
func Run(repoPath string) error {
    var action string

    err := huh.NewSelect[string]().
        Title("jjazy - Quick Actions").
        Options(
            huh.NewOption("Edit - Switch working copy to revision", "edit"),
            huh.NewOption("Rebase - Move revision to new parent", "rebase"),
            huh.NewOption("Squash Range - Combine commits into destination", "squash_range"),
        ).
        Value(&action).
        Run()

    if err != nil {
        return err
    }

    switch action {
    case "edit":
        return runEdit(repoPath)
    case "rebase":
        return runRebase(repoPath)
    case "squash_range":
        return runSquashRange(repoPath)
    }

    return nil
}
```

### 6. Interactive Squash Range (`interactive/actions.go`)

```go
func runSquashRange(repoPath string) error {
    log, err := jj.LogCLI(repoPath)
    if err != nil {
        return fmt.Errorf("failed to get log: %w", err)
    }

    options := buildRevisionOptions(log.Changes)
    if len(options) < 2 {
        fmt.Println("Need at least 2 revisions to squash")
        return nil
    }

    // Select destination (where changes will be squashed INTO)
    var destChangeID string
    err = huh.NewSelect[string]().
        Title("Select destination commit").
        Description("Changes will be squashed INTO this commit").
        Options(options...).
        Value(&destChangeID).
        Run()

    if err != nil {
        return nil // Cancelled
    }

    // Select end of range
    var endChangeID string
    err = huh.NewSelect[string]().
        Title("Select end of range").
        Description(fmt.Sprintf("Everything from %s to this commit will be squashed", destChangeID[:8])).
        Options(options...).
        Value(&endChangeID).
        Run()

    if err != nil {
        return nil // Cancelled
    }

    if destChangeID == endChangeID {
        fmt.Println("Selected same commit - use simple squash instead")
        return nil
    }

    // Build navigation to determine order
    revisions := logToRevisions(log.Changes)
    nav := app.NewNavigation(repoPath, revisions)

    lower, higher, err := nav.OrderByAncestry(destChangeID, endChangeID)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return nil
    }

    // Get current description for pre-fill
    currentDesc, _ := jj.GetDescription(repoPath, lower)

    // Get new description
    var description string
    err = huh.NewInput().
        Title("Enter description for squashed commit").
        Value(&description).
        Placeholder(currentDesc).
        Run()

    if err != nil {
        return nil // Cancelled
    }

    if description == "" {
        description = currentDesc
    }

    // Execute squash
    revset := jj.BuildSquashRevset(lower, higher)
    if err := jj.SquashRange(repoPath, revset, lower, description); err != nil {
        return fmt.Errorf("squash failed: %w", err)
    }

    fmt.Printf("Squashed %s to %s into %s\n", lower[:8], higher[:8], lower[:8])
    return nil
}

// Helper to convert CLI ChangeInfo to jj.Revision for Navigation
func logToRevisions(changes []jj.ChangeInfo) []jj.Revision {
    var revs []jj.Revision
    for _, c := range changes {
        revs = append(revs, jj.Revision{
            ID:       c.CommitID,
            ChangeID: c.ChangeID,
            // Parents would need to be fetched separately or we enhance LogCLI
        })
    }
    return revs
}
```

**Note**: The interactive mode will need parent information to use Navigation. Options:
1. Enhance `jj.LogCLI()` to also fetch parent info
2. Use a simpler CLI-based ancestor check
3. Just use `jj log -r "A & ::B"` to check if A is ancestor of B

Simpler approach for interactive mode:
```go
func isAncestor(repoPath, potentialAncestor, potentialDescendant string) bool {
    // Use jj log to check if ancestor is in the ancestry of descendant
    cmd := exec.Command("jj", "log", "-r",
        fmt.Sprintf("%s & ::%s", potentialAncestor, potentialDescendant),
        "--no-graph", "-T", "change_id")
    cmd.Dir = repoPath
    output, err := cmd.Output()
    if err != nil {
        return false
    }
    return strings.TrimSpace(string(output)) != ""
}
```

---

## Testing

### Unit Tests

**`app/navigation_test.go`**:
- `TestIsAncestor` - linear chain, branching
- `TestOrderByAncestry` - normal case, reversed input, same commit error, divergent branches

**`jj/cli_test.go`**:
- `TestBuildSquashRevset` - verify correct revset format

### Manual Testing Checklist

- [ ] TUI: Press 's', navigate to different commit, press Enter - squash works
- [ ] TUI: Press 's', press 's' again on same commit - simple squash to parent
- [ ] TUI: Press 's', press Esc - cancels cleanly
- [ ] TUI: Press 's', press Left arrow - cancels cleanly
- [ ] TUI: Select commits in reverse order (descendant first) - still works
- [ ] TUI: Select commits on divergent branches - shows error
- [ ] TUI: Description pre-fills with destination's current description
- [ ] TUI: Empty description keeps original
- [ ] Interactive: Full flow works
- [ ] Interactive: Cancel at any step works
- [ ] Verify descendants of squashed commits are rebased correctly
- [ ] Verify abandoned commits are actually gone

---

## Security & Safety

### Risks
- **Data modification**: Squash rewrites history
- **Immutable commits**: User might try to squash pushed commits

### Mitigations
- jj's built-in immutability checking will reject operations on immutable commits
- jj's undo system (`jj undo`) provides recovery
- Clear UX feedback before execution (description prompt acts as confirmation)

---

## Future Enhancements (Out of Scope)

- Interactive hunk selection (`jj squash -i`)
- Squash preview showing combined diff
- Multi-select for non-contiguous commits
- jj-lib native FFI implementation (currently uses CLI)
- Partial squash (select specific files)
- Textarea for multi-line descriptions

---

## Implementation Order

1. **Navigation utilities** - `app/navigation.go` additions (independent)
2. **CLI function** - `jj/cli.go` additions (independent)
3. **TUI state & basic flow** - `ui/app.go` squash mode entry/exit
4. **TUI selection handling** - ancestry check, description prompt
5. **TUI execution** - wire up squash execution
6. **Help bar** - squash mode hints
7. **Interactive mode** - add to menu and implement flow
8. **Tests** - unit tests for navigation, manual testing

Steps 1-2 are independent and can be done in parallel.
Steps 3-6 are sequential (TUI implementation).
Step 7 depends on step 2.
Step 8 runs throughout.

---

## References

- [jj-squash man page](https://man.archlinux.org/man/extra/jujutsu/jj-squash.1.en)
- [The Squash Workflow Tutorial](https://steveklabnik.github.io/jujutsu-tutorial/real-world-workflows/the-squash-workflow.html)
- [jj tip: Squash Changes From Across a Revset](https://v5.chriskrycho.com/notes/jj-tip-squash-changes-from-across-a-revset/)
- Codebase patterns: `ui/app.go` (bookmark set mode), `jj/cli.go` (CLI wrappers)
