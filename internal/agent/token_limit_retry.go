package agent

import (
	"github.com/susugadx/xelyon-cli/internal/agent/token"
)

type tokenLimitRetryOptions struct {
	// Plan Mode の checkpoint restore 前に一時的な planning 履歴を保存しないための opt-out。
	skipCompressionPersistence bool
}

// handleTokenLimitErrorWithRetry はトークン上限エラー時に自動圧縮してリトライ
// 戻り値: リトライ成功時はtrue、失敗時はfalse
func (a *Agent) handleTokenLimitErrorWithRetry(err error, retryFunc func() error) bool {
	return a.handleTokenLimitErrorWithRetryOptions(err, retryFunc, tokenLimitRetryOptions{})
}

func (a *Agent) handleTokenLimitErrorWithRetryOptions(err error, retryFunc func() error, opts tokenLimitRetryOptions) bool {
	if !token.IsTokenLimitError(err) {
		return false
	}

	// リトライ制限（最大1回）
	if a.tokenLimitRetryCount >= 1 {
		// 2回目以降は通常のエラー表示
		handleTokenLimitErrorWithWriter(a.output(), err)
		return false
	}

	// 通知表示
	cyan.Fprintln(a.output(), "\n⚡ トークン上限到達。自動圧縮して再実行します...")

	// LLMサマリー方式で圧縮（既存のCompressHistoryを使用）
	// keepRecentはデフォルト値10を使用
	keepRecent := 10
	cfg := a.cfg()
	if cfg.Compression.KeepRecent > 0 {
		keepRecent = cfg.Compression.KeepRecent
	}

	// 履歴が短すぎる場合は圧縮できない
	if len(a.History) <= keepRecent {
		yellow.Fprintln(a.output(), "⚠️  履歴が短すぎるため圧縮できません")
		handleTokenLimitErrorWithWriter(a.output(), err)
		return false
	}

	// 圧縮実行
	yellow.Fprintln(a.output(), "🗜️  会話を圧縮中...")
	if err := a.compressHistory(keepRecent, compressHistoryOptions{skipPersistenceOnSuccess: opts.skipCompressionPersistence}); err != nil {
		red.Fprintf(a.output(), "❌ 自動圧縮に失敗しました: %v\n", err)
		handleTokenLimitErrorWithWriter(a.output(), err)
		return false
	}

	// リトライカウントをインクリメント
	a.tokenLimitRetryCount++

	// リトライ実行
	cyan.Fprintln(a.output(), "🔄 圧縮完了、再実行します...")
	if retryErr := retryFunc(); retryErr != nil {
		// リトライ後もエラーの場合は通常のエラー表示
		if token.IsTokenLimitError(retryErr) {
			red.Fprintln(a.output(), "❌ 圧縮後もトークン上限を超えています")
		}
		handleTokenLimitErrorWithWriter(a.output(), retryErr)
		return false
	}

	// リトライ成功
	green.Fprintln(a.output(), "✅ 自動圧縮＆リトライ成功")
	a.tokenLimitRetryCount = 0 // 成功したらリセット
	return true
}
