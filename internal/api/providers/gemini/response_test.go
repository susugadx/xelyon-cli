package gemini

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
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

func TestHandleFunctionCallingResponse_ToolJSONTextSkipped(t *testing.T) {
	// テキストパートがツール呼び出しJSON形式の場合はスキップされる
	resp := GeminiFunctionResponse{
		Candidates: []GeminiFunctionCandidate{
			{
				Content: GeminiFunctionContent{
					Parts: []GeminiFunctionPart{
						{Text: `{"tool":"read_file","args":{"path":"/test"}}`},
						{Text: "Normal text response."},
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

func TestHandleFunctionCallingResponse_EmptyBody(t *testing.T) {
	// 空のボディ
	p := New("test-key")
	_, err := p.handleFunctionCallingResponse([]byte{}, nil)
	if err == nil {
		t.Error("handleFunctionCallingResponse() should return error for empty body")
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
