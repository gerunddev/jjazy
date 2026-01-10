package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application
// TODO: Make this configurable via config file
type KeyMap struct {
	// Global
	Quit   key.Binding
	Help   key.Binding
	Escape key.Binding

	// Panel navigation
	Panel0    key.Binding
	Panel1    key.Binding
	Panel2    key.Binding
	Panel3    key.Binding
	NextPanel key.Binding
	PrevPanel key.Binding

	// List navigation (within panels)
	Up         key.Binding
	Down       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Home       key.Binding
	End        key.Binding

	// Actions
	Enter      key.Binding
	Space      key.Binding
	Delete     key.Binding
	Edit       key.Binding

	// Log view change actions
	NewChange  key.Binding
	Describe   key.Binding
	Abandon    key.Binding
	SquashChange key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Global
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close/back"),
		),

		// Panel navigation
		Panel0: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("0", "log"),
		),
		Panel1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "workspace"),
		),
		Panel2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "files"),
		),
		Panel3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "bookmarks"),
		),
		NextPanel: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		PrevPanel: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev panel"),
		),

		// List navigation (emacs-style)
		Up: key.NewBinding(
			key.WithKeys("up", "ctrl+p"),
			key.WithHelp("↑/C-p", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "ctrl+n"),
			key.WithHelp("↓/C-n", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "alt+v"),
			key.WithHelp("M-v", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+v"),
			key.WithHelp("C-v", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "alt+<"),
			key.WithHelp("M-<", "top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "alt+>"),
			key.WithHelp("M->", "bottom"),
		),

		// Actions
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d", "delete"),
			key.WithHelp("d", "delete"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),

		// Log view change actions
		NewChange: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new"),
		),
		Describe: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "describe"),
		),
		Abandon: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "abandon"),
		),
		SquashChange: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "squash"),
		),
	}
}

// ShortHelp returns a short help string for the status bar
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help,
		k.Quit,
	}
}

// FullHelp returns all keybindings for the help screen
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Panel0, k.Panel1, k.Panel2, k.Panel3},
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Enter, k.Space, k.Edit, k.Delete},
		{k.Escape, k.Help, k.Quit},
	}
}
