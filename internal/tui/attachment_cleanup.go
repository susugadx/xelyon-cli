package tui

import (
	"os"
	"path/filepath"

	tuiattachments "github.com/susugadx/xelyon-cli/internal/tui/attachments"
)

const (
	clipboardAttachmentTempDirPrefix = "xelyon-clipboard-image-"
	clipboardAttachmentFileName      = "clipboard.png"
)

func cleanupTemporaryAttachment(att tuiattachments.Attachment) {
	if att.Source != tuiattachments.SourceClipboardImage || att.Path == "" {
		return
	}

	if filepath.Base(att.Path) == clipboardAttachmentFileName {
		_ = os.RemoveAll(filepath.Dir(att.Path))
		return
	}
	_ = os.Remove(att.Path)
}
