package tui

import (
	"os"
	"path/filepath"
)

const (
	clipboardAttachmentTempDirPrefix = "xelyon-clipboard-image-"
	clipboardAttachmentFileName      = "clipboard.png"
)

func cleanupTemporaryAttachment(att composerAttachment) {
	if att.Source != composerAttachmentSourceClipboardImage || att.Path == "" {
		return
	}

	if filepath.Base(att.Path) == clipboardAttachmentFileName {
		_ = os.RemoveAll(filepath.Dir(att.Path))
		return
	}
	_ = os.Remove(att.Path)
}
