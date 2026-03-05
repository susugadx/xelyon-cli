package common

import "strings"

// ToolSafety はツールの安全性レベル
type ToolSafety int

const (
	// SafetyHigh - 読み取り専用操作（常に自動承認OK）
	SafetyHigh ToolSafety = iota
	// SafetyMedium - 書き込み操作（--auto-approve で承認）
	SafetyMedium
	// SafetyLow - 破壊的操作（--auto-approve で自動承認可、通常は確認必須）
	SafetyLow
)

// ToolSafetyLevels は各ツールの安全性レベルを定義
var ToolSafetyLevels = map[string]ToolSafety{
	// SafetyHigh: 読み取り専用操作
	"read_file":         SafetyHigh,
	"read_files":        SafetyHigh,
	"search_code":       SafetyHigh,
	"list_dir":          SafetyHigh,
	"git_status":        SafetyHigh,
	"git_log":           SafetyHigh,
	"git_diff":          SafetyHigh,
	"ask_user_question": SafetyHigh,

	// SafetyMedium: 外部APIコール（確認対象）
	"web_search": SafetyMedium, // Serper APIにクエリ送信

	// SafetyMedium: 書き込み操作（リカバリ可能）
	"write_file":     SafetyMedium,
	"str_replace":    SafetyMedium,
	"copy_file":      SafetyMedium,
	"create_dir":     SafetyMedium,
	"git_add":        SafetyMedium,
	"git_reset_soft": SafetyMedium,

	// SafetyLow: 破壊的操作（--auto-approve で自動承認可、通常は確認必須）
	"delete_file":    SafetyLow,
	"git_push":       SafetyLow,
	"git_branch":     SafetyLow,
	"git_stash":      SafetyLow,
	"git_reset_hard": SafetyLow,
	"git_force_push": SafetyLow,
	"bash":           SafetyLow,
	"command":        SafetyLow,
}

// GetToolSafety は指定されたツールの安全性レベルを返す
// 定義されていないツールは SafetyMedium（中レベル）として扱う
func GetToolSafety(toolName string) ToolSafety {
	// MCP tools are registered dynamically with names like "mcp_<server>_<tool>".
	// Treat them as SafetyLow by default so they are never auto-approved.
	// NOTE: This only affects call sites that consult GetToolSafety (e.g. ConfirmWithAutoApproveDecision).
	if strings.HasPrefix(toolName, "mcp_") {
		return SafetyLow
	}

	if level, ok := ToolSafetyLevels[toolName]; ok {
		return level
	}
	// デフォルトは SafetyMedium（慎重に）
	return SafetyMedium
}

// IsAutoApprovable は --auto-approve フラグで自動承認可能かを判定
// --auto-approve 有効時は SafetyLow 含む全ツールを自動承認する。
// ハードブロック（blockedCommands, パストラバーサル）は別レイヤーで保護。
func IsAutoApprovable(toolName string, autoApprove bool) bool {
	return autoApprove
}

// IsSafeToolAutoApprovable は SafetyHigh ツールを設定に基づいて自動承認するか判定
// config.tool_confirm.auto_approve_safe が true の場合、SafetyHigh ツールは確認なしで実行
func IsSafeToolAutoApprovable(toolName string) bool {
	safety := GetToolSafety(toolName)
	return safety == SafetyHigh
}

// IsMediumToolAutoApprovable は SafetyMedium ツールを設定に基づいて自動承認するか判定
// config.tool_confirm.auto_approve_medium が true の場合、SafetyMedium ツールは確認なしで実行
func IsMediumToolAutoApprovable(toolName string) bool {
	safety := GetToolSafety(toolName)
	return safety == SafetyMedium
}

// GetSafetyDescription は安全性レベルの説明を返す
func GetSafetyDescription(safety ToolSafety) string {
	switch safety {
	case SafetyHigh:
		return "Safe (read-only)"
	case SafetyMedium:
		return "Moderate (reversible changes)"
	case SafetyLow:
		return "Dangerous (destructive operation)"
	default:
		return "Unknown"
	}
}
