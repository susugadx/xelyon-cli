package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

var readAttachedPDFPreviewForContext = readAttachedPDFPreview

func buildAttachmentContextBlocks(attachments []composerAttachment, primaryImagePath string) []string {
	contextBlocks := make([]string, 0, len(attachments))
	for _, att := range attachments {
		switch att.Kind {
		case composerAttachmentImage:
			if att.Path == primaryImagePath {
				continue
			}
			contextBlocks = append(contextBlocks, buildAttachedImagePathContext(att.Path))
		case composerAttachmentFile:
			contextBlocks = append(contextBlocks, buildAttachedFileContext(att.Path))
		}
	}
	return contextBlocks
}

func buildAttachedImagePathContext(path string) string {
	return fmt.Sprintf("[Attached image path]\n%s", attachmentDisplayPath(path))
}

func buildAttachedFileContext(path string) string {
	if isPDFAttachmentPath(path) {
		return buildAttachedPDFContext(path)
	}

	displayPath := attachmentDisplayPath(path)
	preview, truncated, binary, err := readAttachedFilePreview(path)
	if err != nil {
		return fmt.Sprintf("[Attached file: %s]\n<failed to read: %v>", displayPath, err)
	}
	if binary {
		return fmt.Sprintf("[Attached file: %s]\n<binary file omitted>", displayPath)
	}

	block := fmt.Sprintf("[Attached file: %s]\n%s", displayPath, preview)
	if truncated {
		block += fmt.Sprintf("\n\n<content truncated: first %d bytes shown>", maxAttachedFilePreviewBytes)
	}
	return block
}

func buildAttachedPDFContext(path string) string {
	displayPath := attachmentDisplayPath(path)
	preview, err := readAttachedPDFPreviewForContext(path)
	if err != nil {
		return fmt.Sprintf("[Attached file: %s]\n<failed to read PDF: %v>", displayPath, err)
	}
	if strings.TrimSpace(preview.text) == "" {
		return fmt.Sprintf("[Attached file: %s]\n<no extractable text in PDF>", displayPath)
	}

	block := fmt.Sprintf("[Attached file: %s]\n%s", displayPath, preview.text)
	if preview.truncated {
		block += fmt.Sprintf("\n\n<PDF content truncated: first %d pages / %d chars shown>", maxAttachedPDFPreviewPages, maxAttachedPDFPreviewChars)
	}
	return block
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
