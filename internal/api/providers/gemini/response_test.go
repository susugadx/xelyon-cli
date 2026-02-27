package gemini

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// ===== handleFunctionCallingResponse unit tests =====

func TestHandleFunctionCallingResponse_SingleObject(t *testing.T) {
	// 単一オブジェクトレスポンス（配列でない）
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: "Here is the file content."},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if !strings.Contains(result, "Here is the file content.") {
		t.Errorf("result = %q, want to contain 'Here is the file content.'", result)
	}
}

func TestHandleFunctionCallingResponse_ArrayResponse(t *testing.T) {
	// 配列レスポンス（複数チャンク）
	responses := []GeminiFunctionResponse{
		{
			Candidates: []GeminiFunctionCandidate{
				{
					Content: GeminiFunctionContent{
						Parts: []GeminiFunctionPart{
							{Text: "Part 1. "},
						},
					},
				},
			},
		},
		{
			Candidates: []GeminiFunctionCandidate{
				{
					Content: GeminiFunctionContent{
						Parts: []GeminiFunctionPart{
							{Text: "Part 2."},
						},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(responses)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if !strings.Contains(result, "Part 1.") || !strings.Contains(result, "Part 2.") {
		t.Errorf("result = %q, want to contain both 'Part 1.' and 'Part 2.'", result)
	}
}

func TestHandleFunctionCallingResponse_FunctionCallOnly(t *testing.T) {
	// FunctionCall のみ（テキストなし）
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{FunctionCall: &api.GeminiFunctionCall{
							Name: "read_file",
							Args: map[string]any{"path": "/test/file.txt"},
						}},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if !strings.Contains(result, "read_file") {
		t.Errorf("result = %q, want to contain 'read_file'", result)
	}
	if !strings.Contains(result, "/test/file.txt") {
		t.Errorf("result = %q, want to contain '/test/file.txt'", result)
	}
}

func TestHandleFunctionCallingResponse_MixedTextAndFunctionCall(t *testing.T) {
	// Text + FunctionCall 混在
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: "I'll read the file for you."},
						{FunctionCall: &api.GeminiFunctionCall{
							Name: "read_file",
							Args: map[string]any{"path": "/src/main.go"},
						}},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if !strings.Contains(result, "I'll read the file for you.") {
		t.Errorf("result should contain text part, got %q", result)
	}
	if !strings.Contains(result, "read_file") {
		t.Errorf("result should contain function call, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_EmptyResponse(t *testing.T) {
	// 空のレスポンス（candidates なし）
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	_, err := p.handleFunctionCallingResponse(body, nil)
	if err == nil {
		t.Error("handleFunctionCallingResponse() should return error for empty response")
	}
	if !strings.Contains(err.Error(), "no content") {
		t.Errorf("error = %v, want 'no content' error", err)
	}
	// 件数情報がエラーメッセージに含まれることを確認
	if !strings.Contains(err.Error(), "textParts=") {
		t.Errorf("error = %v, want to contain count info for debugging", err)
	}
}

func TestHandleFunctionCallingResponse_NoCandidates(t *testing.T) {
	// candidates フィールドが nil
	body := []byte(`{"candidates":null}`)

	p := New("test-key")
	_, err := p.handleFunctionCallingResponse(body, nil)
	if err == nil {
		t.Error("handleFunctionCallingResponse() should return error for nil candidates")
	}
}

func TestHandleFunctionCallingResponse_ThinkingPartsSkipped(t *testing.T) {
	// Thinking パート（thought=true）はスキップされる
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: "Let me think about this...", Thought: true},
						{Text: "Here is my answer."},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if strings.Contains(result, "Let me think") {
		t.Error("result should NOT contain thinking part")
	}
	if !strings.Contains(result, "Here is my answer.") {
		t.Errorf("result should contain non-thinking part, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_DuplicateFunctionCallDedup(t *testing.T) {
	// 同一 FunctionCall の重複排除
	fc := &api.GeminiFunctionCall{
		Name: "bash",
		Args: map[string]any{"command": "ls -la"},
	}
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{FunctionCall: fc},
						{FunctionCall: fc}, // 重複
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}

	// "bash" が1回だけ含まれることを確認
	count := strings.Count(result, `"tool":"bash"`)
	if count != 1 {
		t.Errorf("expected 1 occurrence of bash tool call, got %d in result: %q", count, result)
	}
}

func TestHandleFunctionCallingResponse_UsageMetadataCallback(t *testing.T) {
	// UsageMetadata のコールバックが呼ばれることを確認
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: "Response with usage."},
					},
				},
			},
		},
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:     100,
			CandidatesTokenCount: 50,
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	var receivedUsage *api.Usage
	p.usageCallback = func(u api.Usage) {
		receivedUsage = &u
	}

	_, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}

	if receivedUsage == nil {
		t.Fatal("usageCallback should have been called")
	}
	if receivedUsage.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", receivedUsage.InputTokens)
	}
	if receivedUsage.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", receivedUsage.OutputTokens)
	}
}

func TestHandleFunctionCallingResponse_NoUsageCallbackNoPanic(t *testing.T) {
	// usageCallback が未設定でもパニックしない
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: "No callback set."},
					},
				},
			},
		},
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:     100,
			CandidatesTokenCount: 50,
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	// usageCallback は nil のまま

	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if !strings.Contains(result, "No callback set.") {
		t.Errorf("result = %q, want to contain 'No callback set.'", result)
	}
}

func TestHandleFunctionCallingResponse_InvalidJSON(t *testing.T) {
	// 不正な JSON
	body := []byte(`{invalid json}`)

	p := New("test-key")
	_, err := p.handleFunctionCallingResponse(body, nil)
	if err == nil {
		t.Error("handleFunctionCallingResponse() should return error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("error = %v, want 'failed to parse' error", err)
	}
	// body preview がエラーメッセージに含まれることを確認
	if !strings.Contains(err.Error(), "body preview") {
		t.Errorf("error = %v, want to contain 'body preview' for debugging", err)
	}
	if !strings.Contains(err.Error(), "invalid json") {
		t.Errorf("error = %v, want to contain actual body content", err)
	}
}

func TestHandleFunctionCallingResponse_ToolJSONTextSkippedWhenFCExists(t *testing.T) {
	// 実FC + ツールJSONテキスト → テキスト側はスキップ（重複防止）
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: `{"tool":"read_file","args":{"path":"/test"}}`},
						{Text: "Normal text response."},
						{FunctionCall: &api.GeminiFunctionCall{
							Name: "bash",
							Args: map[string]any{"command": "ls"},
						}},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if !strings.Contains(result, "Normal text response.") {
		t.Errorf("result should contain normal text, got %q", result)
	}
	if !strings.Contains(result, `"tool":"bash"`) {
		t.Errorf("result should contain FC bash, got %q", result)
	}
	// ツールJSONテキストは実FCがあるのでスキップされ、bash FC のみが含まれる
	if strings.Count(result, `"tool":"bash"`) != 1 {
		t.Errorf("expected exactly 1 bash FC, got result: %q", result)
	}
}

func TestHandleFunctionCallingResponse_ToolJSONTextRescuedWhenNoFC(t *testing.T) {
	// FC が空、テキストにツールJSON → レスキューされる
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: `{"tool":"read_file","args":{"path":"/src/main.go"}}`},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if !strings.Contains(result, "read_file") {
		t.Errorf("rescued tool JSON should contain read_file, got %q", result)
	}
	if !strings.Contains(result, "/src/main.go") {
		t.Errorf("rescued tool JSON should contain path, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_CodeBlockToolJSONRescued(t *testing.T) {
	// FC が空、テキストにコードブロック内ツールJSON → レスキューされる
	textWithCodeBlock := "I'll read the file for you.\n\n```json\n{\"tool\":\"read_file\",\"args\":{\"path\":\"/file.go\"}}\n```\n"
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: textWithCodeBlock},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if !strings.Contains(result, "read_file") {
		t.Errorf("rescued code block tool JSON should contain read_file, got %q", result)
	}
	if !strings.Contains(result, "I'll read the file") {
		t.Errorf("result should still contain normal text, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_MultipleToolJSONRescued(t *testing.T) {
	// FC が空、複数のツールJSONテキスト → 全てレスキュー
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: `{"tool":"read_file","args":{"path":"/a.go"}}`},
						{Text: `{"tool":"bash","args":{"command":"go test"}}`},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if !strings.Contains(result, "read_file") {
		t.Errorf("result should contain read_file, got %q", result)
	}
	if !strings.Contains(result, "bash") {
		t.Errorf("result should contain bash, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_MultipleFunctionCalls(t *testing.T) {
	// 複数の異なる FunctionCall
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{FunctionCall: &api.GeminiFunctionCall{
							Name: "read_file",
							Args: map[string]any{"path": "/file1.go"},
						}},
						{FunctionCall: &api.GeminiFunctionCall{
							Name: "bash",
							Args: map[string]any{"command": "go test ./..."},
						}},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if !strings.Contains(result, "read_file") {
		t.Errorf("result should contain read_file, got %q", result)
	}
	if !strings.Contains(result, "bash") {
		t.Errorf("result should contain bash, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_ThoughtSignatureIgnored(t *testing.T) {
	// thoughtSignature 付きパート（thought=true）はスキップ
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: "encrypted thinking", Thought: true, ThoughtSignature: "abc123"},
						{Text: "Actual response."},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if strings.Contains(result, "encrypted thinking") {
		t.Error("result should NOT contain thought with signature")
	}
	if !strings.Contains(result, "Actual response.") {
		t.Errorf("result should contain actual response, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_ThoughtSignatureOnlyPart(t *testing.T) {
	// thoughtSignature のみ（thought=false）のパートはテキストが空なので出力に影響しない
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{ThoughtSignature: "sig-only-no-thought-flag"},
						{Text: "Normal response."},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if strings.Contains(result, "sig-only") {
		t.Error("result should NOT contain thoughtSignature value")
	}
	if !strings.Contains(result, "Normal response.") {
		t.Errorf("result should contain normal response, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_ThoughtSignatureCycling(t *testing.T) {
	// 複数ターンの思考シグネチャが正しくスキップされ、テキスト/FCのみ出力される
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: "thinking step 1...", Thought: true, ThoughtSignature: "sig-turn1-aaa"},
						{Text: "thinking step 2...", Thought: true, ThoughtSignature: "sig-turn1-bbb"},
						{ThoughtSignature: "sig-turn1-final"},
						{Text: "I'll read the file.", FunctionCall: nil},
						{FunctionCall: &api.GeminiFunctionCall{
							Name: "read_file",
							Args: map[string]interface{}{"path": "/src/main.go"},
						}},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}

	// thinking テキストが表示テキストとして出力されないこと
	// （thought_parts メタデータ内に含まれるのは正常）
	displayText := result
	if idx := strings.Index(displayText, `{"tool"`); idx >= 0 {
		displayText = displayText[:idx]
	}
	if strings.Contains(displayText, "thinking step") {
		t.Errorf("display text should NOT contain thinking text, got %q", displayText)
	}
	// 実際のテキストとFCが含まれること
	if !strings.Contains(result, "I'll read the file.") {
		t.Errorf("result should contain actual text, got %q", result)
	}
	if !strings.Contains(result, "read_file") {
		t.Errorf("result should contain function call, got %q", result)
	}
	// thought_parts がツールJSON内に含まれること
	if !strings.Contains(result, "thought_parts") {
		t.Errorf("result should contain thought_parts metadata, got %q", result)
	}
	if !strings.Contains(result, "sig-turn1-aaa") {
		t.Errorf("result should contain thought signature in thought_parts, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_EmptyBody(t *testing.T) {
	// 空のボディ
	p := New("test-key")
	_, err := p.handleFunctionCallingResponse([]byte{}, nil)
	if err == nil {
		t.Error("handleFunctionCallingResponse() should return error for empty body")
	}
}

// ===== extractCodeBlockToolJSON unit tests =====

func TestExtractCodeBlockToolJSON_Basic(t *testing.T) {
	text := "Some text.\n\n```json\n{\"tool\":\"read_file\",\"args\":{\"path\":\"/file\"}}\n```\n"
	toolJSONs, remaining := extractCodeBlockToolJSON(text)

	if len(toolJSONs) != 1 {
		t.Fatalf("expected 1 tool JSON, got %d", len(toolJSONs))
	}
	if !strings.Contains(toolJSONs[0], "read_file") {
		t.Errorf("toolJSON should contain read_file, got %q", toolJSONs[0])
	}
	if strings.Contains(remaining, "```") {
		t.Errorf("remaining should not contain code block markers, got %q", remaining)
	}
	if !strings.Contains(remaining, "Some text.") {
		t.Errorf("remaining should contain surrounding text, got %q", remaining)
	}
}

func TestExtractCodeBlockToolJSON_NotToolJSON(t *testing.T) {
	text := "```go\nfunc main() {}\n```\n"
	toolJSONs, remaining := extractCodeBlockToolJSON(text)

	if len(toolJSONs) != 0 {
		t.Errorf("expected 0 tool JSONs for non-tool code block, got %d", len(toolJSONs))
	}
	if remaining != text {
		t.Errorf("remaining should be unchanged, got %q", remaining)
	}
}

func TestExtractCodeBlockToolJSON_NoCodeBlock(t *testing.T) {
	text := "Just plain text."
	toolJSONs, remaining := extractCodeBlockToolJSON(text)

	if len(toolJSONs) != 0 {
		t.Errorf("expected 0 tool JSONs, got %d", len(toolJSONs))
	}
	if remaining != text {
		t.Errorf("remaining should be unchanged, got %q", remaining)
	}
}

func TestExtractCodeBlockToolJSON_Multiple(t *testing.T) {
	text := "First\n```json\n{\"tool\":\"read_file\",\"args\":{}}\n```\nMiddle\n```json\n{\"tool\":\"bash\",\"args\":{}}\n```\nLast"
	toolJSONs, remaining := extractCodeBlockToolJSON(text)

	if len(toolJSONs) != 2 {
		t.Fatalf("expected 2 tool JSONs, got %d", len(toolJSONs))
	}
	if !strings.Contains(remaining, "First") || !strings.Contains(remaining, "Last") {
		t.Errorf("remaining should contain surrounding text, got %q", remaining)
	}
}

func TestIsToolJSONPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{`{"tool":"read_file","args":{}}`, true},
		{`{ "tool": "bash", "args": {} }`, true},
		{`{"id":"call_1","tool":"read_file"}`, false},
		{`Just text`, false},
		{``, false},
	}
	for _, tt := range tests {
		got := isToolJSONPrefix(tt.input)
		if got != tt.want {
			t.Errorf("isToolJSONPrefix(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestHandleFunctionCallingResponse_SignatureWithTextSuppressed(t *testing.T) {
	// Gemini 3.1 Pro バグ修正: thought=false, thoughtSignature 付き, text 付きのパートが
	// 生テキストとして出力に漏れないことを確認
	longSig := "dGhpcyBpcyBhIGJhc2U2NCBlbmNvZGVkIHRob3VnaHQgc2lnbmF0dXJl" // Base64風の署名
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						// thought=false だが signature + text を持つパート（バグの原因）
						{Text: "leaked thinking content", ThoughtSignature: longSig},
						{Text: "Actual visible response."},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	// 署名+テキストパートのテキストが出力に漏れないこと
	if strings.Contains(result, "leaked thinking content") {
		t.Error("result should NOT contain text from signature+text part (thought=false)")
	}
	// 通常テキストは表示されること
	if !strings.Contains(result, "Actual visible response.") {
		t.Errorf("result should contain actual response, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_SignatureWithTextPreservedInThoughtParts(t *testing.T) {
	// 署名+テキストパートが thoughtParts に収集され、FC の thought_parts に含まれることを確認
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: "thinking internally...", Thought: true, ThoughtSignature: "sig-step1"},
						{Text: "signature-attached text", ThoughtSignature: "sig-final-abc"}, // thought=false
						{FunctionCall: &api.GeminiFunctionCall{
							Name: "read_file",
							Args: map[string]any{"path": "/main.go"},
						}},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	// 署名テキストが表示テキストに漏れないこと
	displayText := result
	if idx := strings.Index(displayText, `{"tool"`); idx >= 0 {
		displayText = displayText[:idx]
	}
	if strings.Contains(displayText, "signature-attached text") {
		t.Error("signature+text part should NOT appear in display text")
	}
	// thought_parts メタデータに署名が含まれること
	if !strings.Contains(result, "thought_parts") {
		t.Errorf("result should contain thought_parts, got %q", result)
	}
	if !strings.Contains(result, "sig-final-abc") {
		t.Errorf("result should contain signature in thought_parts, got %q", result)
	}
	// FC は正常に含まれること
	if !strings.Contains(result, "read_file") {
		t.Errorf("result should contain function call, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_SignatureWithFCCollected(t *testing.T) {
	// thoughtSignature + FunctionCall のパートは:
	// - FC は正常に収集される
	// - テキストは抑制される（signature フィルタが先に発動）
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{
							FunctionCall: &api.GeminiFunctionCall{
								Name: "bash",
								Args: map[string]any{"command": "ls"},
							},
							ThoughtSignature: "sig-with-fc",
						},
						{Text: "Normal text."},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	// FC が正常に含まれること（署名付きでも FC は収集される）
	if !strings.Contains(result, "bash") {
		t.Errorf("result should contain bash FC, got %q", result)
	}
	// 通常テキスト（別パート）は含まれること
	if !strings.Contains(result, "Normal text.") {
		t.Errorf("result should contain normal text, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_SignatureWithFCAndTextSuppressed(t *testing.T) {
	// ThoughtSignature + FunctionCall + Text を同時に持つパート:
	// - FC は収集される
	// - Text は表示テキストに漏れない（signature フィルタで抑制）
	// - Text は thought_parts に保存される
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{
							Text: "leaked thinking from signature+FC part",
							FunctionCall: &api.GeminiFunctionCall{
								Name: "read_files",
								Args: map[string]any{"paths": "/src/main.go"},
							},
							ThoughtSignature: "zM+8H4uKsignatureblob123456789",
						},
						{Text: "Visible explanation."},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}

	// FC は正常に含まれること
	if !strings.Contains(result, "read_files") {
		t.Errorf("result should contain read_files FC, got %q", result)
	}
	// 通常テキスト（別パート）は含まれること
	if !strings.Contains(result, "Visible explanation.") {
		t.Errorf("result should contain visible text, got %q", result)
	}
	// signature+FC パートのテキストが表示テキストに漏れないこと
	displayText := result
	if idx := strings.Index(displayText, `{"tool"`); idx >= 0 {
		displayText = displayText[:idx]
	}
	if strings.Contains(displayText, "leaked thinking from signature+FC part") {
		t.Error("text from signature+FC part should NOT appear in display text")
	}
	// thought_signature がツールJSON内の thought_parts に含まれること
	if !strings.Contains(result, "thought_parts") {
		t.Errorf("result should contain thought_parts, got %q", result)
	}
	if !strings.Contains(result, "zM+8H4uKsignatureblob123456789") {
		t.Errorf("result should contain the signature in thought_parts, got %q", result)
	}
}

func TestHandleFunctionCallingResponse_SignatureOnlyFCNoText(t *testing.T) {
	// ThoughtSignature + FunctionCall だがテキストなし → FC だけ収集される
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{
							FunctionCall: &api.GeminiFunctionCall{
								Name: "bash",
								Args: map[string]any{"command": "go test"},
							},
							ThoughtSignature: "sig-no-text",
						},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	result, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if !strings.Contains(result, "bash") {
		t.Errorf("result should contain bash FC, got %q", result)
	}
	if !strings.Contains(result, "go test") {
		t.Errorf("result should contain command args, got %q", result)
	}
}

// ===== updateToolJSONDepth unit tests =====

func TestUpdateToolJSONDepth_SimpleObject(t *testing.T) {
	depth := 0
	inStr := false
	updateToolJSONDepth(`{"tool":"read_file"}`, &depth, &inStr)
	if depth != 0 {
		t.Errorf("depth = %d, want 0 (balanced braces)", depth)
	}
	if inStr {
		t.Error("inStr should be false after balanced JSON")
	}
}

func TestUpdateToolJSONDepth_NestedObject(t *testing.T) {
	depth := 0
	inStr := false
	updateToolJSONDepth(`{"tool":"bash","args":{"command":"ls"}}`, &depth, &inStr)
	if depth != 0 {
		t.Errorf("depth = %d, want 0 (balanced nested braces)", depth)
	}
}

func TestUpdateToolJSONDepth_PartialFirstChunk(t *testing.T) {
	// チャンク1: `{"tool":"read` → depth=1 (開いたまま)
	depth := 0
	inStr := false
	updateToolJSONDepth(`{"tool":"read`, &depth, &inStr)
	if depth != 1 {
		t.Errorf("depth = %d, want 1 (unclosed brace)", depth)
	}
	if !inStr {
		t.Error("inStr should be true (inside unclosed string)")
	}
}

func TestUpdateToolJSONDepth_PartialSecondChunk(t *testing.T) {
	// チャンク1 → チャンク2 で閉じる
	depth := 1
	inStr := true
	updateToolJSONDepth(`_files","args":{"path":"/main.go"}}`, &depth, &inStr)
	if depth != 0 {
		t.Errorf("depth = %d, want 0 (closed by second chunk)", depth)
	}
	if inStr {
		t.Error("inStr should be false after balanced JSON")
	}
}

func TestUpdateToolJSONDepth_BracesInString(t *testing.T) {
	// 文字列リテラル内の {} は深度に影響しない
	depth := 0
	inStr := false
	updateToolJSONDepth(`{"content":"value with { and } inside"}`, &depth, &inStr)
	if depth != 0 {
		t.Errorf("depth = %d, want 0 (braces in string should be ignored)", depth)
	}
}

func TestUpdateToolJSONDepth_EscapedQuotes(t *testing.T) {
	// エスケープされた引用符は文字列の終端にならない
	depth := 0
	inStr := false
	updateToolJSONDepth(`{"content":"say \"hello\" world"}`, &depth, &inStr)
	if depth != 0 {
		t.Errorf("depth = %d, want 0", depth)
	}
	if inStr {
		t.Error("inStr should be false after balanced JSON with escaped quotes")
	}
}

func TestUpdateToolJSONDepth_ThoughtSignatureChunk(t *testing.T) {
	// thought_signature を含む巨大チャンク
	depth := 1
	inStr := true
	sig := strings.Repeat("A", 10000) // 巨大な署名
	chunk := fmt.Sprintf(`_files","args":{"path":"/file"},"thought_signature":"%s"}`, sig)
	updateToolJSONDepth(chunk, &depth, &inStr)
	if depth != 0 {
		t.Errorf("depth = %d, want 0 (should close after signature)", depth)
	}
}

func TestUpdateToolJSONDepth_EmptyString(t *testing.T) {
	depth := 5
	inStr := true
	updateToolJSONDepth("", &depth, &inStr)
	if depth != 5 {
		t.Errorf("depth = %d, want 5 (unchanged for empty string)", depth)
	}
	if !inStr {
		t.Error("inStr should remain true for empty string")
	}
}

// ===== ThinkingTimeout config tests =====

func TestThinkingTimeoutDefaults(t *testing.T) {
	// config のデフォルト値が正しいことを確認
	cfg := config.DefaultConfig()
	if cfg.Streaming.ThinkingTimeoutSeconds != 120 {
		t.Errorf("ThinkingTimeoutSeconds default = %d, want 120", cfg.Streaming.ThinkingTimeoutSeconds)
	}
	if cfg.Streaming.IdleTimeoutSeconds != 300 {
		t.Errorf("IdleTimeoutSeconds default = %d, want 300", cfg.Streaming.IdleTimeoutSeconds)
	}
}

func TestHandleFunctionCallingResponse_CachedTokens(t *testing.T) {
	// CachedContentTokenCount が正しくコールバックされる
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: "Cached response."},
					},
				},
			},
		},
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:        200,
			CandidatesTokenCount:    75,
			CachedContentTokenCount: 150,
		},
	}
	body, _ := json.Marshal(resp)

	p := New("test-key")
	var receivedUsage *api.Usage
	p.usageCallback = func(u api.Usage) {
		receivedUsage = &u
	}

	_, err := p.handleFunctionCallingResponse(body, nil)
	if err != nil {
		t.Fatalf("handleFunctionCallingResponse() error = %v", err)
	}
	if receivedUsage == nil {
		t.Fatal("usageCallback should have been called")
	}
	if receivedUsage.CachedInputTokens != 150 {
		t.Errorf("CachedInputTokens = %d, want 150", receivedUsage.CachedInputTokens)
	}
}
