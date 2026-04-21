package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// 警告表示用の色
var yellow = color.New(color.FgYellow)

// AI発言ヘッダー表示用の色
var cyanBold = color.New(color.FgCyan, color.Bold)

// PrintAIHeader はAI発言開始時のヘッダーを表示する
// 全プロバイダーから共通で使用
func PrintAIHeader() {
	PrintAIHeaderWithContext(context.Background())
}

// toolJSONPatterns はツールJSON開始パターン
var toolJSONPatterns = []string{
	`{"tool"`,
	`{ "tool"`,
	`{"id"`,
	`{ "id"`,
	`{"name"`,  // DeepSeek が OpenAI 互換形式で出力するパターン
	`{ "name"`, // 同上（スペース付き）
}

// StreamParser はストリーミングレスポンスの1行をパースする関数型
// 戻り値: (content string, done bool, err error)
//   - content: この行から抽出されたテキストコンテンツ
//   - done: ストリームの終了を示すフラグ
//   - err: パースエラー
type StreamParser func(line string) (content string, done bool, err error)

// ParseStreamingResponse は共通のストリーミングレスポンス処理
// コンテキストキャンセル、スピナー制御、エラーハンドリングを統一的に処理
// アイドルタイムアウト方式: データ受信がない状態がN秒続くとタイムアウト
// ツールJSON部分は内部で記録するが表示しない
func ParseStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner, parser StreamParser) (string, error) {
	cfg := config.FromContext(ctx)
	idleTimeout := time.Duration(cfg.Streaming.IdleTimeoutSeconds) * time.Second
	errOut := errorWriterFromContext(ctx)

	controller := NewStreamLoopController(resp.Body, StreamLoopOptions{
		IdleTimeout:         idleTimeout,
		AutoResetIdleOnLine: true,
	})
	defer controller.Stop()

	state := newStreamOutputState(ctx, spinner)

	for {
		event := controller.Next(ctx, nil)
		switch event.Type {
		case StreamLoopEventContextDone:
			return state.finalizeContextDone(errOut, event.Err)

		case StreamLoopEventIdleTimeout:
			// アイドルタイムアウト
			state.stopSpinner()
			return state.response(), fmt.Errorf("idle timeout: no data received for %v", idleTimeout)

		case StreamLoopEventScannerDone:
			return state.finalizeScannerDone(event.Err)

		case StreamLoopEventLine:
			done, err := state.processLine(event.Line, parser)
			if err != nil {
				return state.finalizeParserError(err)
			}
			if done {
				return state.finalizeDone()
			}
		}
	}
}

// matchesPatternPrefix はチャンク末尾がツールJSONパターンのプレフィックスに
// 一致する長さを返す。一致しない場合は 0。
// チャンク分割対応: 次チャンクと結合してからパターン判定するため。
func matchesPatternPrefix(content string) int {
	for _, pattern := range toolJSONPatterns {
		for prefixLen := len(pattern) - 1; prefixLen >= 1; prefixLen-- {
			prefix := pattern[:prefixLen]
			if strings.HasSuffix(content, prefix) {
				return prefixLen
			}
		}
	}
	return 0
}

// filterToolJSON はストリーミング中のツールJSONを検知して非表示にする
//
// 設計（簡略化版）:
// - inToolJSON が true の間は全て非表示
// - strings.Index でパターンを検出（チャンク単位でシンプルに判定）
// - inString: JSON文字列リテラル内かどうかを追跡（文字列内の{}を無視するため）
// - escaped: JSON文字列内のエスケープシーケンス追跡用（\\ や \" を正しく処理する）
func filterToolJSON(content string, inToolJSON *bool, jsonDepth *int, inString *bool, escaped *bool) string {
	var result strings.Builder
	remaining := content

	for len(remaining) > 0 {
		if *inToolJSON {
			// ツールJSON内: 終了位置を探しながら非表示
			endIdx := -1
			for i, ch := range remaining {
				if *inString {
					if *escaped {
						// escape sequence consumed
						*escaped = false
					} else if ch == '\\' {
						*escaped = true
					} else if ch == '"' {
						*inString = false
					}
				} else {
					if ch == '"' {
						*inString = true
						*escaped = false
					} else if ch == '{' {
						*jsonDepth++
					} else if ch == '}' {
						*jsonDepth--
						if *jsonDepth == 0 {
							*inToolJSON = false
							*inString = false
							*escaped = false
							endIdx = i + 1 // '}' の次の位置
							break
						}
					}
				}
			}

			if endIdx == -1 {
				// JSON終了なし → 残り全て非表示
				return result.String()
			}
			// JSON終了 → 残りを継続処理
			remaining = remaining[endIdx:]
			continue
		}

		// パターン検出
		foundIdx := -1
		patternLen := 0
		for _, pattern := range toolJSONPatterns {
			if idx := strings.Index(remaining, pattern); idx != -1 {
				if foundIdx == -1 || idx < foundIdx {
					foundIdx = idx
					patternLen = len(pattern)
				}
			}
		}

		if foundIdx == -1 {
			// パターンなし → 残り全て表示
			result.WriteString(remaining)
			return result.String()
		}

		// パターン前の部分を出力
		result.WriteString(remaining[:foundIdx])
		// パターン自体をスキップし、inToolJSON状態に移行する
		remaining = remaining[foundIdx+patternLen:]

		// パターン以降を処理開始
		*inToolJSON = true
		*jsonDepth = 1 // パターンに含まれる最初の '{' をカウントした状態から開始
		*inString = false
		*escaped = false
	}

	return result.String()
}
