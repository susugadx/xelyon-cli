package api

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// toolJSONPatterns はツールJSON開始パターン
var toolJSONPatterns = []string{
	`{"tool"`,
	`{ "tool"`,
}

// maxPatternLen は最長パターンの長さ
var maxPatternLen = 8 // `{ "tool"` の長さ

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
	cfg := config.GetGlobalConfig()
	idleTimeout := time.Duration(cfg.Streaming.IdleTimeoutSeconds) * time.Second

	var fullResponse strings.Builder
	var displayBuffer strings.Builder // 表示用バッファ
	firstChunk := true
	inToolJSON := false // ツールJSON内にいるか
	jsonDepth := 0      // JSONのネスト深度
	inString := false   // JSON文字列リテラル内にいるか
	var prevChar rune   // 前の文字（エスケープ検出用）

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
			return "", ctx.Err()

		case <-idleTimer.C:
			// アイドルタイムアウト
			spinner.Stop()
			return fullResponse.String(), fmt.Errorf("idle timeout: no data received for %v", idleTimeout)

		case result, ok := <-lineCh:
			if !ok {
				// チャンネルクローズ（予期しない終了）
				if !firstChunk {
					fmt.Println()
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
				if result.err != nil {
					spinner.Stop()
					return fullResponse.String(), fmt.Errorf("scanner error: %w", result.err)
				}
				if !firstChunk {
					fmt.Println()
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
				// パースエラーは無視して次の行へ
				continue
			}

			if done {
				if !firstChunk {
					fmt.Println()
				}
				return fullResponse.String(), nil
			}

			if content != "" {
				// fullResponseには常に追加（内部処理用）
				fullResponse.WriteString(content)

				// ツールJSON検出・非表示処理
				displayContent := filterToolJSON(content, &displayBuffer, &inToolJSON, &jsonDepth, &inString, &prevChar)

				if displayContent != "" {
					// 最初のコンテンツでスピナー停止
					if firstChunk {
						spinner.Stop()
						firstChunk = false
					}
					fmt.Print(displayContent)
				}
			}
		}
	}
}

// filterToolJSON はストリーミング中のツールJSONを検知して非表示にする
//
// 設計:
// - displayBuffer: パターンの途中かもしれない文字を保留するバッファ
// - パターンが確定するまで表示を保留し、ツールJSONでないと確定したら出力
// - チャンク境界をまたいでもパターンを正しく検出できる
// - inString: JSON文字列リテラル内かどうかを追跡（文字列内の{}を無視するため）
// - prevChar: エスケープシーケンス検出用（\"を無視するため）
func filterToolJSON(content string, displayBuffer *strings.Builder, inToolJSON *bool, jsonDepth *int, inString *bool, prevChar *rune) string {
	var result strings.Builder

	for _, ch := range content {
		if *inToolJSON {
			// ツールJSON内: 文字列リテラルとネスト深度を追跡、表示しない
			if ch == '"' && *prevChar != '\\' {
				// エスケープされていない " で文字列の開始/終了を切り替え
				*inString = !*inString
			} else if !*inString {
				// 文字列外でのみ {} をカウント
				if ch == '{' {
					*jsonDepth++
				} else if ch == '}' {
					*jsonDepth--
					if *jsonDepth == 0 {
						// JSON終了
						*inToolJSON = false
						*inString = false
						displayBuffer.Reset()
					}
				}
			}
			*prevChar = ch
			continue
		}

		// 通常モード: バッファに追加してパターンチェック
		displayBuffer.WriteRune(ch)
		bufStr := displayBuffer.String()

		// パターン完全一致をチェック
		matched := false
		for _, pattern := range toolJSONPatterns {
			if strings.HasSuffix(bufStr, pattern) {
				// パターン一致 → ツールJSON開始
				*inToolJSON = true
				*jsonDepth = 1 // 最初の { をカウント
				// バッファからパターン部分を除いた分を出力
				if len(bufStr) > len(pattern) {
					result.WriteString(bufStr[:len(bufStr)-len(pattern)])
				}
				displayBuffer.Reset()
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		// パターンの途中かもしれないかチェック
		// いずれかのパターンのプレフィックスと一致するか
		isPotentialMatch := false
		for _, pattern := range toolJSONPatterns {
			// バッファがパターンのプレフィックスになっているか
			if len(bufStr) < len(pattern) {
				if strings.HasPrefix(pattern, bufStr) {
					isPotentialMatch = true
					break
				}
			}
			// バッファの末尾がパターンのプレフィックスになっているか
			for prefixLen := 1; prefixLen < len(pattern) && prefixLen <= len(bufStr); prefixLen++ {
				suffix := bufStr[len(bufStr)-prefixLen:]
				prefix := pattern[:prefixLen]
				if suffix == prefix {
					isPotentialMatch = true
					break
				}
			}
			if isPotentialMatch {
				break
			}
		}

		if isPotentialMatch {
			// パターンの途中かもしれない → 保留を継続
			continue
		}

		// パターンではないと確定 → バッファの内容を出力
		result.WriteString(bufStr)
		displayBuffer.Reset()
	}

	return result.String()
}
