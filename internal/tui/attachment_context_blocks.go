package tui

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	tuiattachments "github.com/susugadx/xelyon-cli/internal/tui/attachments"
)

var readAttachedPDFPreviewForContext = readAttachedPDFPreview

func buildAttachmentContextBlocks(attachments []tuiattachments.Attachment, primaryImagePath string) []string {
	specs := tuiattachments.ContextBlockSpecs(attachments, primaryImagePath)
	contextBlocks := make([]string, 0, len(specs))
	for _, spec := range specs {
		switch spec.Kind {
		case tuiattachments.ContextBlockImagePath:
			contextBlocks = append(contextBlocks, buildAttachedImagePathContext(spec.Path))
		case tuiattachments.ContextBlockFile:
			contextBlocks = append(contextBlocks, buildAttachedFileContext(spec.Path))
		}
	}
	return contextBlocks
}

func buildAttachedImagePathContext(path string) string {
	return tuiattachments.BuildAttachedImagePathContext(attachmentDisplayPath(path))
}

func buildAttachedFileContext(path string) string {
	if isPDFAttachmentPath(path) {
		return buildAttachedPDFContext(path)
	}

	displayPath := attachmentDisplayPath(path)
	preview, truncated, binary, err := readAttachedFilePreview(path)
	return tuiattachments.BuildAttachedFileContextBlock(displayPath, preview, truncated, binary, err, maxAttachedFilePreviewBytes)
}

func buildAttachedPDFContext(path string) string {
	displayPath := attachmentDisplayPath(path)
	preview, err := readAttachedPDFPreviewForContext(path)
	return tuiattachments.BuildAttachedPDFContextBlock(displayPath, preview.text, preview.truncated, err, maxAttachedPDFPreviewPages, maxAttachedPDFPreviewChars)
}

func readAttachedFilePreview(path string) (text string, truncated bool, binary bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, false, err
	}
	defer f.Close()

	buf := make([]byte, maxAttachedFilePreviewBytes+1)
	n, readErr := f.Read(buf)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return "", false, false, readErr
	}

	data := buf[:n]
	if n > maxAttachedFilePreviewBytes {
		truncated = true
		data = data[:maxAttachedFilePreviewBytes]
	}
	if looksBinary(data) {
		return "", truncated, true, nil
	}

	txt := string(data)
	if !utf8.ValidString(txt) {
		return "", truncated, true, nil
	}
	return txt, truncated, false, nil
}

func looksBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	// 主要テキスト拡張子でなければ conservative に binary 扱いしない。
	return false
}

func attachmentDisplayPath(path string) string {
	if rel, err := filepath.Rel(".", path); err == nil && rel != "" && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}
