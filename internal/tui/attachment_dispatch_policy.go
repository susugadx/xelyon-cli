package tui

import (
	"fmt"
	"strings"
)

func selectPrimaryImagePath(attachments []composerAttachment) string {
	for _, att := range attachments {
		if att.Kind == composerAttachmentImage {
			return att.Path
		}
	}
	return ""
}

func resolveDispatchBasePrompt(trimmedPayload string, imagePath string, attachmentCount int) string {
	if trimmedPayload != "" {
		return trimmedPayload
	}
	switch {
	case imagePath != "":
		return "Please analyze this image."
	case attachmentCount > 0:
		return "添付ファイルを確認して、要点をまとめてください。"
	default:
		return ""
	}
}

func buildDispatchInput(basePrompt string, contextBlocks []string) string {
	finalInput := basePrompt
	if len(contextBlocks) == 0 {
		return finalInput
	}
	if finalInput == "" {
		finalInput = "以下の添付コンテキストを確認してください。"
	}
	return finalInput + "\n\nAttached context:\n" + strings.Join(contextBlocks, "\n\n")
}

func buildDispatchDisplay(trimmedPayload string, basePrompt string, attachments []composerAttachment) string {
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
		b.WriteString(fmt.Sprintf("\n- %s: %s", att.kindLabel(), att.basename()))
	}
	return b.String()
}
