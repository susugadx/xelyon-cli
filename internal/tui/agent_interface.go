package tui

// AgentInterface は tui パッケージから agent パッケージへの依存を逆転させる。
// agent パッケージがこのインターフェースを実装する。
type AgentInterface interface {
	// Chat はユーザー入力をAIに送信する（非同期、goroutineで呼ぶ）
	Chat(input string)

	// HandleCommand は /status, /use, /clear 等のコマンドを処理する。
	// 処理した場合 true を返す。
	HandleCommand(cmd string) bool

	// GetStatusLine はステータスバーに表示する文字列を返す。
	GetStatusLine() string

	// Cancel は現在のAPI呼び出しをキャンセルする。
	Cancel()

	// Cleanup は終了処理を行う。
	Cleanup()

	// IsProcessing はAI処理中（chat実行中）かどうかを返す。
	IsProcessing() bool
}
