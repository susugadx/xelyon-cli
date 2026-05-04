package tui

const (
	dispatchDefaultImagePrompt        = "Please analyze this image."
	dispatchDefaultAttachmentPromptJP = "添付ファイルを確認して、要点をまとめてください。"
	dispatchDefaultContextPromptJP    = "以下の添付コンテキストを確認してください。"
	dispatchContextSectionTitle       = "Attached context:"
)

func fallbackDispatchPrompt(imagePath string, attachmentCount int) string {
	switch {
	case imagePath != "":
		return dispatchDefaultImagePrompt
	case attachmentCount > 0:
		return dispatchDefaultAttachmentPromptJP
	default:
		return ""
	}
}
