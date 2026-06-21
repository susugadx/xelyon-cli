package attachments

import (
	"fmt"
	"strings"
)

const (
	dispatchDefaultImagePrompt        = "Please analyze this image."
	dispatchDefaultAttachmentPromptJP = "添付ファイルを確認して、要点をまとめてください。"
	dispatchDefaultContextPromptJP    = "以下の添付コンテキストを確認してください。"
	dispatchContextSectionTitle       = "Attached context:"
)

// SelectPrimaryImagePath は image chat に渡す主画像 path を選ぶ。
func SelectPrimaryImagePath(attachments []Attachment) string {
	for _, att := range attachments {
		if att.Kind == KindImage {
			return att.Path
		}
	}
	return ""
}

// ResolveDispatchBasePrompt は添付送信用の基礎 prompt を決める。
func ResolveDispatchBasePrompt(trimmedPayload string, imagePath string, attachmentCount int) string {
	if trimmedPayload != "" {
		return trimmedPayload
	}
	return fallbackDispatchPrompt(imagePath, attachmentCount)
}

// BuildDispatchInput は provider に送る入力本文を組み立てる。
func BuildDispatchInput(basePrompt string, contextBlocks []string) string {
	finalInput := basePrompt
	if len(contextBlocks) == 0 {
		return finalInput
	}
	if finalInput == "" {
		finalInput = dispatchDefaultContextPromptJP
	}
	return finalInput + "\n\n" + dispatchContextSectionTitle + "\n" + strings.Join(contextBlocks, "\n\n")
}

// BuildDispatchDisplay は chat transcript に表示する添付付き入力を組み立てる。
func BuildDispatchDisplay(trimmedPayload string, basePrompt string, attachments []Attachment) string {
	display := basePrompt
	if display == "" {
		display = trimmedPayload
	}
	if len(attachments) == 0 {
		return display
	}

	var b strings.Builder
	b.WriteString(display)
	b.WriteString("\n\n[Attachments]")
	for _, att := range attachments {
		fmt.Fprintf(&b, "\n- %s: %s", att.KindLabel(), att.Basename())
	}
	return b.String()
}

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
