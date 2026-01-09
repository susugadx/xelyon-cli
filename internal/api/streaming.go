package api

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"

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
func ParseStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner, parser StreamParser) (string, error) {
	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	firstChunk := true

	for scanner.Scan() {
		// contextキャンセルチェック
		select {
		case <-ctx.Done():
			spinner.Stop()
			return "", ctx.Err()
		default:
		}

		line := scanner.Text()
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
			break
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

	// scanner.Err()チェック（重要: scanner終了後のエラーを検出）
	if err := scanner.Err(); err != nil {
		spinner.Stop()
		return fullResponse.String(), fmt.Errorf("scanner error: %w", err)
	}

	fmt.Println()
	return fullResponse.String(), nil
}
