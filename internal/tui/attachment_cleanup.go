package tui

import (
	"os"
	"path/filepath"
)

const clipboardAttachmentFileName = "clipboard.png"

func cleanupTemporaryAttachment(att composerAttachment) {
	if !att.Temporary || att.Path == "" {
		return
	}

	if filepath.Base(att.Path) == clipboardAttachmentFileName {
		_ = os.RemoveAll(filepath.Dir(att.Path))
		return
	}
	_ = os.Remove(att.Path)
}
