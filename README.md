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

### Help Bar

The help bar at the bottom of the screen has three sections that update based on context:

- **Actions (left)**: Context-dependent commands that modify state (e.g., edit, switch workspace). Only shown when actions are available for the current panel.
- **Navigation (center)**: Context-dependent movement commands (e.g., tab to cycle panels, arrows to select, enter to drill down). Changes based on current panel and mode.
- **Always (right)**: Global commands available everywhere (? help, q quit).
