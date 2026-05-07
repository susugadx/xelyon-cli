package theme

// ConfigPalette は /config 画面の ANSI color palette。
type ConfigPalette struct {
	BgNormal   string
	BgSelected string
	BgInactive string
	BgHeader   string
	FgNormal   string
	FgDim      string
	FgBright   string
	FgGreen    string
	FgYellow   string
	FgRed      string
	FgCyan     string
	Reset      string
	Bold       string
}

// Config は /config 画面で使う既定 palette。
var Config = ConfigPalette{
	BgNormal:   "\033[48;5;235m",
	BgSelected: "\033[48;5;25m",
	BgInactive: "\033[48;5;238m",
	BgHeader:   "\033[48;5;236m",
	FgNormal:   "\033[38;5;252m",
	FgDim:      "\033[38;5;244m",
	FgBright:   "\033[38;5;255m",
	FgGreen:    "\033[38;5;82m",
	FgYellow:   "\033[38;5;220m",
	FgRed:      "\033[38;5;196m",
	FgCyan:     "\033[38;5;87m",
	Reset:      "\033[0m",
	Bold:       "\033[1m",
}

// ViewportPalette は navigation/selection viewport の ANSI color palette。
type ViewportPalette struct {
	CursorLineBg     string
	CursorCharBg     string
	VisualBg         string
	VisualCursorBg   string
	MouseSelectionBg string
}

// Viewport は viewport overlay で使う既定 palette。
var Viewport = ViewportPalette{
	CursorLineBg:     "\033[48;5;236m",
	CursorCharBg:     "\033[48;5;255;38;5;16m",
	VisualBg:         "\033[48;5;240m",
	VisualCursorBg:   "\033[48;5;255;38;5;16m",
	MouseSelectionBg: "\033[48;5;240m",
}

// ChromePalette は input dock と status bar の ANSI color palette。
type ChromePalette struct {
	InputBg                 string
	InputTextFg             string
	InputDraftFg            string
	InputPasteFg            string
	InputPasteID            string
	InputPrompt             string
	InputRowMarkerFg        string
	InputMetaMarkerFg       string
	SuggestionBg            string
	SuggestionSelectedBg    string
	SuggestionPrefixFg      string
	SuggestionCommandFg     string
	SuggestionDescFg        string
	SuggestionSelectedFg    string
	SuggestionSelectedDimFg string
	StatusFg                string
	StatusSepFg             string
	StatusPathFg            string
	HintFg                  string
	NavBadge                string
	NewOutput               string
	SuccessFg               string
	Reset                   string
}

// Chrome は chat chrome で使う既定 palette。
var Chrome = ChromePalette{
	InputBg:                 "\033[48;5;236m",
	InputTextFg:             "\033[38;5;252m",
	InputDraftFg:            "\033[38;5;252m",
	InputPasteFg:            "\033[38;5;250m",
	InputPasteID:            "\033[38;5;117m",
	InputPrompt:             "\033[38;5;46m",
	InputRowMarkerFg:        "\033[38;5;108m",
	InputMetaMarkerFg:       "\033[38;5;81m",
	SuggestionBg:            "\033[48;5;235m",
	SuggestionSelectedBg:    "\033[48;5;24m",
	SuggestionPrefixFg:      "\033[38;5;244m",
	SuggestionCommandFg:     "\033[38;5;117m",
	SuggestionDescFg:        "\033[38;5;250m",
	SuggestionSelectedFg:    "\033[38;5;255m",
	SuggestionSelectedDimFg: "\033[38;5;254m",
	StatusFg:                "\033[38;5;252m",
	StatusSepFg:             "\033[38;5;240m",
	StatusPathFg:            "\033[38;5;246m",
	HintFg:                  "\033[38;5;244m",
	NavBadge:                "\033[48;5;33;38;5;255m",
	NewOutput:               "\033[48;5;63;38;5;230m",
	SuccessFg:               "\033[38;5;82m",
	Reset:                   "\033[0m",
}
