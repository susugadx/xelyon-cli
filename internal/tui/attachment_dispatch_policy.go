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
	return fallbackDispatchPrompt(imagePath, attachmentCount)
}

func buildDispatchInput(basePrompt string, contextBlocks []string) string {
	finalInput := basePrompt
	if len(contextBlocks) == 0 {
		return finalInput
	}
	if finalInput == "" {
		finalInput = dispatchDefaultContextPromptJP
	}
	return finalInput + "\n\n" + dispatchContextSectionTitle + "\n" + strings.Join(contextBlocks, "\n\n")
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
