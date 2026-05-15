package ui

const (
	// ToolNameCompress は会話履歴圧縮を TUI/tool line に載せるための表示用 tool 名。
	ToolNameCompress = "compress"

	ToolArgCompressionMode         = "mode"
	ToolArgCompressionReason       = "reason"
	ToolArgCompressionKeepRecent   = "keep_recent"
	ToolArgCompressionBeforeTokens = "before_tokens"
	ToolArgCompressionAfterTokens  = "after_tokens"
	ToolArgCompressionOutcome      = "outcome"

	ToolCompressionModeHistory    = "history"
	ToolCompressionModeCompactAPI = "compact_api"

	ToolCompressionReasonManual     = "manual"
	ToolCompressionReasonAuto       = "auto"
	ToolCompressionReasonTokenLimit = "token-limit"
)
