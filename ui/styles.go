package ui

import "github.com/gerund/jayz/ui/theme"

// Re-export theme constants for backward compatibility
var (
	ColorYellow     = theme.ColorYellow
	ColorOrange     = theme.ColorOrange
	ColorRed        = theme.ColorRed
	ColorMagenta    = theme.ColorMagenta
	ColorBlue       = theme.ColorBlue
	ColorGreen      = theme.ColorGreen
	ColorWhite      = theme.ColorWhite
	ColorDimWhite   = theme.ColorDimWhite
	ColorBackground = theme.ColorBackground
	ColorSurface    = theme.ColorSurface
	ColorOverlay    = theme.ColorOverlay

	FocusedBorder     = theme.FocusedBorder
	UnfocusedBorder   = theme.UnfocusedBorder
	TitleStyle        = theme.TitleStyle
	FocusedTitleStyle = theme.FocusedTitleStyle

	SelectedItemStyle = theme.SelectedItemStyle
	NormalItemStyle   = theme.NormalItemStyle
	DimmedStyle       = theme.DimmedStyle

	ModifiedStyle = theme.ModifiedStyle
	AddedStyle    = theme.AddedStyle
	DeletedStyle  = theme.DeletedStyle
	RenamedStyle  = theme.RenamedStyle
	ConflictStyle = theme.ConflictStyle

	DiffAddLine     = theme.DiffAddLine
	DiffRemoveLine  = theme.DiffRemoveLine
	DiffContextLine = theme.DiffContextLine
	DiffHunkHeader  = theme.DiffHunkHeader

	RevisionIDStyle  = theme.RevisionIDStyle
	ChangeIDStyle    = theme.ChangeIDStyle
	AuthorStyle      = theme.AuthorStyle
	TimestampStyle   = theme.TimestampStyle
	WorkingCopyStyle = theme.WorkingCopyStyle

	FloatingWindowStyle = theme.FloatingWindowStyle
	FloatingTitleStyle  = theme.FloatingTitleStyle

	HelpBarStyle  = theme.HelpBarStyle
	HelpKeyStyle  = theme.HelpKeyStyle
	HelpDescStyle = theme.HelpDescStyle
)

const (
	SidebarWidth      = theme.SidebarWidth
	SidebarMinWidth   = theme.SidebarMinWidth
	SidebarMaxWidth   = theme.SidebarMaxWidth
	PanelMinHeight    = theme.PanelMinHeight
	FloatingLogWidth  = theme.FloatingLogWidth
	FloatingLogHeight = theme.FloatingLogHeight
)
