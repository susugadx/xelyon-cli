package tools

import "os"

// IsInteractiveModeEnabled は対話的確認モードが有効かチェック
// 環境変数 XELYON_INTERACTIVE_CONFIRM=1 で有効化
func IsInteractiveModeEnabled() bool {
	return os.Getenv("XELYON_INTERACTIVE_CONFIRM") == "1"
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
