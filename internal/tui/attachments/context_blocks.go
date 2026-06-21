package attachments

import (
	"fmt"
	"strings"
)

// ContextBlockKind は添付 context block の種類を表す。
type ContextBlockKind int

const (
	// ContextBlockImagePath は追加画像 path context を表す。
	ContextBlockImagePath ContextBlockKind = iota
	// ContextBlockFile は file preview context を表す。
	ContextBlockFile
)

// ContextBlockSpec は root が I/O 境界で context block を作るための pure DTO である。
type ContextBlockSpec struct {
	Kind ContextBlockKind
	Path string
}

// ContextBlockSpecs は送信入力へ追加する添付 context block の候補を返す。
func ContextBlockSpecs(attachments []Attachment, primaryImagePath string) []ContextBlockSpec {
	specs := make([]ContextBlockSpec, 0, len(attachments))
	for _, att := range attachments {
		switch att.Kind {
		case KindImage:
			if att.Path == primaryImagePath {
				continue
			}
			specs = append(specs, ContextBlockSpec{Kind: ContextBlockImagePath, Path: att.Path})
		case KindFile:
			specs = append(specs, ContextBlockSpec{Kind: ContextBlockFile, Path: att.Path})
		}
	}
	return specs
}

// BuildAttachedImagePathContext は追加画像 path の context block を組み立てる。
func BuildAttachedImagePathContext(displayPath string) string {
	return fmt.Sprintf("[Attached image path]\n%s", displayPath)
}

// BuildAttachedFileContextBlock は file preview の context block を組み立てる。
func BuildAttachedFileContextBlock(displayPath string, preview string, truncated bool, binary bool, readErr error, maxPreviewBytes int) string {
	if readErr != nil {
		return fmt.Sprintf("[Attached file: %s]\n<failed to read: %v>", displayPath, readErr)
	}
	if binary {
		return fmt.Sprintf("[Attached file: %s]\n<binary file omitted>", displayPath)
	}

	block := fmt.Sprintf("[Attached file: %s]\n%s", displayPath, preview)
	if truncated {
		block += fmt.Sprintf("\n\n<content truncated: first %d bytes shown>", maxPreviewBytes)
	}
	return block
}

// BuildAttachedPDFContextBlock は PDF preview の context block を組み立てる。
func BuildAttachedPDFContextBlock(displayPath string, text string, truncated bool, readErr error, maxPreviewPages int, maxPreviewChars int) string {
	if readErr != nil {
		return fmt.Sprintf("[Attached file: %s]\n<failed to read PDF: %v>", displayPath, readErr)
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Sprintf("[Attached file: %s]\n<no extractable text in PDF>", displayPath)
	}

	block := fmt.Sprintf("[Attached file: %s]\n%s", displayPath, text)
	if truncated {
		block += fmt.Sprintf("\n\n<PDF content truncated: first %d pages / %d chars shown>", maxPreviewPages, maxPreviewChars)
	}
	return block
}
