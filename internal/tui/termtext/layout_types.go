package termtext

// VisualRow は 1 本の raw line を折り返した後の表示行を表す。
type VisualRow struct {
	RawLineIdx  int
	SubRowIdx   int
	Content     string
	Width       int
	PrefixWidth int
}

// Layout は raw line と visual row の相互マッピングを保持する。
type Layout struct {
	Rows         []VisualRow
	LineToRowMap []int
	RowToLineMap []int
	Width        int
}
