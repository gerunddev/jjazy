package messages

// FileSelectedMsg is sent when a file is selected in FilesPanel
type FileSelectedMsg struct {
	Path string
}

// RevisionSelectedMsg is sent when a revision is selected in LogOverlay
type RevisionSelectedMsg struct {
	RevisionID string
}

// DiffContentMsg carries diff content to be displayed in DiffViewer
type DiffContentMsg struct {
	Content string
	Title   string
}
