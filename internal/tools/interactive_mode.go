package tools

import "os"

// IsInteractiveModeEnabled は対話的確認モードが有効かチェック
// 環境変数 XELYON_INTERACTIVE_CONFIRM=1 で有効化
func IsInteractiveModeEnabled() bool {
	return os.Getenv("XELYON_INTERACTIVE_CONFIRM") == "1"
}

// ConfirmWithFeedback は対話的確認を行う
// interactiveMode=true の場合は y/n/c、false の場合は従来の y/n
func ConfirmWithFeedback(message string) (approved bool, comment string) {
	if !IsInteractiveModeEnabled() {
		// 従来の confirm() を使用
		approved = confirm(message)
		return approved, ""
	}

	// 対話的確認モード
	result := ConfirmInteractive(message)
	switch result.Action {
	case "yes":
		return true, ""
	case "comment":
		// コメント = 修正要求 = 実行しない
		return false, result.Comment
	default: // "no"
		return false, ""
	}
}
