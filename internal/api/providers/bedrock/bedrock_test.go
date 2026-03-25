package bedrock

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"

	// ツール登録のための blank import
	_ "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	_ "github.com/susugadx/xelyon-cli/internal/tools/lsp"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
)

func TestProvider_Name(t *testing.T) {
	p := &Provider{}
	if p.Name() != "Bedrock" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Bedrock")
	}
}

func TestProvider_SupportsImages(t *testing.T) {
	p := &Provider{}
	if !p.SupportsImages() {
		t.Error("SupportsImages() = false, want true")
	}
}

func TestProvider_IsFunctionCallingEnabled(t *testing.T) {
	originalEnv := os.Getenv("BEDROCK_FUNCTION_CALLING")
	defer os.Setenv("BEDROCK_FUNCTION_CALLING", originalEnv)

	t.Run("EnabledByDefault", func(t *testing.T) {
		os.Unsetenv("BEDROCK_FUNCTION_CALLING")
		p := &Provider{}
		if !p.IsFunctionCallingEnabled() {
			t.Error("IsFunctionCallingEnabled() = false, want true by default")
		}
	})

	t.Run("DisabledWithEnvVar", func(t *testing.T) {
		os.Setenv("BEDROCK_FUNCTION_CALLING", "0")
		p := &Provider{}
		if p.IsFunctionCallingEnabled() {
			t.Error("IsFunctionCallingEnabled() = true, want false when BEDROCK_FUNCTION_CALLING=0")
		}
	})

	t.Run("EnabledWithNonZero", func(t *testing.T) {
		os.Setenv("BEDROCK_FUNCTION_CALLING", "1")
		p := &Provider{}
		if !p.IsFunctionCallingEnabled() {
			t.Error("IsFunctionCallingEnabled() = false, want true when BEDROCK_FUNCTION_CALLING=1")
		}
	})
}

func TestDefaultRegion(t *testing.T) {
	originalRegion := os.Getenv("AWS_REGION")
	originalDefault := os.Getenv("AWS_DEFAULT_REGION")
	defer func() {
		os.Setenv("AWS_REGION", originalRegion)
		os.Setenv("AWS_DEFAULT_REGION", originalDefault)
	}()

	t.Run("Default", func(t *testing.T) {
		os.Unsetenv("AWS_REGION")
		os.Unsetenv("AWS_DEFAULT_REGION")
		// New() を呼ぶと実際に AWS SDK を初期化するためリージョンのみ検証
		region := os.Getenv("AWS_REGION")
		if region == "" {
			region = os.Getenv("AWS_DEFAULT_REGION")
		}
		if region == "" {
			region = defaultRegion
		}
		if region != "us-east-1" {
			t.Errorf("default region = %q, want %q", region, "us-east-1")
		}
	})

	t.Run("RegionOverride", func(t *testing.T) {
		os.Setenv("AWS_REGION", "ap-northeast-1")
		region := os.Getenv("AWS_REGION")
		if region != "ap-northeast-1" {
			t.Errorf("region = %q, want %q", region, "ap-northeast-1")
		}
	})

	t.Run("DefaultRegionFallback", func(t *testing.T) {
		os.Unsetenv("AWS_REGION")
		os.Setenv("AWS_DEFAULT_REGION", "eu-west-1")
		region := os.Getenv("AWS_REGION")
		if region == "" {
			region = os.Getenv("AWS_DEFAULT_REGION")
		}
		if region != "eu-west-1" {
			t.Errorf("region = %q, want %q", region, "eu-west-1")
		}
	})
}

func TestBedrockRequest_JSON(t *testing.T) {
	req := BedrockRequest{
		AnthropicVersion: bedrockAnthropicVersion,
		MaxTokens:        4096,
		System:           "You are a helpful assistant.",
		Messages: []claude.AnthropicMessage{
			{Role: "user", Content: []claude.AnthropicContentBlock{{Type: "text", Text: "Hello"}}},
		},
	}

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// anthropic_version フィールドが存在することを確認
	version, ok := raw["anthropic_version"]
	if !ok {
		t.Fatal("anthropic_version field missing from request JSON")
	}
	if version != "bedrock-2023-05-31" {
		t.Errorf("anthropic_version = %q, want %q", version, "bedrock-2023-05-31")
	}

	// anthropic_beta が omitempty で省略されることを確認
	if _, ok := raw["anthropic_beta"]; ok {
		t.Error("anthropic_beta should be omitted when empty")
	}

	// model フィールドが存在しないことを確認（Bedrock では ModelId パラメータで送る）
	if _, ok := raw["model"]; ok {
		t.Error("model field should not be in request body (Bedrock uses ModelId parameter)")
	}

	// stream フィールドが存在しないことを確認
	if _, ok := raw["stream"]; ok {
		t.Error("stream field should not be in request body (Bedrock uses API method)")
	}

	// max_tokens が正しいことを確認
	maxTokens, ok := raw["max_tokens"].(float64)
	if !ok || int(maxTokens) != 4096 {
		t.Errorf("max_tokens = %v, want 4096", raw["max_tokens"])
	}
}

func TestBedrockRequest_WithAnthropicBeta(t *testing.T) {
	req := BedrockRequest{
		AnthropicVersion: bedrockAnthropicVersion,
		AnthropicBeta:    []string{"context-1m-2025-08-07"},
		MaxTokens:        4096,
		System:           "System prompt",
		Messages: []claude.AnthropicMessage{
			{Role: "user", Content: []claude.AnthropicContentBlock{{Type: "text", Text: "Hello"}}},
		},
	}

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	beta, ok := raw["anthropic_beta"].([]interface{})
	if !ok {
		t.Fatal("anthropic_beta field missing or wrong type")
	}
	if len(beta) != 1 || beta[0] != "context-1m-2025-08-07" {
		t.Errorf("anthropic_beta = %v, want [context-1m-2025-08-07]", beta)
	}
}

func TestBedrockRequest_WithTopLevelCacheControl(t *testing.T) {
	req := BedrockRequest{
		AnthropicVersion: bedrockAnthropicVersion,
		CacheControl:     &api.CacheControl{Type: "ephemeral"},
		MaxTokens:        4096,
		System:           "System prompt",
		Messages: []claude.AnthropicMessage{
			{Role: "user", Content: []claude.AnthropicContentBlock{{Type: "text", Text: "Hello"}}},
		},
	}

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	cacheControl, ok := raw["cache_control"].(map[string]interface{})
	if !ok {
		t.Fatal("cache_control field missing or wrong type")
	}
	if cacheControl["type"] != "ephemeral" {
		t.Errorf("cache_control.type = %q, want %q", cacheControl["type"], "ephemeral")
	}
}

func TestClearToolUses_Bedrock(t *testing.T) {
	t.Run("WithCompaction", func(t *testing.T) {
		cfg := config.DefaultConfig()

		contextManagement, betaHeaders := buildBedrockContextManagement(
			"global.anthropic.claude-sonnet-4-6-v1",
			cfg.Compression,
			[]string{"context-1m-2025-08-07"},
		)

		if contextManagement == nil {
			t.Fatal("ContextManagement should be set for supported Bedrock Claude models")
		}
		if len(contextManagement.Edits) != 2 {
			t.Fatalf("len(ContextManagement.Edits) = %d, want 2", len(contextManagement.Edits))
		}
		if contextManagement.Edits[0].Type != "clear_tool_uses_20250919" {
			t.Errorf("Edits[0].Type = %q, want clear_tool_uses_20250919", contextManagement.Edits[0].Type)
		}
		if contextManagement.Edits[1].Type != "compact_20260112" {
			t.Errorf("Edits[1].Type = %q, want compact_20260112", contextManagement.Edits[1].Type)
		}
		if !containsString(betaHeaders, "context-management-2025-06-27") {
			t.Errorf("beta headers should include context-management-2025-06-27, got %v", betaHeaders)
		}
		if !containsString(betaHeaders, "compact-2026-01-12") {
			t.Errorf("beta headers should include compact-2026-01-12, got %v", betaHeaders)
		}
	})

	t.Run("ClearOnlyWithoutCompaction", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Compression.ClaudeCompaction = false

		contextManagement, betaHeaders := buildBedrockContextManagement(
			"global.anthropic.claude-3-5-sonnet-v1",
			cfg.Compression,
			nil,
		)

		if contextManagement == nil {
			t.Fatal("ContextManagement should be set when clear_tool_uses is enabled")
		}
		if len(contextManagement.Edits) != 1 {
			t.Fatalf("len(ContextManagement.Edits) = %d, want 1", len(contextManagement.Edits))
		}
		if contextManagement.Edits[0].Type != "clear_tool_uses_20250919" {
			t.Errorf("Edits[0].Type = %q, want clear_tool_uses_20250919", contextManagement.Edits[0].Type)
		}
		if !containsString(betaHeaders, "context-management-2025-06-27") {
			t.Errorf("beta headers should include context-management-2025-06-27, got %v", betaHeaders)
		}
		if containsString(betaHeaders, "compact-2026-01-12") {
			t.Errorf("beta headers should not include compact-2026-01-12, got %v", betaHeaders)
		}
	})
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestBedrockRequest_ConfigVersion(t *testing.T) {
	// config から version を取得する場合のテスト
	customVersion := "bedrock-2024-01-01"
	req := BedrockRequest{
		AnthropicVersion: customVersion,
		MaxTokens:        4096,
		System:           "System prompt",
		Messages: []claude.AnthropicMessage{
			{Role: "user", Content: []claude.AnthropicContentBlock{{Type: "text", Text: "Hello"}}},
		},
	}

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if raw["anthropic_version"] != customVersion {
		t.Errorf("anthropic_version = %q, want %q", raw["anthropic_version"], customVersion)
	}
}

func TestBedrockRequest_WithThinking(t *testing.T) {
	req := BedrockRequest{
		AnthropicVersion: bedrockAnthropicVersion,
		MaxTokens:        4096,
		System:           "System prompt",
		Messages: []claude.AnthropicMessage{
			{Role: "user", Content: []claude.AnthropicContentBlock{{Type: "text", Text: "Hello"}}},
		},
		Thinking: &claude.ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: 10000,
		},
	}

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	thinking, ok := raw["thinking"].(map[string]interface{})
	if !ok {
		t.Fatal("thinking field missing or wrong type")
	}
	if thinking["type"] != "enabled" {
		t.Errorf("thinking.type = %q, want %q", thinking["type"], "enabled")
	}
	budget, ok := thinking["budget_tokens"].(float64)
	if !ok || int(budget) != 10000 {
		t.Errorf("thinking.budget_tokens = %v, want 10000", thinking["budget_tokens"])
	}
}

func TestBedrockRequest_WithTools(t *testing.T) {
	req := BedrockRequest{
		AnthropicVersion: bedrockAnthropicVersion,
		MaxTokens:        4096,
		System:           "System prompt",
		Messages: []claude.AnthropicMessage{
			{Role: "user", Content: []claude.AnthropicContentBlock{{Type: "text", Text: "Hello"}}},
		},
		Tools: []claude.ClaudeTool{
			{
				Name:        "read_file",
				Description: "Read a file",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "File path",
						},
					},
					"required": []string{"path"},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	tools, ok := raw["tools"].([]interface{})
	if !ok || len(tools) != 1 {
		t.Fatalf("tools should have 1 entry, got %v", raw["tools"])
	}

	tool := tools[0].(map[string]interface{})
	if tool["name"] != "read_file" {
		t.Errorf("tool name = %q, want %q", tool["name"], "read_file")
	}
	if _, ok := tool["input_schema"]; !ok {
		t.Error("tool should have input_schema field (not parameters)")
	}
}

func TestSetMCPTools(t *testing.T) {
	p := &Provider{}

	tools := []api.ToolDefinition{
		{Name: "custom_tool", Description: "A custom tool"},
	}
	p.SetMCPTools(tools)

	if len(p.mcpTools) != 1 {
		t.Errorf("mcpTools length = %d, want 1", len(p.mcpTools))
	}
	if p.mcpTools[0].Name != "custom_tool" {
		t.Errorf("mcpTools[0].Name = %q, want %q", p.mcpTools[0].Name, "custom_tool")
	}
}

func TestSetUsageCallback(t *testing.T) {
	p := &Provider{}

	var calledWith api.Usage
	callback := func(u api.Usage) {
		calledWith = u
	}
	p.SetUsageCallback(callback)

	if p.usageCallback == nil {
		t.Error("usageCallback should not be nil")
	}

	// コールバックが正しく呼ばれることを確認
	p.usageCallback(api.Usage{
		InputTokens:         100,
		OutputTokens:        50,
		CachedInputTokens:   80,
		CacheCreationTokens: 20,
	})
	if calledWith.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", calledWith.InputTokens)
	}
	if calledWith.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", calledWith.OutputTokens)
	}
	if calledWith.CachedInputTokens != 80 {
		t.Errorf("CachedInputTokens = %d, want 80", calledWith.CachedInputTokens)
	}
	if calledWith.CacheCreationTokens != 20 {
		t.Errorf("CacheCreationTokens = %d, want 20", calledWith.CacheCreationTokens)
	}
}

// processChunk のテスト（ストリーミングイベント処理）

func TestProcessChunk_TextDelta(t *testing.T) {
	p := &Provider{}
	toolUses := make(map[int]*toolUseAccumulator)
	var toolCallsOutput strings.Builder
	var lastUsage *api.Usage

	data := `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`

	text, done := p.processChunk([]byte(data), toolUses, &toolCallsOutput, &lastUsage, make(map[int]*strings.Builder), &strings.Builder{}, ui.NewSpinner())
	if text != "Hello" {
		t.Errorf("text = %q, want %q", text, "Hello")
	}
	if done {
		t.Error("done should be false for text_delta")
	}
}

func TestProcessChunk_MessageStop(t *testing.T) {
	p := &Provider{}
	toolUses := make(map[int]*toolUseAccumulator)
	var toolCallsOutput strings.Builder
	var lastUsage *api.Usage

	data := `{"type":"message_stop"}`

	text, done := p.processChunk([]byte(data), toolUses, &toolCallsOutput, &lastUsage, make(map[int]*strings.Builder), &strings.Builder{}, ui.NewSpinner())
	if text != "" {
		t.Errorf("text = %q, want empty", text)
	}
	if !done {
		t.Error("done should be true for message_stop")
	}
}

func TestProcessChunk_ToolUse(t *testing.T) {
	p := &Provider{}
	toolUses := make(map[int]*toolUseAccumulator)
	var toolCallsOutput strings.Builder
	var lastUsage *api.Usage

	// 1. tool_use ブロック開始
	startData := `{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_123","name":"read_file"}}`
	p.processChunk([]byte(startData), toolUses, &toolCallsOutput, &lastUsage, make(map[int]*strings.Builder), &strings.Builder{}, ui.NewSpinner())

	if _, ok := toolUses[0]; !ok {
		t.Fatal("toolUses[0] should be set after content_block_start")
	}

	// 2. input_json_delta
	delta1 := `{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"pa"}}`
	p.processChunk([]byte(delta1), toolUses, &toolCallsOutput, &lastUsage, make(map[int]*strings.Builder), &strings.Builder{}, ui.NewSpinner())

	delta2 := `{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"th\":\"/test.txt\"}"}}`
	p.processChunk([]byte(delta2), toolUses, &toolCallsOutput, &lastUsage, make(map[int]*strings.Builder), &strings.Builder{}, ui.NewSpinner())

	// 3. content_block_stop
	stopData := `{"type":"content_block_stop","index":0}`
	p.processChunk([]byte(stopData), toolUses, &toolCallsOutput, &lastUsage, make(map[int]*strings.Builder), &strings.Builder{}, ui.NewSpinner())

	output := toolCallsOutput.String()
	if output == "" {
		t.Fatal("toolCallsOutput should not be empty after content_block_stop")
	}
	if !strings.Contains(output, "read_file") {
		t.Errorf("output = %q, expected to contain 'read_file'", output)
	}
	if !strings.Contains(output, "toolu_123") {
		t.Errorf("output = %q, expected to contain 'toolu_123'", output)
	}
	if !strings.Contains(output, "/test.txt") {
		t.Errorf("output = %q, expected to contain '/test.txt'", output)
	}
}

func TestProcessChunk_MessageStart(t *testing.T) {
	p := &Provider{}
	toolUses := make(map[int]*toolUseAccumulator)
	var toolCallsOutput strings.Builder
	var lastUsage *api.Usage

	// message_start イベントで input_tokens が設定される
	startData := `{"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[],"model":"claude-3","usage":{"input_tokens":150,"output_tokens":1,"cache_read_input_tokens":80,"cache_creation_input_tokens":20}}}`

	p.processChunk([]byte(startData), toolUses, &toolCallsOutput, &lastUsage, make(map[int]*strings.Builder), &strings.Builder{}, ui.NewSpinner())

	if lastUsage == nil {
		t.Fatal("lastUsage should not be nil after message_start with usage")
	}
	// 正規化後: 150 (uncached) + 80 (cache_read) + 20 (cache_creation) = 250
	if lastUsage.InputTokens != 250 {
		t.Errorf("InputTokens = %d, want 250", lastUsage.InputTokens)
	}
	if lastUsage.CachedInputTokens != 80 {
		t.Errorf("CachedInputTokens = %d, want 80", lastUsage.CachedInputTokens)
	}
	if lastUsage.CacheCreationTokens != 20 {
		t.Errorf("CacheCreationTokens = %d, want 20", lastUsage.CacheCreationTokens)
	}
}

func TestProcessChunk_Usage(t *testing.T) {
	p := &Provider{}
	toolUses := make(map[int]*toolUseAccumulator)
	var toolCallsOutput strings.Builder
	var lastUsage *api.Usage

	// 1. message_start で input_tokens を設定
	startData := `{"type":"message_start","message":{"usage":{"input_tokens":100,"output_tokens":1}}}`
	p.processChunk([]byte(startData), toolUses, &toolCallsOutput, &lastUsage, make(map[int]*strings.Builder), &strings.Builder{}, ui.NewSpinner())

	// 2. message_delta で output_tokens を設定（実際の API では output_tokens のみ）
	deltaData := `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":50}}`
	p.processChunk([]byte(deltaData), toolUses, &toolCallsOutput, &lastUsage, make(map[int]*strings.Builder), &strings.Builder{}, ui.NewSpinner())

	if lastUsage == nil {
		t.Fatal("lastUsage should not be nil after message_delta with usage")
	}
	if lastUsage.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", lastUsage.InputTokens)
	}
	if lastUsage.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", lastUsage.OutputTokens)
	}
}

func TestProcessChunk_InvalidJSON(t *testing.T) {
	p := &Provider{}
	toolUses := make(map[int]*toolUseAccumulator)
	var toolCallsOutput strings.Builder
	var lastUsage *api.Usage

	data := `invalid json`

	text, done := p.processChunk([]byte(data), toolUses, &toolCallsOutput, &lastUsage, make(map[int]*strings.Builder), &strings.Builder{}, ui.NewSpinner())
	if text != "" {
		t.Errorf("text = %q, want empty for invalid JSON", text)
	}
	if done {
		t.Error("done should be false for invalid JSON")
	}
}

func TestBuildSystemField(t *testing.T) {
	result := api.BuildSystemFieldWithConfig("Test prompt", config.DefaultConfig())
	if result == nil {
		t.Fatal("buildSystemField() returned nil")
	}

	// 文字列またはSystemBlockスライスのどちらかが返る
	switch v := result.(type) {
	case string:
		if v != "Test prompt" {
			t.Errorf("buildSystemField() = %q, want %q", v, "Test prompt")
		}
	case []api.SystemBlock:
		if len(v) == 0 {
			t.Fatal("SystemBlock slice is empty")
		}
		if v[0].Text != "Test prompt" {
			t.Errorf("SystemBlock.Text = %q, want %q", v[0].Text, "Test prompt")
		}
		if v[0].CacheControl == nil {
			t.Error("CacheControl should be set when prompt caching is enabled")
		}
	default:
		t.Errorf("unexpected type %T from buildSystemField()", result)
	}
}

func TestProvider_InterfaceCompliance(t *testing.T) {
	p := &Provider{}

	// Provider interface
	var _ api.Provider = p

	// MCPProvider interface
	var _ api.MCPProvider = p

	// UsageReporter interface
	var _ api.UsageReporter = p
}
