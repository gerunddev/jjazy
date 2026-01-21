# Feature: Interactive Mode (-i)

A lightweight interactive mode using [Huh](https://github.com/charmbracelet/huh) components for quick jj actions without the full TUI experience.

## Requirements

### User Stories
- As a user, I want to run `jjazy -i` to get a quick action menu
- As a user, I want to select a revision and perform Edit or Rebase quickly
- As a user, I want a simpler alternative to the full TUI for quick operations

### Acceptance Criteria
- [ ] `-i` flag launches interactive mode instead of full TUI
- [ ] Action menu presents: Edit, Rebase
- [ ] Edit: Select revision from log, execute `jj edit`
- [ ] Rebase: Select source revision, select destination, execute `jj rebase`
- [ ] Clean exit on cancel (Esc/Ctrl+C)
- [ ] Errors displayed clearly to user

## Architecture

### Component Structure

```
main.go                    # Flag parsing, mode dispatch
interactive/
  interactive.go           # Main interactive mode entry point
  actions.go               # Action implementations (edit, rebase)
jj/
  cli.go                   # Add Rebase function
```

### Flow Diagram

```
main.go
  |
  +-- flag "-i" present?
       |
       +-- No  --> Launch full TUI (existing)
       |
       +-- Yes --> interactive.Run(repo, repoPath)
                     |
                     +-- Show Action Select (Huh)
                          |
                          +-- "Edit"   --> showEditFlow()
                          |                 +-- Select revision (Huh)
                          |                 +-- jj.Edit()
                          |
                          +-- "Rebase" --> showRebaseFlow()
                                           +-- Select source (Huh)
                                           +-- Select destination (Huh)
                                           +-- jj.Rebase()
```

### Huh Usage Patterns

```go
import "github.com/charmbracelet/huh"

// Action selection
var action string
huh.NewSelect[string]().
    Title("Select action").
    Options(
        huh.NewOption("Edit - Switch working copy", "edit"),
        huh.NewOption("Rebase - Move revision", "rebase"),
    ).
    Value(&action).
    Run()

// Revision selection (build options from log)
var revision string
huh.NewSelect[string]().
    Title("Select revision to edit").
    Options(revisionOptions...).  // Built from jj.LogCLI()
    Value(&revision).
    Run()
```

### Integration with Existing Code

- Reuse `jj.LogCLI()` to get revision list for selection
- Reuse `jj.Edit()` for edit action
- Add new `jj.Rebase()` function for rebase action
- No jj-lib changes needed (CLI operations only)

## Testability

### Unit Tests
- `jj.Rebase()` - test command construction, error handling
- Revision option builder - test formatting

### Manual Testing
- Run `jjazy -i` and verify action menu appears
- Test Edit flow with various revision selections
- Test Rebase flow with source/destination selection
- Test cancel behavior at each step
- Test error cases (invalid revision, rebase conflicts)

## Deployability

### Dependencies
- Add `github.com/charmbracelet/huh` to go.mod (v2 recommended)
- Compatible with existing charmbracelet dependencies

### No Breaking Changes
- `-i` flag is additive
- Default behavior (no flags) unchanged
- Full TUI remains primary interface

## Security

### Risks
- None significant - uses existing jj CLI operations
- No new file system access beyond current scope

### Mitigations
- All operations go through jj CLI (same as full TUI)
- User must explicitly select revisions

## Tasks

### Task 1: Add Rebase CLI Function
**File:** `jj/cli.go`

Add `Rebase` function to execute `jj rebase`:

```go
// Rebase moves a revision to a new parent
// jj rebase -r <source> -d <destination>
func Rebase(repoPath, sourceRev, destRev string) error {
    cmd := exec.Command("jj", "rebase", "-r", sourceRev, "-d", destRev)
    cmd.Dir = repoPath
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("rebase failed: %s", string(output))
    }
    return nil
}
```

Also add branch rebase variant:
```go
// RebaseBranch rebases a revision and its descendants
// jj rebase -b <branch> -d <destination>
func RebaseBranch(repoPath, branchRev, destRev string) error {
    cmd := exec.Command("jj", "rebase", "-b", branchRev, "-d", destRev)
    cmd.Dir = repoPath
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("rebase failed: %s", string(output))
    }
    return nil
}
```

---

### Task 2: Create Interactive Package
**File:** `interactive/interactive.go`

```go
package interactive

import (
    "github.com/charmbracelet/huh"
    "github.com/gerund/jjazy/jj"
)

// Run starts the interactive mode
func Run(repo *jj.Repo, repoPath string) error {
    var action string

    err := huh.NewSelect[string]().
        Title("jjazy - Quick Actions").
        Options(
            huh.NewOption("Edit - Switch working copy to revision", "edit"),
            huh.NewOption("Rebase - Move revision to new parent", "rebase"),
        ).
        Value(&action).
        Run()

    if err != nil {
        return err // User cancelled
    }

    switch action {
    case "edit":
        return runEdit(repoPath)
    case "rebase":
        return runRebase(repoPath)
    }

    return nil
}
```

---

### Task 3: Implement Edit Action
**File:** `interactive/actions.go`

```go
package interactive

import (
    "fmt"
    "github.com/charmbracelet/huh"
    "github.com/gerund/jjazy/jj"
)

func runEdit(repoPath string) error {
    // Get log for revision selection
    log, err := jj.LogCLI(repoPath)
    if err != nil {
        return fmt.Errorf("failed to get log: %w", err)
    }

    // Build options from changes
    options := buildRevisionOptions(log.Changes)
    if len(options) == 0 {
        fmt.Println("No revisions available")
        return nil
    }

    var revision string
    err = huh.NewSelect[string]().
        Title("Select revision to edit").
        Options(options...).
        Value(&revision).
        Run()

    if err != nil {
        return nil // User cancelled
    }

    // Execute edit
    if err := jj.Edit(repoPath, revision); err != nil {
        return fmt.Errorf("edit failed: %w", err)
    }

    fmt.Printf("Now editing %s\n", revision)
    return nil
}

func buildRevisionOptions(changes []jj.ChangeInfo) []huh.Option[string] {
    var options []huh.Option[string]
    for _, c := range changes {
        label := c.ChangeID
        if c.IsWorkingCopy {
            label += " (@)"
        }
        options = append(options, huh.NewOption(label, c.ChangeID))
    }
    return options
}
```

---

### Task 4: Implement Rebase Action
**File:** `interactive/actions.go` (add to existing)

```go
func runRebase(repoPath string) error {
    log, err := jj.LogCLI(repoPath)
    if err != nil {
        return fmt.Errorf("failed to get log: %w", err)
    }

    options := buildRevisionOptions(log.Changes)
    if len(options) < 2 {
        fmt.Println("Need at least 2 revisions to rebase")
        return nil
    }

    // Select source revision
    var source string
    err = huh.NewSelect[string]().
        Title("Select revision to rebase (source)").
        Options(options...).
        Value(&source).
        Run()

    if err != nil {
        return nil // Cancelled
    }

    // Select destination revision
    var dest string
    err = huh.NewSelect[string]().
        Title("Select destination (new parent)").
        Description(fmt.Sprintf("Rebasing %s onto...", source)).
        Options(options...).
        Value(&dest).
        Run()

    if err != nil {
        return nil // Cancelled
    }

    if source == dest {
        fmt.Println("Source and destination cannot be the same")
        return nil
    }

    // Execute rebase
    if err := jj.Rebase(repoPath, source, dest); err != nil {
        return fmt.Errorf("rebase failed: %w", err)
    }

    fmt.Printf("Rebased %s onto %s\n", source, dest)
    return nil
}
```

---

### Task 5: Update main.go with Flag Handling
**File:** `main.go`

```go
package main

import (
    "flag"
    "fmt"
    "os"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/gerund/jjazy/interactive"
    "github.com/gerund/jjazy/jj"
    "github.com/gerund/jjazy/ui"
)

func main() {
    // Parse flags
    interactiveMode := flag.Bool("i", false, "Run in interactive mode (quick actions)")
    flag.Parse()

    // Open the repository
    repo, err := jj.Open(".")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error opening repository: %v\n", err)
        os.Exit(1)
    }
    defer repo.Close()

    // Dispatch based on mode
    if *interactiveMode {
        if err := interactive.Run(repo, "."); err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
        return
    }

    // Full TUI mode (default)
    app := ui.NewApp(repo, ".")
    p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

---

### Task 6: Add Huh Dependency
**Command:** `go get github.com/charmbracelet/huh@latest`

Update `go.mod` to include:
```
github.com/charmbracelet/huh v0.x.x
```

---

## Implementation Order

1. **Task 6** - Add Huh dependency (quick, unblocks everything)
2. **Task 1** - Add Rebase function to jj/cli.go (independent)
3. **Task 2** - Create interactive package skeleton
4. **Task 3** - Implement Edit action (uses existing jj.Edit)
5. **Task 4** - Implement Rebase action (uses new jj.Rebase)
6. **Task 5** - Wire up main.go flag handling

Tasks 1-2 can be done in parallel.
Tasks 3-4 can be done in parallel after Task 2.

## Future Enhancements (Out of Scope)

- Additional actions: Describe, Abandon, Squash, New
- Revision filtering/search in selection
- Multi-revision rebase selection
- Theming to match full TUI
- `-i <action>` to skip action menu (e.g., `jjazy -i edit`)

## References

- [Huh Documentation](https://pkg.go.dev/github.com/charmbracelet/huh)
- [Huh GitHub](https://github.com/charmbracelet/huh)
- Existing jj CLI wrapper: `jj/cli.go`
