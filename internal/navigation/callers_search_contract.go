package navigation

const (
	// maxRipgrepResults は findReferences が返す参照の上限。
	// これを超える結果が検出された場合は truncated=true を返す。
	maxRipgrepResults = 500

	ripgrepScannerInitialBufferSize = 64 * 1024
	ripgrepScannerMaxBufferSize     = 1024 * 1024
)

// referenceSearchResult は ripgrep 参照検索の内部状態を保持する。
type referenceSearchResult struct {
	Refs          []Reference
	Truncated     bool
	Incomplete    bool
	StopRequested bool
}
