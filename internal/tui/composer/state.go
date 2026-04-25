package composer

import (
	"strings"
	"unicode/utf8"
)

// PasteBlockFoldThreshold は paste を folded block として扱う文字数しきい値。
const PasteBlockFoldThreshold = 160

// PartKind は composer payload を構成する part 種別を表す。
type PartKind int

const (
	// PartText は通常テキスト part を表す。
	PartText PartKind = iota
	// PartPaste は folded paste block part を表す。
	PartPaste
)

// Part は composer payload 内の text または paste block 参照を表す。
type Part struct {
	Kind     PartKind
	Text     string
	PasteUID int
}

// PasteBlock は folded paste の内容と表示用 metadata を保持する。
type PasteBlock struct {
	UID       int
	Content   string
	CharCount int
	LineCount int
}

// VisiblePasteBlock は表示順番号つきの paste block を表す。
type VisiblePasteBlock struct {
	Block  PasteBlock
	Number int
}

// VisibleRow は composer footer に表示する folded row を表す。
type VisibleRow struct {
	Kind       PartKind
	Text       string
	PasteBlock VisiblePasteBlock
}

// State は folded paste を含む composer payload 状態を保持する。
type State struct {
	Parts        []Part
	PasteBlocks  []PasteBlock
	NextPasteUID int
}

// Clear は composer state を初期化する。
func (s *State) Clear() {
	s.Parts = nil
	s.PasteBlocks = nil
	s.NextPasteUID = 0
}

// HasDraft は入力欄または folded state に draft があるかを返す。
func (s State) HasDraft(input string) bool {
	return strings.TrimSpace(input) != "" || len(s.Parts) > 0 || len(s.PasteBlocks) > 0
}

// HasSubmittableContent は送信対象になる内容があるかを返す。
func (s State) HasSubmittableContent(input string) bool {
	if strings.TrimSpace(input) != "" {
		return true
	}
	for _, part := range s.Parts {
		switch part.Kind {
		case PartText:
			if strings.TrimSpace(part.Text) != "" {
				return true
			}
		case PartPaste:
			block, ok := s.FindPasteBlock(part.PasteUID)
			if ok && block.Content != "" {
				return true
			}
		}
	}
	return false
}

// HasFoldedPasteBlocks は folded paste block があるかを返す。
func (s State) HasFoldedPasteBlocks() bool {
	return len(s.PasteBlocks) > 0
}

// IsPlainInput は folded state を持たない通常入力かどうかを返す。
func (s State) IsPlainInput() bool {
	return len(s.Parts) == 0 && len(s.PasteBlocks) == 0
}

// FindPasteBlock は UID に対応する paste block を返す。
func (s State) FindPasteBlock(uid int) (PasteBlock, bool) {
	for _, block := range s.PasteBlocks {
		if block.UID == uid {
			return block, true
		}
	}
	return PasteBlock{}, false
}

// BuildPayload は folded state と現在の input から送信用 payload を構築する。
func (s State) BuildPayload(input string) string {
	var builder strings.Builder
	for _, part := range s.Parts {
		switch part.Kind {
		case PartText:
			builder.WriteString(part.Text)
		case PartPaste:
			block, ok := s.FindPasteBlock(part.PasteUID)
			if ok {
				builder.WriteString(block.Content)
			}
		}
	}
	builder.WriteString(input)
	return builder.String()
}

// AppendText は text part を追加する。直前が text part の場合は結合する。
func (s *State) AppendText(text string) {
	if text == "" {
		return
	}
	if n := len(s.Parts); n > 0 && s.Parts[n-1].Kind == PartText {
		s.Parts[n-1].Text += text
		return
	}
	s.Parts = append(s.Parts, Part{Kind: PartText, Text: text})
}

// AppendPasteBlock は folded paste block を追加する。
func (s *State) AppendPasteBlock(content string) {
	if content == "" {
		return
	}
	s.NextPasteUID++
	block := PasteBlock{
		UID:       s.NextPasteUID,
		Content:   content,
		CharCount: utf8.RuneCountInString(content),
		LineCount: strings.Count(content, "\n") + 1,
	}
	s.PasteBlocks = append(s.PasteBlocks, block)
	s.Parts = append(s.Parts, Part{Kind: PartPaste, PasteUID: block.UID})
}

// RemoveLastPasteBlock は最後の folded paste block を削除し、末尾 text part を返す。
func (s *State) RemoveLastPasteBlock() (string, bool) {
	if len(s.PasteBlocks) == 0 {
		return "", false
	}
	lastUID := s.PasteBlocks[len(s.PasteBlocks)-1].UID
	s.PasteBlocks = s.PasteBlocks[:len(s.PasteBlocks)-1]
	for i := len(s.Parts) - 1; i >= 0; i-- {
		if s.Parts[i].Kind != PartPaste || s.Parts[i].PasteUID != lastUID {
			continue
		}
		s.Parts = append(s.Parts[:i], s.Parts[i+1:]...)
		break
	}
	return s.PopTrailingText(), true
}

// PopTrailingText は末尾の連続した text part を取り除き、元の順序で返す。
func (s *State) PopTrailingText() string {
	if len(s.Parts) == 0 {
		return ""
	}
	tail := make([]string, 0, len(s.Parts))
	for len(s.Parts) > 0 {
		last := s.Parts[len(s.Parts)-1]
		if last.Kind != PartText {
			break
		}
		tail = append(tail, last.Text)
		s.Parts = s.Parts[:len(s.Parts)-1]
	}
	if len(tail) == 0 {
		return ""
	}
	var builder strings.Builder
	for i := len(tail) - 1; i >= 0; i-- {
		builder.WriteString(tail[i])
	}
	return builder.String()
}

// VisibleRows は footer に表示する composer row を返す。
func (s State) VisibleRows(maxVisible int) []VisibleRow {
	if len(s.Parts) == 0 || maxVisible <= 0 {
		return nil
	}
	rows := make([]VisibleRow, 0, len(s.Parts))
	pasteNumber := 0
	for _, part := range s.Parts {
		switch part.Kind {
		case PartText:
			if part.Text == "" {
				continue
			}
			rows = append(rows, VisibleRow{Kind: PartText, Text: part.Text})
		case PartPaste:
			block, ok := s.FindPasteBlock(part.PasteUID)
			if !ok {
				continue
			}
			pasteNumber++
			rows = append(rows, VisibleRow{Kind: PartPaste, PasteBlock: VisiblePasteBlock{Block: block, Number: pasteNumber}})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	if len(rows) <= maxVisible {
		return rows
	}
	return rows[len(rows)-maxVisible:]
}

// NormalizePastedText は paste text の改行コードを LF に正規化する。
func NormalizePastedText(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

// ShouldFoldPasteBlock は paste text を folded block として扱うかを返す。
func ShouldFoldPasteBlock(content string) bool {
	content = NormalizePastedText(content)
	return strings.Contains(content, "\n") || utf8.RuneCountInString(content) >= PasteBlockFoldThreshold
}

// SplitRunesAt は rune position で文字列を左右に分割する。
func SplitRunesAt(str string, pos int) (string, string) {
	runes := []rune(str)
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}
	return string(runes[:pos]), string(runes[pos:])
}
