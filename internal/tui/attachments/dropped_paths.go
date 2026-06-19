package attachments

// DroppedPathParseKind は dropped path parse の分類である。
type DroppedPathParseKind int

const (
	// DroppedPathParseNotPath は paste content を path として扱わない分類である。
	DroppedPathParseNotPath DroppedPathParseKind = iota
	// DroppedPathParseReady は path 候補が parse 済みの分類である。
	DroppedPathParseReady
	// DroppedPathParseLimit は dropped path 数が上限超過した分類である。
	DroppedPathParseLimit
)

// DroppedPathParseResult は dropped path parse の結果 DTO である。
type DroppedPathParseResult struct {
	Kind  DroppedPathParseKind
	Paths []string
}
