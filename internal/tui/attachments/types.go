package attachments

import (
	"path/filepath"
	"strings"
)

// Kind は composer 添付の種類を表す。
type Kind int

const (
	// KindFile は通常ファイル添付を表す。
	KindFile Kind = iota
	// KindImage は画像添付を表す。
	KindImage
)

// Source は composer 添付が追加された経路を表す。
type Source int

const (
	// SourceUnknown は追加経路が未指定の添付を表す。
	SourceUnknown Source = iota
	// SourceDroppedPath は貼り付け/ドロップされた path 由来の添付を表す。
	SourceDroppedPath
	// SourceClipboardImage は clipboard screenshot 由来の添付を表す。
	SourceClipboardImage
	// SourceCommand は /attach command 由来の添付を表す。
	SourceCommand
)

// Attachment は composer に保持する添付値である。
type Attachment struct {
	Kind   Kind
	Source Source
	Path   string
	Size   int64
}

// MaxComposerAttachments は composer が保持できる添付数の上限である。
const MaxComposerAttachments = 12

// Basename は添付 path の表示用 basename を返す。
func (a Attachment) Basename() string {
	base := filepath.Base(a.Path)
	if base == "." || base == string(filepath.Separator) {
		return a.Path
	}
	return base
}

// KindLabel は添付種類の表示ラベルを返す。
func (a Attachment) KindLabel() string {
	if a.Kind == KindImage {
		return "image"
	}
	return "file"
}

// AppendResult は添付追加判定の結果である。
type AppendResult int

const (
	// AppendAdded は添付を追加できる状態を表す。
	AppendAdded AppendResult = iota
	// AppendRejectedEmptyPath は path が空で追加できない状態を表す。
	AppendRejectedEmptyPath
	// AppendRejectedDuplicate は同一 path が既にあり追加できない状態を表す。
	AppendRejectedDuplicate
	// AppendRejectedLimit は添付上限に達していて追加できない状態を表す。
	AppendRejectedLimit
)

// PrepareAppend は添付追加前の path 正規化と重複/上限判定を行う。
func PrepareAppend(existing []Attachment, att Attachment, limit int) (Attachment, AppendResult) {
	path := strings.TrimSpace(att.Path)
	if path == "" {
		return Attachment{}, AppendRejectedEmptyPath
	}
	att.Path = path
	for _, current := range existing {
		if current.Path == att.Path {
			return att, AppendRejectedDuplicate
		}
	}
	if len(existing) >= limit {
		return att, AppendRejectedLimit
	}
	return att, AppendAdded
}
