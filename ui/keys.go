package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application
// TODO: Make this configurable via config file
type KeyMap struct {
	// Global
	Quit       key.Binding
	Help       key.Binding
	ToggleLog  key.Binding
	Escape     key.Binding

	// Panel navigation
	Panel1     key.Binding
	Panel2     key.Binding
	Panel3     key.Binding
	Panel4     key.Binding
	NextPanel  key.Binding
	PrevPanel  key.Binding

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
		ToggleLog: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "log"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close/back"),
		),

		// Panel navigation
		Panel1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "status"),
		),
		Panel2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "files"),
		),
		Panel3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "bookmarks"),
		),
		Panel4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "operations"),
		),
		NextPanel: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		PrevPanel: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev panel"),
		),

		// List navigation
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "bottom"),
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
	}
}

// ShortHelp returns a short help string for the status bar
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.ToggleLog,
		k.Help,
		k.Quit,
	}
}

// FullHelp returns all keybindings for the help screen
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Panel1, k.Panel2, k.Panel3, k.Panel4},
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Enter, k.Space, k.Edit, k.Delete},
		{k.ToggleLog, k.Escape, k.Help, k.Quit},
	}
}
