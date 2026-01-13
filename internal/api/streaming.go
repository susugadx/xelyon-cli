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

// StreamParser はストリーミングレスポンスの1行をパースする関数型
// 戻り値: (content string, done bool, err error)
//   - content: この行から抽出されたテキストコンテンツ
//   - done: ストリームの終了を示すフラグ
//   - err: パースエラー
type StreamParser func(line string) (content string, done bool, err error)

// ParseStreamingResponse は共通のストリーミングレスポンス処理
// コンテキストキャンセル、スピナー制御、エラーハンドリングを統一的に処理
// アイドルタイムアウト方式: データ受信がない状態がN秒続くとタイムアウト
func ParseStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner, parser StreamParser) (string, error) {
	cfg := config.GetGlobalConfig()
	idleTimeout := time.Duration(cfg.Streaming.IdleTimeoutSeconds) * time.Second

	var fullResponse strings.Builder
	firstChunk := true

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
				fmt.Println()
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
				fmt.Println()
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
				fmt.Println()
				return fullResponse.String(), nil
			}

			if content != "" {
				// 最初のコンテンツでスピナー停止
				if firstChunk {
					spinner.Stop()
					firstChunk = false
				}

				fmt.Print(content)
				fullResponse.WriteString(content)
			}
		}
	}
}
