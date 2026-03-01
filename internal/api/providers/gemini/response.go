package gemini

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// handleSSEResponse は streamGenerateContent?alt=sse の SSE ストリームを処理する
func (p *Provider) handleSSEResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"
	var fullResponse strings.Builder
	var functionCalls []*api.GeminiFunctionCall
	var thoughtParts []map[string]any // Gemini 3: thought パートを収集（次リクエストに返す）
	var rescuedToolJSONs []string     // FC救済: コードブロックから抽出したツールJSON
	var headerPrinted bool            // テキスト応答時のAIヘッダー表示済みフラグ
	var usage *GeminiUsageMetadata
	var suppressingToolJSON bool // テキストパート内のツールJSON抑制中フラグ
	var toolJSONDepth int        // ツールJSON の {} ネスト深度
	var toolJSONInStr bool       // ツールJSON 内の文字列リテラルフラグ

	// goroutine + channel でスキャン（ctx キャンセル・idle timeout 対応）
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

	cfg := config.GetGlobalConfig()
	idleTimeout := time.Duration(cfg.Streaming.IdleTimeoutSeconds) * time.Second
	idleTimer := time.NewTimer(idleTimeout)
	defer idleTimer.Stop()

	// thinking タイマー: text/FC を受信せず thinking のみが続く場合のタイムアウト
	thinkingTimeout := time.Duration(cfg.Streaming.ThinkingTimeoutSeconds) * time.Second
	if thinkingTimeout <= 0 {
		thinkingTimeout = 300 * time.Second // フォールバック
	}
	thinkingTimer := time.NewTimer(thinkingTimeout)
	defer thinkingTimer.Stop()
	hadOutput := false // text/FC を1つでも受信したら true
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

		case <-idleTimer.C:
			if spinner != nil {
				spinner.Stop()
			}
			return fullResponse.String(), fmt.Errorf("idle timeout: no data received for %v", idleTimeout)

		case <-thinkingTimer.C:
			// thinking のみが続き text/FC が来ない場合のタイムアウト
			if !hadOutput {
				if spinner != nil {
					spinner.Stop()
				}
				return fullResponse.String(), fmt.Errorf("thinking timeout: no text or function call received for %v (thinking data was received but no actionable output)", thinkingTimeout)
			}
			// hadOutput == true: text/FC を受信済みだが thinking が続いている
			// → タイマーをリセットして再度待つ（最大2回まで）
			thinkingRetries++
			if thinkingRetries >= 2 {
				if spinner != nil {
					spinner.Stop()
				}
				return fullResponse.String(), fmt.Errorf("thinking timeout: extended thinking exceeded %v with no new output", thinkingTimeout*time.Duration(thinkingRetries+1))
			}
			thinkingTimer.Reset(thinkingTimeout)

		case result, ok := <-lineCh:
			if !ok {
				// チャンネルクローズ（予期しない終了）
				break loop
			}

			// hadNonThoughtData: このチャンクで text/FC データがあったかを追跡
			// thought パートのみの場合は idleTimer をリセットしない
			hadNonThoughtData := false

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
					fmt.Fprintf(os.Stderr, "[DEBUG Gemini SSE] Failed to unmarshal chunk: %v\n", err)
				}
				continue
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
						fmt.Fprintf(os.Stderr, "[DEBUG Gemini SSE] Collected thought part (text=%d chars, sig=%q)\n", len(part.Text), sig)
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
						fmt.Fprintf(os.Stderr, "[DEBUG Gemini SSE] Collected signature part (text=%d chars, sig=%q, hasFC=%v)\n", len(part.Text), sig, part.FunctionCall != nil)
					}
					// FunctionCall が同時にある場合は FC も収集してから continue
					if part.FunctionCall != nil {
						hadNonThoughtData = true
						if !hadOutput {
							hadOutput = true
							thinkingTimer.Stop()
						}
						if headerPrinted && spinner != nil && !spinner.IsActive() {
							spinner.Start(ui.SpinnerMessageForTool(part.FunctionCall.Name))
						}
						part.FunctionCall.ThoughtSignature = part.ThoughtSignature
						functionCalls = append(functionCalls, part.FunctionCall)
					}
					continue
				}

				if part.Text != "" {
					hadNonThoughtData = true
					// text を受信 → thinking タイマー停止
					if !hadOutput {
						hadOutput = true
						thinkingTimer.Stop()
					}
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
								api.PrintAIHeader()
								headerPrinted = true
							}
							fmt.Print(remaining)
						}
						fullResponse.WriteString(remaining)
						continue
					}
					// 通常テキスト: 最初の表示可能コンテンツでスピナー停止
					if !headerPrinted {
						if spinner != nil {
							spinner.Stop()
						}
						api.PrintAIHeader()
						headerPrinted = true
					}
					fmt.Print(part.Text)
					fullResponse.WriteString(part.Text)
				}

				if part.FunctionCall != nil {
					hadNonThoughtData = true
					// FC を受信 → thinking タイマー停止
					if !hadOutput {
						hadOutput = true
						thinkingTimer.Stop()
					}
					// テキスト表示後にFCが来た場合、ツール準備中スピナーを再開
					if headerPrinted && spinner != nil && !spinner.IsActive() {
						spinner.Start(ui.SpinnerMessageForTool(part.FunctionCall.Name))
					}
					part.FunctionCall.ThoughtSignature = part.ThoughtSignature
					functionCalls = append(functionCalls, part.FunctionCall)
				}
			}

			// idleTimer: text/FC を含むチャンクのみリセット（thought のみではリセットしない）
			if hadNonThoughtData {
				if !idleTimer.Stop() {
					select {
					case <-idleTimer.C:
					default:
					}
				}
				idleTimer.Reset(idleTimeout)
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
			fmt.Fprintf(os.Stderr, "[DEBUG Gemini SSE] Rescuing %d tool call(s) from text\n", len(rescuedToolJSONs))
		}
		fmt.Fprintf(os.Stderr, "⚠️  FC rescue: %d tool call(s) extracted from text response\n", len(rescuedToolJSONs))
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

	fmt.Println()
	return fullResponse.String(), nil
}

// handleFunctionCallingResponse は Function Calling レスポンスを処理
// :generateContent エンドポイントは plain JSON を返すため、SSE パーサーではなくこちらを使用
func (p *Provider) handleFunctionCallingResponse(body []byte, spinner *ui.Spinner) (string, error) {
	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"

	var responses []GeminiFunctionResponse

	// まず配列としてパースを試みる（ストリーミングレスポンス）
	if err := json.Unmarshal(body, &responses); err != nil {
		// 配列でない場合は単一オブジェクトとして試す
		var singleResponse GeminiFunctionResponse
		if err := json.Unmarshal(body, &singleResponse); err != nil {
			if spinner != nil {
				spinner.Stop()
			}
			preview := string(body)
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			return "", fmt.Errorf("failed to parse Function Calling response (body preview: %s): %w", preview, err)
		}
		responses = []GeminiFunctionResponse{singleResponse}
	}

	if spinner != nil {
		spinner.Stop()
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Parsed %d responses\n", len(responses))
	}

	var fullResponse strings.Builder
	var functionCalls []*api.GeminiFunctionCall // FunctionCall を収集
	var thoughtParts []map[string]any           // Gemini 3: thought パートを収集
	var textParts []string                      // テキストパートを収集

	for i, response := range responses {
		if len(response.Candidates) == 0 {
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Response %d: no candidates\n", i)
			}
			continue
		}

		candidate := response.Candidates[0]

		if response.UsageMetadata != nil && p.usageCallback != nil {
			p.usageCallback(api.Usage{
				InputTokens:       response.UsageMetadata.PromptTokenCount,
				OutputTokens:      response.UsageMetadata.CandidatesTokenCount,
				ThinkingTokens:    response.UsageMetadata.ThoughtsTokenCount,
				CachedInputTokens: response.UsageMetadata.CachedContentTokenCount,
			})
		}

		for _, part := range candidate.Content.Parts {
			// Gemini 3: thought パートを収集（次リクエストに返す必要がある）
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
					fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Collected thought part (text=%d chars, sig=%q)\n", len(part.Text), sig)
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
					fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Collected signature part (text=%d chars, sig=%q, hasFC=%v)\n", len(part.Text), sig, part.FunctionCall != nil)
				}
				// FunctionCall が同時にある場合は FC も収集してから continue
				if part.FunctionCall != nil {
					if debug {
						fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Found FunctionCall (with signature): %s\n", part.FunctionCall.Name)
					}
					part.FunctionCall.ThoughtSignature = part.ThoughtSignature
					functionCalls = append(functionCalls, part.FunctionCall)
				}
				continue
			}

			// Function Call パートを収集
			if part.FunctionCall != nil {
				if debug {
					fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Found FunctionCall: %s\n", part.FunctionCall.Name)
				}
				part.FunctionCall.ThoughtSignature = part.ThoughtSignature
				functionCalls = append(functionCalls, part.FunctionCall)
			}

			// テキストパートを収集
			if part.Text != "" {
				if debug {
					if len(part.Text) > 100 {
						fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Text part: %q\n", part.Text[:100])
					} else {
						fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Text part: %q\n", part.Text)
					}
				}
				textParts = append(textParts, part.Text)
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

	// テキストパートを分類: ツールJSON vs 通常テキスト
	var toolJSONTexts []string
	var displayTexts []string
	for _, text := range textParts {
		trimmed := strings.TrimSpace(text)
		if isToolJSONPrefix(trimmed) {
			toolJSONTexts = append(toolJSONTexts, trimmed)
		} else {
			displayTexts = append(displayTexts, text)
		}
	}

	// FC が空の場合、コードブロック内のツールJSON も探す
	if len(functionCalls) == 0 && len(toolJSONTexts) == 0 {
		for i, text := range displayTexts {
			extracted, remaining := extractCodeBlockToolJSON(text)
			if len(extracted) > 0 {
				toolJSONTexts = append(toolJSONTexts, extracted...)
				displayTexts[i] = remaining
			}
		}
	}

	if debug && len(toolJSONTexts) > 0 {
		fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] toolJSONTexts=%d, functionCalls=%d\n",
			len(toolJSONTexts), len(functionCalls))
	}

	// 通常テキストを出力
	headerPrinted := false
	for _, text := range displayTexts {
		if strings.TrimSpace(text) == "" {
			continue
		}
		if !headerPrinted {
			api.PrintAIHeader()
			headerPrinted = true
		}
		fmt.Print(text)
		fullResponse.WriteString(text)
	}

	// FunctionCall がある場合はそちらを出力（重複排除: signature を除いた表示用JSONでキー比較）
	seenTools := make(map[string]bool)
	if len(functionCalls) > 0 {
		for _, fc := range functionCalls {
			displayKey := convertFunctionCallToDisplayJSON(fc)
			if seenTools[displayKey] {
				continue
			}
			seenTools[displayKey] = true
			// 内部用は ThoughtSignature/ThoughtParts を含む完全 JSON
			fullResponse.WriteString(convertFunctionCallToToolJSON(fc))
		}
	} else if len(toolJSONTexts) > 0 {
		// FC が空 → テキストから救済したツールJSONを使用
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Rescuing %d tool call(s) from text\n", len(toolJSONTexts))
		}
		fmt.Fprintf(os.Stderr, "⚠️  FC rescue: %d tool call(s) extracted from text response\n", len(toolJSONTexts))
		for _, tj := range toolJSONTexts {
			fullResponse.WriteString(tj)
		}
	}

	// テキストもFunctionCallも救済ツールJSONもない場合のみエラー
	if fullResponse.Len() == 0 && len(functionCalls) == 0 {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] No content: textParts=%d, functionCalls=%d\n",
				len(textParts), len(functionCalls))
		}
		return "", fmt.Errorf("no content in Function Calling response (textParts=%d, functionCalls=%d, responses=%d)",
			len(textParts), len(functionCalls), len(responses))
	}

	fmt.Println()
	return fullResponse.String(), nil
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
