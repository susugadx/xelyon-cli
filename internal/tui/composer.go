package tui

const pasteBlockFoldThreshold = 160

type composerPartKind int

const (
	composerPartText composerPartKind = iota
	composerPartPaste
)

type composerPart struct {
	kind     composerPartKind
	text     string
	pasteUID int
}

type pasteBlock struct {
	uid       int
	content   string
	charCount int
	lineCount int
}

type visiblePasteBlock struct {
	block  pasteBlock
	number int
}

type visibleComposerRow struct {
	kind       composerPartKind
	text       string
	pasteBlock visiblePasteBlock
}
