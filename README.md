# JJazy = JJ + Lazy(git)

> Lazygit inspired TUI for Jujutsu.

## Prerequisites

- [Difftastic](https://difftastic.wilfred.me.uk/) - A structural diff tool that understands syntax

## Technical Details

### Panel Interaction Model

**Focus Modes:**
- **Focus mode**: Panel has yellow border. Arrow keys navigate between panels.
- **Cursor mode**: Inside a panel, navigating items with up/down. Yellow cursor visible on selected item.

**Panel Types:**
- **Browsable panel**: Requires Enter to enter cursor mode (e.g., Workspace, Bookmarks). Escape/Left exits back to focus mode.
- **Direct panel**: Always in cursor mode when focused (e.g., Log, Files, Diff). Cursor is immediately active.
