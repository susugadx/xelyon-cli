package gemini

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
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

	scanner := bufio.NewScanner(resp.Body)
	firstChunk := true

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		// スピナーを停止（最初のデータが来たら）
		if firstChunk && spinner != nil {
			spinner.Stop()
			firstChunk = false
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

			// thoughtSignature のみのパート（thought=false, text="", functionCall=nil）を収集
			if part.ThoughtSignature != "" && part.FunctionCall == nil && part.Text == "" {
				tp := map[string]any{"thought_signature": part.ThoughtSignature}
				thoughtParts = append(thoughtParts, tp)
				if debug {
					sig := part.ThoughtSignature
					if len(sig) > 20 {
						sig = sig[:20] + "..."
					}
					fmt.Fprintf(os.Stderr, "[DEBUG Gemini SSE] Collected signature-only part (sig=%q)\n", sig)
				}
				continue
			}

			if part.Text != "" {
				trimmed := strings.TrimSpace(part.Text)
				// ツールJSONプレフィックスの場合は表示せず fullResponse に記録のみ
				if isToolJSONPrefix(trimmed) {
					fullResponse.WriteString(part.Text)
					continue
				}
				// コードブロック内のツールJSON救済（ループ内で即時分離）
				extracted, remaining := extractCodeBlockToolJSON(part.Text)
				if len(extracted) > 0 {
					rescuedToolJSONs = append(rescuedToolJSONs, extracted...)
					if strings.TrimSpace(remaining) != "" {
						if !headerPrinted {
							api.PrintAIHeader()
							headerPrinted = true
						}
						fmt.Print(remaining)
					}
					fullResponse.WriteString(remaining)
					continue
				}
				// 通常テキスト
				if !headerPrinted {
					api.PrintAIHeader()
					headerPrinted = true
				}
				fmt.Print(part.Text)
				fullResponse.WriteString(part.Text)
			}

			if part.FunctionCall != nil {
				part.FunctionCall.ThoughtSignature = part.ThoughtSignature
				functionCalls = append(functionCalls, part.FunctionCall)
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

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("SSE scan error: %w", err)
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

	// FunctionCall を出力（重複排除）
	seenTools := make(map[string]bool)
	for _, fc := range functionCalls {
		toolJSON := convertFunctionCallToToolJSON(fc)
		if seenTools[toolJSON] {
			continue
		}
		seenTools[toolJSON] = true
		// 内部用は ThoughtSignature/ThoughtParts を含む完全 JSON
		fullResponse.WriteString(toolJSON)
	}

	if usage != nil && p.usageCallback != nil {
		p.usageCallback(api.Usage{
			InputTokens:       usage.PromptTokenCount,
			OutputTokens:      usage.CandidatesTokenCount,
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

			// thoughtSignature のみのパート（thought=false, text="", functionCall=nil）を収集
			if part.ThoughtSignature != "" && part.FunctionCall == nil && part.Text == "" {
				tp := map[string]any{"thought_signature": part.ThoughtSignature}
				thoughtParts = append(thoughtParts, tp)
				if debug {
					sig := part.ThoughtSignature
					if len(sig) > 20 {
						sig = sig[:20] + "..."
					}
					fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Collected signature-only part (sig=%q)\n", sig)
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

	// FunctionCall がある場合はそちらを出力（重複排除）
	seenTools := make(map[string]bool)
	if len(functionCalls) > 0 {
		for _, fc := range functionCalls {
			toolJSON := convertFunctionCallToToolJSON(fc)
			if seenTools[toolJSON] {
				continue
			}
			seenTools[toolJSON] = true
			// 内部用は ThoughtSignature/ThoughtParts を含む完全 JSON
			fullResponse.WriteString(toolJSON)
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
