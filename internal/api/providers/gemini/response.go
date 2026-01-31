package gemini

import (
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

// handleFunctionCallingResponse は Function Calling レスポンスを処理
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

		for _, part := range candidate.Content.Parts {
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
					fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Text part: %q\n", part.Text[:min(len(part.Text), 100)])
				}
				textParts = append(textParts, part.Text)
			}
		}
	}

	// テキストパートを出力（ツール呼び出しJSONは除外）
	for _, text := range textParts {
		// テキストがツール呼び出しJSONの場合はスキップ
		trimmed := strings.TrimSpace(text)
		if strings.HasPrefix(trimmed, "{\"tool\"") || strings.HasPrefix(trimmed, "{ \"tool\"") {
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Skipping text (tool JSON): %s\n", trimmed[:min(len(trimmed), 50)])
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
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Skipping duplicate: %s\n", toolJSON)
			}
			continue
		}
		seenTools[toolJSON] = true
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

	// Usage callback: 最後のレスポンスからusage情報を取得
	if p.usageCallback != nil && len(responses) > 0 {
		lastResp := responses[len(responses)-1]
		if lastResp.UsageMetadata != nil {
			p.usageCallback(api.Usage{
				InputTokens:  lastResp.UsageMetadata.PromptTokenCount,
				OutputTokens: lastResp.UsageMetadata.CandidatesTokenCount,
			})
		}
	}

	fmt.Println()
	return fullResponse.String(), nil
}

// handleStreamingResponse はストリーミングレスポンスを処理
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	var fullResponse strings.Builder

	// 全レスポンスを読み込み
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// JSON配列としてパース
	var responses []GeminiResponse
	if err := json.Unmarshal(body, &responses); err != nil {
		spinner.Stop()

		// 配列でない場合は単一オブジェクトとして試す
		var singleResp GeminiResponse
		if err := json.Unmarshal(body, &singleResp); err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}
		responses = []GeminiResponse{singleResp}
	}

	spinner.Stop()

	// 各レスポンスからテキストを抽出
	firstChunk := true
	for _, geminiResp := range responses {
		if len(geminiResp.Candidates) > 0 {
			candidate := geminiResp.Candidates[0]
			if len(candidate.Content.Parts) > 0 {
				content := candidate.Content.Parts[0].Text

				if firstChunk && content != "" {
					firstChunk = false
				}

				fmt.Print(content)
				fullResponse.WriteString(content)
			}
		}
	}

	// Usage callback: 最後のレスポンスからusage情報を取得
	if p.usageCallback != nil && len(responses) > 0 {
		lastResp := responses[len(responses)-1]
		if lastResp.UsageMetadata != nil {
			p.usageCallback(api.Usage{
				InputTokens:  lastResp.UsageMetadata.PromptTokenCount,
				OutputTokens: lastResp.UsageMetadata.CandidatesTokenCount,
			})
		}
	}

	fmt.Println()
	return fullResponse.String(), nil
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（フォールバック）
func (p *Provider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	var result GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		spinner.Stop()
		return "", err
	}

	spinner.Stop()

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	// Usage callback
	if p.usageCallback != nil && result.UsageMetadata != nil {
		p.usageCallback(api.Usage{
			InputTokens:  result.UsageMetadata.PromptTokenCount,
			OutputTokens: result.UsageMetadata.CandidatesTokenCount,
		})
	}

	content := result.Candidates[0].Content.Parts[0].Text
	fmt.Println(content)
	return content, nil
}
