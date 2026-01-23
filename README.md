# JJazy = JJ + Lazy(git)

> Lazygit inspired TUI for Jujutsu.

## Installation

### Homebrew (macOS)

```bash
brew install gerunddev/tap/jjazy
```

### Install Script (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/gerunddev/jjazy/main/install.sh | bash
```

To install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/gerunddev/jjazy/main/install.sh | bash -s v0.1.0
```

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/gerunddev/jjazy/releases).

### Build from Source

Requires Go 1.23+ and Rust 1.70+:

```bash
git clone https://github.com/gerunddev/jjazy.git
cd jjazy
make install
```

> **Note:** `go install github.com/gerunddev/jjazy@latest` does not work because this project includes a Rust FFI component that must be compiled locally.

## Prerequisites

- [Difftastic](https://difftastic.wilfred.me.uk/) - A structural diff tool that understands syntax

## Technical Details

### Panel Interaction Model

**Experiences**
- **Log Experience**: Initial view that focuses on the log and navigating it with workspaces and bookmarks.
- **Change Experience**: Drill down into a change to see the files and diffs.

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
