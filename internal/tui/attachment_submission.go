package tui

import "strings"

const maxAttachedFilePreviewBytes = 64 * 1024
const maxAttachedPDFPreviewPages = 20
const maxAttachedPDFPreviewChars = 30000

type chatDispatchRequest struct {
	display   string
	input     string
	imagePath string
}

func buildChatDispatchRequest(payload string, attachments []composerAttachment) chatDispatchRequest {
	trimmedPayload := strings.TrimSpace(payload)
	imagePath := selectPrimaryImagePath(attachments)
	basePrompt := resolveDispatchBasePrompt(trimmedPayload, imagePath, len(attachments))
	contextBlocks := buildAttachmentContextBlocks(attachments, imagePath)
	finalInput := buildDispatchInput(basePrompt, contextBlocks)
	display := buildDispatchDisplay(trimmedPayload, basePrompt, attachments)

	return chatDispatchRequest{
		display:   strings.TrimSpace(display),
		input:     strings.TrimSpace(finalInput),
		imagePath: imagePath,
	}
}
