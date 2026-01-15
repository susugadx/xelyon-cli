package tools

import "os"

// IsInteractiveModeEnabled は対話的確認モードが有効かチェック
// デフォルトは有効。
//
// 無効化したい場合は以下のいずれかを設定:
//   - XELYON_INTERACTIVE_CONFIRM=0
//   - XELYON_INTERACTIVE_CONFIRM=false
func IsInteractiveModeEnabled() bool {
	v := os.Getenv("XELYON_INTERACTIVE_CONFIRM")
	switch v {
	case "0", "false", "FALSE":
		return false
	default:
		return true
	}
}

// ConfirmWithFeedback は対話的確認を行う（互換ラッパー）
// interactiveMode=true の場合は y/n/c、false の場合は従来の y/n
//
// NOTE: 新コードでは Confirm() / ConfirmDecision の利用を推奨。
func ConfirmWithFeedback(message string) (approved bool, comment string, image *ImageData) {
	dec := Confirm(message)
	switch dec.Action {
	case ConfirmYes:
		return true, "", nil
	case ConfirmComment:
		return false, dec.Comment, dec.Image
	default:
		return false, "", nil
	}
}
