package gemini

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
			if part.Thought {
				continue
			}

			if part.Text != "" {
				// ツール呼び出しJSONのチェック
				trimmed := strings.TrimSpace(part.Text)
				if strings.HasPrefix(trimmed, "{\"tool\"") || strings.HasPrefix(trimmed, "{ \"tool\"") {
					fullResponse.WriteString(part.Text)
					continue
				}
				fmt.Print(part.Text)
				fullResponse.WriteString(part.Text)
			}

			if part.FunctionCall != nil {
				functionCalls = append(functionCalls, part.FunctionCall)
			}
		}
	}

	if spinner != nil {
		spinner.Stop()
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("SSE scan error: %w", err)
	}

	// FunctionCall を出力（重複排除）
	seenTools := make(map[string]bool)
	for _, fc := range functionCalls {
		toolJSON := convertFunctionCallToToolJSON(fc)
		if seenTools[toolJSON] {
			continue
		}
		seenTools[toolJSON] = true
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
		return "", fmt.Errorf("no content in Gemini SSE response")
	}

	fmt.Println()
	return fullResponse.String(), nil
}

// handleStreamingResponse は互換性のために SSE パーサーを呼び出す
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	return p.handleSSEResponse(ctx, resp, spinner)
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
			return "", fmt.Errorf("failed to parse Function Calling response: %w", err)
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
			if part.Thought {
				continue
			}

			// Function Call パートを収集
			if part.FunctionCall != nil {
				if debug {
					fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Found FunctionCall: %s\n", part.FunctionCall.Name)
				}
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

	// テキストパートを出力（ツール呼び出しJSONは除外）
	for _, text := range textParts {
		trimmed := strings.TrimSpace(text)
		if strings.HasPrefix(trimmed, "{\"tool\"") || strings.HasPrefix(trimmed, "{ \"tool\"") {
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Skipping text (tool JSON)\n")
			}
			continue
		}
		fmt.Print(text)
		fullResponse.WriteString(text)
	}

	// FunctionCall を出力（重複排除）
	seenTools := make(map[string]bool)
	for _, fc := range functionCalls {
		toolJSON := convertFunctionCallToToolJSON(fc)
		if seenTools[toolJSON] {
			continue
		}
		seenTools[toolJSON] = true
		fmt.Printf("\n%s", toolJSON)
		fullResponse.WriteString(toolJSON)
	}

	// テキストもFunctionCallもない場合のみエラー
	if fullResponse.Len() == 0 && len(functionCalls) == 0 {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] No content: textParts=%d, functionCalls=%d\n",
				len(textParts), len(functionCalls))
		}
		return "", fmt.Errorf("no content in Function Calling response")
	}

	fmt.Println()
	return fullResponse.String(), nil
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（エラーメッセージ表示用などに残す）
func (p *Provider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	spinner.Stop()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}
	content := geminiResp.Candidates[0].Content.Parts[0].Text
	fmt.Println(content)
	return content, nil
}
