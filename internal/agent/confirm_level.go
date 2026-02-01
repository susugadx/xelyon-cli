package agent

import (
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// ConfirmLevel は確認レベル
type ConfirmLevel string

const (
	ConfirmAll       ConfirmLevel = "all"       // 全ツール確認
	ConfirmDangerous ConfirmLevel = "dangerous" // 危険ツールのみ
	ConfirmNone      ConfirmLevel = "none"      // 確認なし
)

// ShouldConfirmTool はツール実行前に確認が必要か判定
// 注: 計画承認は常にあり、これは実行中の確認のみ
func ShouldConfirmTool(toolName string, level string) bool {
	switch level {
	case string(ConfirmAll):
		return true
	case string(ConfirmDangerous):
		safety := common.GetToolSafety(toolName)
		return safety == common.SafetyLow
	case string(ConfirmNone):
		return false
	default:
		// デフォルトは dangerous
		safety := common.GetToolSafety(toolName)
		return safety == common.SafetyLow
	}
}

// IsDangerousTool は危険なツールかを判定
func IsDangerousTool(toolName string) bool {
	safety := common.GetToolSafety(toolName)
	return safety == common.SafetyLow
}

// ValidateConfirmLevel は確認レベルが有効かを判定
func ValidateConfirmLevel(level string) bool {
	switch level {
	case string(ConfirmAll), string(ConfirmDangerous), string(ConfirmNone):
		return true
	default:
		return false
	}
}
