package taskstate

// RecommendedReads は後続で読むべきファイルを初出順で保持する。
type RecommendedReads struct {
	items []recommendedReadFact
}

func (g RecommendedReads) clone() RecommendedReads {
	if len(g.items) == 0 {
		return RecommendedReads{}
	}
	items := make([]recommendedReadFact, len(g.items))
	copy(items, g.items)
	return RecommendedReads{items: items}
}

// Items は記録済み推奨 read を防御コピーで返す。
func (g RecommendedReads) Items() []recommendedReadFact {
	if len(g.items) == 0 {
		return nil
	}
	items := make([]recommendedReadFact, len(g.items))
	copy(items, g.items)
	return items
}

// Len は記録済み推奨 read 数を返す。
func (g RecommendedReads) Len() int {
	return len(g.items)
}

func (g *RecommendedReads) record(path, reason string) {
	g.recordFact(recommendedReadFact{path: path, reason: reason})
}

func (g *RecommendedReads) recordFact(fact recommendedReadFact) {
	if g == nil || fact.path == "" || g.contains(fact.path) {
		return
	}
	if len(g.items) >= maxRecommendedReads {
		return
	}
	g.items = append(g.items, fact)
}

func (g RecommendedReads) contains(path string) bool {
	for _, item := range g.items {
		if item.path == path {
			return true
		}
	}
	return false
}

type recommendedReadFact struct {
	path       string
	reason     string
	source     string
	toolCallID string
}

// Path は推奨されたファイルパスを返す。
func (f recommendedReadFact) Path() string {
	return f.path
}

// Reason は推奨理由を返す。
func (f recommendedReadFact) Reason() string {
	return f.reason
}

// Source は推奨 read の出所を返す。
func (f recommendedReadFact) Source() string {
	return f.source
}

// ToolCallID は推奨 read の元 tool call id を返す。
func (f recommendedReadFact) ToolCallID() string {
	return f.toolCallID
}
