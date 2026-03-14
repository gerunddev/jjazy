## JJ-lib
Interactions with core jj functionality need to go through jj-lib (wrapped rust). The ONLY exception to this is retreiving the log.

## ID Types in jjazy

There are three distinct ID types used throughout the codebase. Understanding which to use for each context is critical:

### ID Types

| Type | Source | Usage | Example |
|------|--------|-------|---------|
| `jj.Revision.ID` | `repo.Log()` | **jj-lib FFI calls** (SetBookmark, etc.) - full revision hash | `abc1234def5678...` (complete hash) |
| `jj.ChangeInfo.ChangeID` | `logPanel.SelectedChange()` | UI display, Navigation lookups, bookmarks list | `abc1234d` or full change ID |
| `jj.ChangeInfo.CommitID` | `logPanel.SelectedChange()` | Git commit hash (informational only) | `e8b8bbe8` (short form) |

### When to Use Each

- **For jj-lib FFI calls** (e.g., `repo.SetBookmark()`, likely `repo.Abandon()`, etc.): Use the **full `revision.ID`**
  - Get revisions from `repo.Log()`, then use `FindByChangeID()` to locate the target revision
  - Example: `a.executeBookmarkSet(targetRev.ID)` ✓ (correct)

- **For UI display and Navigation queries**: Use `ChangeID`
  - Example: `nav.FindByChangeID(change.ChangeID)` ✓ (correct)

- **For CommitID**: Avoid using this for operations; it's the git commit hash and jj-lib doesn't understand it
  - Previous incorrect attempt: `a.executeBookmarkSet(change.CommitID)` ✗ (error: "Revision not found")

### Pattern: Mapping ChangeID to Full Revision

When you have a ChangeInfo from the log and need to call a jj-lib operation:
```go
revisions, _ := a.repo.Log()
nav := app.NewNavigation(a.repoPath, revisions)
targetRev, found := nav.FindByChangeID(change.ChangeID)
if found && targetRev != nil {
    a.repo.SetBookmark(name, targetRev.ID, false, false)  // Use full ID for FFI
}
```
