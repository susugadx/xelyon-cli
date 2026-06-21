package tui

import tuiattachments "github.com/susugadx/xelyon-cli/internal/tui/attachments"

func selectPrimaryImagePath(attachments []tuiattachments.Attachment) string {
	return tuiattachments.SelectPrimaryImagePath(attachments)
}

func resolveDispatchBasePrompt(trimmedPayload string, imagePath string, attachmentCount int) string {
	return tuiattachments.ResolveDispatchBasePrompt(trimmedPayload, imagePath, attachmentCount)
}

func buildDispatchInput(basePrompt string, contextBlocks []string) string {
	return tuiattachments.BuildDispatchInput(basePrompt, contextBlocks)
}

func buildDispatchDisplay(trimmedPayload string, basePrompt string, attachments []tuiattachments.Attachment) string {
	return tuiattachments.BuildDispatchDisplay(trimmedPayload, basePrompt, attachments)
}
