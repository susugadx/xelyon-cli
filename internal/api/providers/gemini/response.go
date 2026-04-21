package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ErrThinkingTimeout はthought は流れているが actionable output が進まない場合のタイムアウトエラー
type ErrThinkingTimeout struct {
	Message string
}

// Error はエラーメッセージを返す
func (e *ErrThinkingTimeout) Error() string {
	return e.Message
}

// isThinkingTimeoutError はエラーがErrThinkingTimeoutかどうかを判定する
func isThinkingTimeoutError(err error) bool {
	var target *ErrThinkingTimeout
	return errors.As(err, &target)
}

// ErrIdleTimeout はSSE ストリームで有効な data を受信できない場合の transport idle timeout エラー
type ErrIdleTimeout struct {
	Message string
}

func (e *ErrIdleTimeout) Error() string {
	return e.Message
}

// ErrResponseStartTimeout はHTTPレスポンスヘッダー受信前のタイムアウトエラー
// Google側でリクエスト処理が詰まり、SSEストリームが開始されない場合に発生
type ErrResponseStartTimeout struct {
	Message string
}

// Error はエラーメッセージを返す
func (e *ErrResponseStartTimeout) Error() string {
	return e.Message
}

// handleSSEResponse は streamGenerateContent?alt=sse の SSE ストリームを処理する
// thinkingMsg はSSEストリーム開始時にスピナーを切り替えるメッセージ（空なら切り替えなし）
func (p *Provider) handleSSEResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner, thinkingMsg string) (string, error) {
	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"
	state := newSSEInterpretState(ctx, spinner, thinkingMsg, debug)
	// どの return 経路でもスピナー停止を保証し、描画崩れを防ぐ。
	defer state.stopSpinner()

	cfg := config.FromContext(ctx)
	transportIdleTimeout := time.Duration(cfg.Streaming.IdleTimeoutSeconds) * time.Second
	controller := api.NewStreamLoopController(resp.Body, api.StreamLoopOptions{
		IdleTimeout:         transportIdleTimeout,
		AutoResetIdleOnLine: false, // Gemini は「有効な data chunk」を受信したときだけ idle をリセットする
	})
	defer controller.Stop()

	// thinking/progress タイマー: actionable output を受信せず thought のみが続く場合のタイムアウト
	thinkingTimeout := time.Duration(cfg.Streaming.ThinkingTimeoutSeconds) * time.Second
	if thinkingTimeout <= 0 {
		thinkingTimeout = 300 * time.Second // フォールバック
	}
	thinkingTimer := time.NewTimer(thinkingTimeout)
	defer thinkingTimer.Stop()

	var scanErr error

loop:
	for {
		eventResult := controller.Next(ctx, thinkingTimer.C)
		switch eventResult.Type {
		case api.StreamLoopEventContextDone:
			partial := state.response()
			if partial != "" {
				return partial, nil
			}
			if eventResult.Err != nil {
				return "", eventResult.Err
			}
			return "", ctx.Err()

		case api.StreamLoopEventIdleTimeout:
			return state.response(), &ErrIdleTimeout{Message: fmt.Sprintf("transport idle timeout: no valid SSE data received for %v", transportIdleTimeout)}

		case api.StreamLoopEventExternal:
			if err := state.handleThinkingTimeout(thinkingTimeout); err != nil {
				return state.response(), err
			}
			thinkingTimer.Reset(thinkingTimeout)

		case api.StreamLoopEventScannerDone:
			scanErr = eventResult.Err
			break loop

		case api.StreamLoopEventLine:
			data, handled := parseGeminiSSEDataLine(eventResult.Line)
			if !handled {
				continue
			}

			chunk, err := decodeGeminiSSEChunk(data)
			if err != nil {
				state.debugf("[DEBUG Gemini SSE] Failed to unmarshal chunk: %v\n", err)
				continue
			}
			controller.ResetIdleTimer()
			state.processChunk(ctx, chunk, thinkingTimer, thinkingTimeout)
		}
	}

	if scanErr != nil {
		return "", fmt.Errorf("SSE scan error: %w", scanErr)
	}

	return state.finalize(p)
}

func resetTimer(timer *time.Timer, timeout time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(timeout)
}

func parseGeminiSSEDataLine(line string) (string, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false
	}
	return strings.TrimPrefix(line, "data: "), true
}

func decodeGeminiSSEChunk(data string) (GeminiFunctionResponse, error) {
	var chunk GeminiFunctionResponse
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return GeminiFunctionResponse{}, err
	}
	return chunk, nil
}

// updateToolJSONDepth はテキスト中の {} ネスト深度を追跡する（文字列リテラル内は無視）
// SSE テキストパートが複数チャンクに分割された場合に、ツールJSON全体を抑制するために使用
func updateToolJSONDepth(s string, depth *int, inStr *bool) {
	escaped := false
	for _, ch := range s {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && *inStr {
			escaped = true
			continue
		}
		if ch == '"' {
			*inStr = !*inStr
			continue
		}
		if !*inStr {
			switch ch {
			case '{':
				*depth++
			case '}':
				*depth--
			}
		}
	}
}

// isToolJSONPrefix はテキストがツールJSON形式で始まるか判定
func isToolJSONPrefix(s string) bool {
	return strings.HasPrefix(s, `{"tool"`) || strings.HasPrefix(s, `{ "tool"`)
}

// extractCodeBlockToolJSON はテキスト内の ```json...``` コードブロックからツールJSON を抽出する
// 返値: (抽出されたツールJSON, コードブロック除去後のテキスト)
func extractCodeBlockToolJSON(text string) ([]string, string) {
	var toolJSONs []string
	remaining := text
	searchFrom := 0

	for searchFrom < len(remaining) {
		// ``` を探す
		idx := strings.Index(remaining[searchFrom:], "```")
		if idx == -1 {
			break
		}
		blockStart := searchFrom + idx

		// 言語指定をスキップ（```json\n の場合）
		afterTicks := blockStart + 3
		if afterTicks >= len(remaining) {
			break
		}
		nlIdx := strings.Index(remaining[afterTicks:], "\n")
		if nlIdx == -1 {
			break
		}
		contentStart := afterTicks + nlIdx + 1

		// 閉じ ``` を探す
		closeIdx := strings.Index(remaining[contentStart:], "```")
		if closeIdx == -1 {
			break
		}
		contentEnd := contentStart + closeIdx
		blockEnd := contentEnd + 3

		content := strings.TrimSpace(remaining[contentStart:contentEnd])

		if isToolJSONPrefix(content) {
			toolJSONs = append(toolJSONs, content)
			// コードブロック全体を除去
			before := strings.TrimRight(remaining[:blockStart], "\n")
			after := ""
			if blockEnd < len(remaining) {
				after = strings.TrimLeft(remaining[blockEnd:], "\n")
			}
			if before != "" && after != "" {
				remaining = before + "\n" + after
			} else {
				remaining = before + after
			}
			// searchFrom はそのまま（除去で位置がずれるため）
			continue
		}

		// ツールJSONでないブロックはスキップ
		searchFrom = blockEnd
	}

	return toolJSONs, remaining
}
