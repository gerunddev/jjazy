package theme

import "github.com/charmbracelet/lipgloss"

// Monokai Pro color palette
var (
	ColorYellow     = lipgloss.Color("#FFD866")
	ColorOrange     = lipgloss.Color("#FC9867")
	ColorRed        = lipgloss.Color("#FF6188")
	ColorMagenta    = lipgloss.Color("#AB9DF2")
	ColorBlue       = lipgloss.Color("#78DCE8")
	ColorGreen      = lipgloss.Color("#A9DC76")
	ColorWhite      = lipgloss.Color("#FCFCFA")
	ColorDimWhite   = lipgloss.Color("#939293")
	ColorBackground = lipgloss.Color("#2D2A2E")
	ColorSurface    = lipgloss.Color("#403E41")
	ColorOverlay    = lipgloss.Color("#5B595C")
)

// Panel styles
var (
	// Focused panel border
	FocusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorYellow)

	// Unfocused panel border
	UnfocusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDimWhite)

	// Panel title style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorSurface).
			Padding(0, 1)

	// Focused title style
	FocusedTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorBackground).
				Background(ColorYellow).
				Padding(0, 1)
)

// List item styles
var (
	// Selected item in a list
	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(ColorYellow).
				Bold(true)

	// Normal item in a list
	NormalItemStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)

	// Dimmed/secondary text
	DimmedStyle = lipgloss.NewStyle().
			Foreground(ColorDimWhite)
)

// File status styles
var (
	ModifiedStyle = lipgloss.NewStyle().Foreground(ColorOrange)
	AddedStyle    = lipgloss.NewStyle().Foreground(ColorGreen)
	DeletedStyle  = lipgloss.NewStyle().Foreground(ColorRed)
	RenamedStyle  = lipgloss.NewStyle().Foreground(ColorBlue)
	ConflictStyle = lipgloss.NewStyle().Foreground(ColorMagenta).Bold(true)
)

// Diff styles
var (
	DiffAddLine     = lipgloss.NewStyle().Foreground(ColorGreen)
	DiffRemoveLine  = lipgloss.NewStyle().Foreground(ColorRed)
	DiffContextLine = lipgloss.NewStyle().Foreground(ColorDimWhite)
	DiffHunkHeader  = lipgloss.NewStyle().Foreground(ColorBlue).Bold(true)
)

// Log/revision styles
var (
	RevisionIDStyle  = lipgloss.NewStyle().Foreground(ColorOrange)
	ChangeIDStyle    = lipgloss.NewStyle().Foreground(ColorMagenta)
	AuthorStyle      = lipgloss.NewStyle().Foreground(ColorYellow)
	TimestampStyle   = lipgloss.NewStyle().Foreground(ColorBlue)
	WorkingCopyStyle = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)

	// Unique prefix highlighting styles (for jj-style ID display)
	// Prefix: colored, Rest: grey/dimmed
	ChangeIDPrefixStyle   = lipgloss.NewStyle().Foreground(ColorMagenta)
	ChangeIDRestStyle     = lipgloss.NewStyle().Foreground(ColorDimWhite)
	RevisionIDPrefixStyle = lipgloss.NewStyle().Foreground(ColorOrange)
	RevisionIDRestStyle   = lipgloss.NewStyle().Foreground(ColorDimWhite)
)

// Floating window styles
var (
	FloatingWindowStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorYellow).
				Background(ColorBackground)

	FloatingTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorBackground).
				Background(ColorYellow).
				Padding(0, 1)
)

// Help bar style
var (
	HelpBarStyle = lipgloss.NewStyle().
			Foreground(ColorDimWhite)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorDimWhite)
)

// Layout constants
const (
	SidebarWidth      = 30
	SidebarMinWidth   = 20
	SidebarMaxWidth   = 50
	PanelMinHeight    = 5
	FloatingLogWidth  = 80
	FloatingLogHeight = 20
)
