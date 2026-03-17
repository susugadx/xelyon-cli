package api

import (
	"bufio"
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
	out := outputWriterFromContext(ctx)
	errOut := errorWriterFromContext(ctx)

	var fullResponse strings.Builder
	firstChunk := true
	inToolJSON := false     // ツールJSON内にいるか
	jsonDepth := 0          // JSONのネスト深度
	inString := false       // JSON文字列リテラル内にいるか
	var prevChar rune       // 前の文字（エスケープ検出用）
	var pendingChunk string // チャンク分割対応: パターンプレフィックスが末尾にある場合に保留

	// チャンネル経由でスキャン結果を受け取る
	type scanResult struct {
		line string
		err  error
		done bool // scanner終了
	}
	lineCh := make(chan scanResult)

	// バックグラウンドでスキャン
	go func() {
		defer close(lineCh)
		scanner := bufio.NewScanner(resp.Body)
		// バッファサイズを10MBに拡張（Function Callingの長いレスポンス対応）
		buf := make([]byte, 0, 10*1024*1024)
		scanner.Buffer(buf, 10*1024*1024)
		for scanner.Scan() {
			lineCh <- scanResult{line: scanner.Text()}
		}
		if err := scanner.Err(); err != nil {
			lineCh <- scanResult{err: err, done: true}
		} else {
			lineCh <- scanResult{done: true}
		}
	}()

	// アイドルタイマー
	idleTimer := time.NewTimer(idleTimeout)
	defer idleTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			spinner.Stop()
			partialResponse := fullResponse.String()
			if partialResponse != "" {
				// 部分結果がある場合は警告と共に返す
				if !firstChunk {
					_, _ = fmt.Fprintln(out) // 改行
				}
				yellow.Fprintln(errOut, "\n⚠️  Response interrupted. Partial result returned.")
				return partialResponse, nil // エラーではなく部分結果を返す
			}
			return "", ctx.Err()

		case <-idleTimer.C:
			// アイドルタイムアウト
			spinner.Stop()
			return fullResponse.String(), fmt.Errorf("idle timeout: no data received for %v", idleTimeout)

		case result, ok := <-lineCh:
			if !ok {
				// チャンネルクローズ（予期しない終了）
				spinner.Stop()
				if !firstChunk {
					_, _ = fmt.Fprintln(out)
				}
				return fullResponse.String(), nil
			}

			// アイドルタイマーをリセット（データ受信ごとに）
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(idleTimeout)

			if result.done {
				// スキャン完了
				spinner.Stop()
				if result.err != nil {
					return fullResponse.String(), fmt.Errorf("scanner error: %w", result.err)
				}
				if !firstChunk {
					_, _ = fmt.Fprintln(out)
				}
				return fullResponse.String(), nil
			}

			line := result.line
			if line == "" {
				continue
			}

			// プロバイダー固有のパース処理
			content, done, err := parser(line)
			if err != nil {
				spinner.Stop()
				if !firstChunk {
					_, _ = fmt.Fprintln(out)
				}
				return fullResponse.String(), err
			}

			if done {
				spinner.Stop()
				// チャンク分割対応: 保留分を flush（パターンが完成しなかった = テキスト）
				if pendingChunk != "" {
					_, _ = fmt.Fprint(out, pendingChunk)
				}
				if !firstChunk {
					_, _ = fmt.Fprintln(out)
				}
				return fullResponse.String(), nil
			}

			if content != "" {
				// fullResponseには常に追加（内部処理用）
				fullResponse.WriteString(content)

				// チャンク分割対応: 前回保留分と結合
				if pendingChunk != "" {
					content = pendingChunk + content
					pendingChunk = ""
				}

				// チャンク末尾がパターンプレフィックスに一致する場合、表示を保留
				if !inToolJSON {
					if prefixLen := matchesPatternPrefix(content); prefixLen > 0 {
						pendingChunk = content[len(content)-prefixLen:]
						content = content[:len(content)-prefixLen]
					}
				}

				// ツールJSON検出・非表示処理
				displayContent := filterToolJSON(content, &inToolJSON, &jsonDepth, &inString, &prevChar)

				// ツールJSON開始時にスピナーを表示
				if inToolJSON && spinner != nil && !spinner.IsActive() {
					msg := "Preparing..."
					responseStr := fullResponse.String()
					if strings.Contains(responseStr, "write_file") {
						msg = ui.SpinnerMessageForTool("write_file")
					} else if strings.Contains(responseStr, "str_replace") {
						msg = ui.SpinnerMessageForTool("str_replace")
					}
					spinner.Start(msg)
				}

				// 最初のコンテンツでスピナー停止 + AI発言ヘッダー表示
				if firstChunk {
					spinner.Stop()
					firstChunk = false
					PrintAIHeaderWithContext(ctx)
				}
				if displayContent != "" {
					_, _ = fmt.Fprint(out, displayContent)
				}
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
// - prevChar: エスケープシーケンス検出用（\"を無視するため）
func filterToolJSON(content string, inToolJSON *bool, jsonDepth *int, inString *bool, prevChar *rune) string {
	var result strings.Builder
	remaining := content

	for len(remaining) > 0 {
		if *inToolJSON {
			// ツールJSON内: 終了位置を探しながら非表示
			endIdx := -1
			for i, ch := range remaining {
				if ch == '"' && *prevChar != '\\' {
					*inString = !*inString
				} else if !*inString {
					if ch == '{' {
						*jsonDepth++
					} else if ch == '}' {
						*jsonDepth--
						if *jsonDepth == 0 {
							*inToolJSON = false
							*inString = false
							endIdx = i + 1 // '}' の次の位置
							*prevChar = ch
							break
						}
					}
				}
				*prevChar = ch
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
		*prevChar = 0
	}

	return result.String()
}
