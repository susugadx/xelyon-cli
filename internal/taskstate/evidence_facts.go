package taskstate

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// Evidence はタスク判断の根拠テキストを保持する。
type Evidence struct {
	items []evidenceFact
}

func (g Evidence) clone() Evidence {
	if len(g.items) == 0 {
		return Evidence{}
	}
	items := make([]evidenceFact, len(g.items))
	copy(items, g.items)
	return Evidence{items: items}
}

// Items は記録済み根拠を防御コピーで返す。
func (g Evidence) Items() []evidenceFact {
	if len(g.items) == 0 {
		return nil
	}
	items := make([]evidenceFact, len(g.items))
	copy(items, g.items)
	return items
}

// Pointers は記録済み根拠を外部向け pointer の防御コピーで返す。
func (g Evidence) Pointers() []EvidencePointer {
	return evidencePointersFromFacts(g.items)
}

// Len は記録済み根拠数を返す。
func (g Evidence) Len() int {
	return len(g.items)
}

func (g *Evidence) record(fact evidenceFact) {
	fact, ok := prepareEvidenceFact(fact)
	if g == nil || !ok || g.contains(fact) {
		return
	}
	if len(g.items) >= maxEvidenceItems {
		return
	}
	g.items = append(g.items, fact)
}

func prepareEvidenceFact(fact evidenceFact) (evidenceFact, bool) {
	fact = normalizeEvidenceFact(fact)
	return fact, validEvidenceFact(fact)
}

func normalizeEvidenceFact(fact evidenceFact) evidenceFact {
	fact.startLine, fact.endLine = tools.NormalizeObservationLineRange(fact.startLine, fact.endLine)
	return fact
}

func validEvidenceFact(fact evidenceFact) bool {
	return fact.path != "" &&
		fact.startLine > 0 &&
		fact.endLine > 0 &&
		fact.endLine >= fact.startLine &&
		strings.TrimSpace(fact.excerpt) != ""
}

func (g Evidence) contains(fact evidenceFact) bool {
	for _, item := range g.items {
		if item.path == fact.path &&
			item.startLine == fact.startLine &&
			item.endLine == fact.endLine &&
			item.source == fact.source &&
			item.toolCallID == fact.toolCallID &&
			item.excerpt == fact.excerpt {
			return true
		}
	}
	return false
}

type evidenceFact struct {
	path       string
	startLine  int
	endLine    int
	source     string
	toolCallID string
	fileHash   string
	stale      bool
	excerpt    string
}

// Path は evidence の対象ファイルパスを返す。
func (f evidenceFact) Path() string {
	return f.path
}

// StartLine は evidence の開始行を返す。
func (f evidenceFact) StartLine() int {
	return f.startLine
}

// EndLine は evidence の終了行を返す。
func (f evidenceFact) EndLine() int {
	return f.endLine
}

// Text は evidence の本文を返す。
func (f evidenceFact) Text() string {
	return f.excerpt
}

// Excerpt は evidence の短い抜粋を返す。
func (f evidenceFact) Excerpt() string {
	return f.excerpt
}

// Source は evidence の出所を返す。
func (f evidenceFact) Source() string {
	return f.source
}

// ToolCallID は evidence の元 tool call id を返す。
func (f evidenceFact) ToolCallID() string {
	return f.toolCallID
}

// FileHash は evidence の対象ファイル hash を返す。P0a では空値のまま保持する。
func (f evidenceFact) FileHash() string {
	return f.fileHash
}

// Stale は evidence が古い可能性を返す。P0a では常に false。
func (f evidenceFact) Stale() bool {
	return f.stale
}

// EvidencePointersFromState は RuntimeTaskState から evidence pointer を防御コピーで返す。
func EvidencePointersFromState(state RuntimeTaskState) []EvidencePointer {
	return state.Evidence.Pointers()
}
