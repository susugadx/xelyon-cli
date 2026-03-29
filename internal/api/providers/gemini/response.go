package gemini

import (
	"bufio"
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
	out := api.OutputWriterFromContext(ctx)
	errOut := api.ErrorWriterFromContext(ctx)
	var fullResponse strings.Builder
	var functionCalls []*api.GeminiFunctionCall
	var thoughtParts []map[string]any // Gemini 3: thought パートを収集（次リクエストに返す）
	var rescuedToolJSONs []string     // FC救済: コードブロックから抽出したツールJSON
	var headerPrinted bool            // テキスト応答時のAIヘッダー表示済みフラグ
	var contentNewlineEmitted bool    // スピナー上書き防止用
	var usage *GeminiUsageMetadata
	var suppressingToolJSON bool // テキストパート内のツールJSON抑制中フラグ
	var toolJSONDepth int        // ツールJSON の {} ネスト深度
	var toolJSONInStr bool       // ツールJSON 内の文字列リテラルフラグ
	var streamStarted bool       // SSEストリーム開始フラグ（スピナー切り替え用）

	// goroutine + channel でスキャン（ctx キャンセル・transport idle timeout 対応）
	type scanResult struct {
		line string
		err  error
		done bool
	}
	lineCh := make(chan scanResult)

	go func() {
		defer close(lineCh)
		scanner := bufio.NewScanner(resp.Body)
		// バッファサイズを10MBに拡張（thought_signature + thought_parts で1行が巨大になる対応）
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

	cfg := config.FromContext(ctx)
	transportIdleTimeout := time.Duration(cfg.Streaming.IdleTimeoutSeconds) * time.Second
	transportIdleTimer := time.NewTimer(transportIdleTimeout)
	defer transportIdleTimer.Stop()

	// thinking/progress タイマー: actionable output を受信せず thought のみが続く場合のタイムアウト
	thinkingTimeout := time.Duration(cfg.Streaming.ThinkingTimeoutSeconds) * time.Second
	if thinkingTimeout <= 0 {
		thinkingTimeout = 300 * time.Second // フォールバック
	}
	thinkingTimer := time.NewTimer(thinkingTimeout)
	defer thinkingTimer.Stop()
	hadActionableOutput := false // text/FC/救済済み tool JSON を1つでも受信したら true
	thinkingRetries := 0

	var scanErr error

loop:
	for {
		select {
		case <-ctx.Done():
			if spinner != nil {
				spinner.Stop()
			}
			partial := fullResponse.String()
			if partial != "" {
				return partial, nil
			}
			return "", ctx.Err()

		case <-transportIdleTimer.C:
			if spinner != nil {
				spinner.Stop()
			}
			return fullResponse.String(), &ErrIdleTimeout{Message: fmt.Sprintf("transport idle timeout: no valid SSE data received for %v", transportIdleTimeout)}

		case <-thinkingTimer.C:
			// thought は来ているが actionable output が進まない場合のタイムアウト
			if !hadActionableOutput {
				if spinner != nil {
					spinner.Stop()
				}
				return fullResponse.String(), &ErrThinkingTimeout{Message: fmt.Sprintf("thinking timeout: no actionable output received for %v (thought/progress data may have arrived, but no text or function call was produced)", thinkingTimeout)}
			}
			// hadActionableOutput == true: text/FC を受信済みだが、その後 progress が止まっている
			// → タイマーをリセットして再度待つ（最大2回まで）
			thinkingRetries++
			if thinkingRetries >= 2 {
				if spinner != nil {
					spinner.Stop()
				}
				return fullResponse.String(), &ErrThinkingTimeout{Message: fmt.Sprintf("thinking timeout: extended thinking exceeded %v with no new output", thinkingTimeout*time.Duration(thinkingRetries+1))}
			}
			thinkingTimer.Reset(thinkingTimeout)

		case result, ok := <-lineCh:
			if !ok {
				// チャンネルクローズ（予期しない終了）
				break loop
			}

			if result.done {
				scanErr = result.err
				break loop
			}

			line := result.line
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			var chunk GeminiFunctionResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				if debug {
					fmt.Fprintf(errOut, "[DEBUG Gemini SSE] Failed to unmarshal chunk: %v\n", err)
				}
				continue
			}
			resetTimer(transportIdleTimer, transportIdleTimeout)

			// SSEストリーム開始: "Waiting for Gemini..." → thinking メッセージに切り替え
			if !streamStarted {
				streamStarted = true
				if spinner != nil && thinkingMsg != "" {
					spinner.Stop()
					spinner.Start(thinkingMsg)
				}
			}

			if chunk.UsageMetadata != nil {
				usage = chunk.UsageMetadata
			}

			if len(chunk.Candidates) == 0 {
				continue
			}

			for _, part := range chunk.Candidates[0].Content.Parts {
				// Gemini 3: thought パートを収集（表示はしないが次リクエストに返す必要がある）
				if part.Thought {
					tp := map[string]any{"thought": true}
					if part.Text != "" {
						tp["text"] = part.Text
					}
					if part.ThoughtSignature != "" {
						tp["thought_signature"] = part.ThoughtSignature
					}
					thoughtParts = append(thoughtParts, tp)
					if debug {
						sig := part.ThoughtSignature
						if len(sig) > 20 {
							sig = sig[:20] + "..."
						}
						fmt.Fprintf(errOut, "[DEBUG Gemini SSE] Collected thought part (text=%d chars, sig=%q)\n", len(part.Text), sig)
					}
					continue
				}

				// thoughtSignature を含むパート（thought=false）を収集し、テキスト出力を抑制
				// Gemini 3.1 Pro: signature 付きパートの Text が生出力に漏れるのを防ぐ
				// FunctionCall が同時に存在する場合も、テキスト出力は抑制しつつ FC は収集する
				if part.ThoughtSignature != "" {
					tp := map[string]any{"thought_signature": part.ThoughtSignature}
					if part.Text != "" {
						tp["text"] = part.Text
					}
					thoughtParts = append(thoughtParts, tp)
					if debug {
						sig := part.ThoughtSignature
						if len(sig) > 20 {
							sig = sig[:20] + "..."
						}
						fmt.Fprintf(errOut, "[DEBUG Gemini SSE] Collected signature part (text=%d chars, sig=%q, hasFC=%v)\n", len(part.Text), sig, part.FunctionCall != nil)
					}
					// FunctionCall が同時にある場合は FC も収集してから continue
					if part.FunctionCall != nil {
						hadActionableOutput = true
						thinkingRetries = 0
						resetTimer(thinkingTimer, thinkingTimeout)
						if spinner != nil {
							spinner.Stop()
							if headerPrinted && !contentNewlineEmitted {
								_, _ = fmt.Fprintln(out)
								contentNewlineEmitted = true
							}
							spinner.Start(ui.SpinnerMessageForTool(part.FunctionCall.Name))
						}
						part.FunctionCall.ThoughtSignature = part.ThoughtSignature
						functionCalls = append(functionCalls, part.FunctionCall)
					}
					continue
				}

				if part.Text != "" {
					hadActionableOutput = true
					thinkingRetries = 0
					resetTimer(thinkingTimer, thinkingTimeout)
					// ツールJSON抑制中: 後続チャンクも非表示にする（チャンク分割対応）
					if suppressingToolJSON {
						fullResponse.WriteString(part.Text)
						updateToolJSONDepth(part.Text, &toolJSONDepth, &toolJSONInStr)
						if toolJSONDepth <= 0 {
							suppressingToolJSON = false
						}
						continue
					}

					trimmed := strings.TrimSpace(part.Text)
					// ツールJSONプレフィックスの場合は表示せず fullResponse に記録のみ
					// 複数チャンクに分割される場合は depth tracking で後続も抑制
					if isToolJSONPrefix(trimmed) {
						suppressingToolJSON = true
						toolJSONDepth = 0
						toolJSONInStr = false
						updateToolJSONDepth(part.Text, &toolJSONDepth, &toolJSONInStr)
						if toolJSONDepth <= 0 {
							suppressingToolJSON = false
						}
						fullResponse.WriteString(part.Text)
						continue
					}
					// コードブロック内のツールJSON救済（ループ内で即時分離）
					extracted, remaining := extractCodeBlockToolJSON(part.Text)
					if len(extracted) > 0 {
						rescuedToolJSONs = append(rescuedToolJSONs, extracted...)
						if strings.TrimSpace(remaining) != "" {
							if !headerPrinted {
								if spinner != nil {
									spinner.Stop()
								}
								api.PrintAIHeaderWithContext(ctx)
								headerPrinted = true
							}
							_, _ = fmt.Fprint(out, remaining)
						}
						fullResponse.WriteString(remaining)
						continue
					}
					// 通常テキスト: 最初の表示可能コンテンツでスピナー停止
					if !headerPrinted {
						if spinner != nil {
							spinner.Stop()
						}
						api.PrintAIHeaderWithContext(ctx)
						headerPrinted = true
					}
					_, _ = fmt.Fprint(out, part.Text)
					fullResponse.WriteString(part.Text)
				}

				if part.FunctionCall != nil {
					hadActionableOutput = true
					thinkingRetries = 0
					resetTimer(thinkingTimer, thinkingTimeout)
					// テキスト表示後にFCが来た場合、ツール準備中スピナーを再開
					if spinner != nil {
						spinner.Stop()
						if headerPrinted && !contentNewlineEmitted {
							_, _ = fmt.Fprintln(out)
							contentNewlineEmitted = true
						}
						spinner.Start(ui.SpinnerMessageForTool(part.FunctionCall.Name))
					}
					part.FunctionCall.ThoughtSignature = part.ThoughtSignature
					functionCalls = append(functionCalls, part.FunctionCall)
				}
			}
		}
	}

	// Gemini 3: thought パートを全 functionCall に付与
	// NOTE: Gemini 3 仕様: thought パートは1ターン内の全 FC で共有される。
	// 履歴再構築時（function_calling.go）は ToolCalls[0].ThoughtParts のみを使用し重複を防止。
	if len(thoughtParts) > 0 {
		for _, fc := range functionCalls {
			fc.ThoughtParts = thoughtParts
		}
	}

	if spinner != nil {
		spinner.Stop()
	}

	if scanErr != nil {
		return "", fmt.Errorf("SSE scan error: %w", scanErr)
	}

	// FC が空の場合、テキストから救済したツールJSONを使用
	if len(functionCalls) == 0 && len(rescuedToolJSONs) > 0 {
		if debug {
			fmt.Fprintf(errOut, "[DEBUG Gemini SSE] Rescuing %d tool call(s) from text\n", len(rescuedToolJSONs))
		}
		fmt.Fprintf(errOut, "⚠️  FC rescue: %d tool call(s) extracted from text response\n", len(rescuedToolJSONs))
		for _, tj := range rescuedToolJSONs {
			fullResponse.WriteString(tj)
		}
	}

	// FunctionCall を出力（重複排除: signature を除いた表示用JSONでキー比較）
	seenTools := make(map[string]bool)
	for _, fc := range functionCalls {
		displayKey := convertFunctionCallToDisplayJSON(fc)
		if seenTools[displayKey] {
			continue
		}
		seenTools[displayKey] = true
		// 内部用は ThoughtSignature/ThoughtParts を含む完全 JSON
		fullResponse.WriteString(convertFunctionCallToToolJSON(fc))
	}

	if usage != nil && p.usageCallback != nil {
		p.usageCallback(api.Usage{
			InputTokens:       usage.PromptTokenCount,
			OutputTokens:      usage.CandidatesTokenCount,
			ThinkingTokens:    usage.ThoughtsTokenCount,
			CachedInputTokens: usage.CachedContentTokenCount,
		})
	}

	if fullResponse.Len() == 0 {
		return "", fmt.Errorf("no content in Gemini SSE response (stream ended without generating any text or function calls)")
	}

	if !contentNewlineEmitted {
		_, _ = fmt.Fprintln(out)
	}
	return fullResponse.String(), nil
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
