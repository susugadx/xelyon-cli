package subagent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type registerTestProvider struct{}

func (p *registerTestProvider) Name() string { return "openai" }

func (p *registerTestProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "", nil
}

func (p *registerTestProvider) SupportsImages() bool { return false }

func (p *registerTestProvider) ChatWithImage(context.Context, string, []api.Message, string, *api.ImageData, string) (string, error) {
	return "", nil
}

func (p *registerTestProvider) IsFunctionCallingEnabled() bool { return true }

// TestSpawnAgentToolParameters は schema を確認します。
func TestSpawnAgentToolParameters(t *testing.T) {
	tool := NewSpawnAgentTool(NewManager())
	params := tool.Parameters()

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties map, got %T", params["properties"])
	}
	if len(properties) != 3 {
		t.Fatalf("len(properties) = %d, want 3", len(properties))
	}
	if _, ok := properties["message"]; !ok {
		t.Fatal("message parameter is missing")
	}
	if _, ok := properties["model"]; !ok {
		t.Fatal("model parameter is missing")
	}
	if _, ok := properties["reasoning_effort"]; !ok {
		t.Fatal("reasoning_effort parameter is missing")
	}

	modelParam, ok := properties["model"].(map[string]interface{})
	if !ok {
		t.Fatalf("model parameter should be an object, got %T", properties["model"])
	}
	description, ok := modelParam["description"].(string)
	if !ok {
		t.Fatalf("model description should be a string, got %T", modelParam["description"])
	}
	if !strings.Contains(description, "DO NOT set this field") {
		t.Fatalf("model description should strongly discourage manual override: %q", description)
	}

	effortParam, ok := properties["reasoning_effort"].(map[string]interface{})
	if !ok {
		t.Fatalf("reasoning_effort parameter should be an object, got %T", properties["reasoning_effort"])
	}
	effortDescription, ok := effortParam["description"].(string)
	if !ok {
		t.Fatalf("reasoning_effort description should be a string, got %T", effortParam["description"])
	}
	if !strings.Contains(effortDescription, "DO NOT set this field") {
		t.Fatalf("reasoning_effort description should strongly discourage manual override: %q", effortDescription)
	}

}

// TestSpawnAgentToolRunEmptyMessage は空 message を拒否することを確認します。
func TestSpawnAgentToolRunEmptyMessage(t *testing.T) {
	tool := NewSpawnAgentTool(NewManager())
	result, change, err := tool.Run(tools.ExecutionContext{}, map[string]string{"message": ""})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if change != nil {
		t.Fatal("change should be nil")
	}
	if result != "Error: message is required" {
		t.Fatalf("Run() = %q, want %q", result, "Error: message is required")
	}
}

// TestSpawnAgentToolRun は JSON 応答を返すことを確認します。
func TestSpawnAgentToolRun(t *testing.T) {
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, _ string, _ api.Provider, _ *config.Config) *RunResult {
			return &RunResult{Status: "completed", Response: "done"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &registerTestProvider{}, nil
		},
	})
	tool := NewSpawnAgentTool(manager)
	execCtx := tools.ExecutionContext{
		Provider: &registerTestProvider{},
		Config:   config.DefaultConfig(),
		Stdin:    strings.NewReader(""),
	}

	result, change, err := tool.Run(execCtx, map[string]string{"message": "inspect files"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if change != nil {
		t.Fatal("change should be nil")
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON result: %v", err)
	}
	if parsed["status"] != "running" {
		t.Fatalf("status = %q, want running", parsed["status"])
	}
	if parsed["agent_id"] == "" {
		t.Fatal("agent_id should not be empty")
	}
}

// TestWaitAgentToolRunEmptyIDs は空 ids を拒否することを確認します。
func TestWaitAgentToolRunEmptyIDs(t *testing.T) {
	tool := NewWaitAgentTool(NewManager())
	result, change, err := tool.Run(tools.ExecutionContext{}, map[string]string{"ids": "[]"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if change != nil {
		t.Fatal("change should be nil")
	}
	if result != "Error: ids is empty" {
		t.Fatalf("Run() = %q, want %q", result, "Error: ids is empty")
	}
}

// TestWaitAgentToolParameters は schema を確認します。
func TestWaitAgentToolParameters(t *testing.T) {
	tool := NewWaitAgentTool(NewManager())
	params := tool.Parameters()

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties map, got %T", params["properties"])
	}
	timeoutParam, ok := properties["timeout_ms"].(map[string]interface{})
	if !ok {
		t.Fatalf("timeout_ms parameter should be an object, got %T", properties["timeout_ms"])
	}
	timeoutDescription, ok := timeoutParam["description"].(string)
	if !ok {
		t.Fatalf("timeout_ms description should be a string, got %T", timeoutParam["description"])
	}
	if !strings.Contains(timeoutDescription, "DO NOT set this field") {
		t.Fatalf("timeout_ms description should strongly discourage manual override: %q", timeoutDescription)
	}
}

// TestWaitAgentToolRun は wait_agent の JSON 応答を確認します。
func TestWaitAgentToolRun(t *testing.T) {
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, _ string, _ api.Provider, _ *config.Config) *RunResult {
			return &RunResult{Status: "completed", Response: "done"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &registerTestProvider{}, nil
		},
	})
	id, err := manager.Spawn(context.Background(), "inspect files", "", "", &registerTestProvider{}, config.DefaultConfig())
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	tool := NewWaitAgentTool(manager)
	result, change, err := tool.Run(tools.ExecutionContext{}, map[string]string{
		"ids": `["` + id + `"]`,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if change != nil {
		t.Fatal("change should be nil")
	}

	var parsed WaitResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON result: %v", err)
	}
	if parsed.Status != "completed" {
		t.Fatalf("parsed.Status = %q, want completed", parsed.Status)
	}
	if len(parsed.Results) != 1 || parsed.Results[0].Output != "done" {
		t.Fatalf("unexpected parsed results: %+v", parsed.Results)
	}
}

// TestWaitAgentToolRun_ShortTimeoutIgnored は短すぎる timeout_ms を無制限待機へフォールバックすることを確認します。
func TestWaitAgentToolRun_ShortTimeoutIgnored(t *testing.T) {
	release := make(chan struct{})
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, _ string, _ api.Provider, _ *config.Config) *RunResult {
			<-release
			return &RunResult{Status: "completed", Response: "done"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &registerTestProvider{}, nil
		},
	})
	id, err := manager.Spawn(context.Background(), "inspect files", "", "", &registerTestProvider{}, config.DefaultConfig())
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		close(release)
	}()

	tool := NewWaitAgentTool(manager)
	result, change, err := tool.Run(tools.ExecutionContext{}, map[string]string{
		"ids":        `["` + id + `"]`,
		"timeout_ms": "10",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if change != nil {
		t.Fatal("change should be nil")
	}

	var parsed WaitResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON result: %v", err)
	}
	if parsed.Status != "completed" {
		t.Fatalf("parsed.Status = %q, want completed", parsed.Status)
	}
	if len(parsed.Results) != 1 || parsed.Results[0].Status != "completed" {
		t.Fatalf("unexpected parsed results: %+v", parsed.Results)
	}
}
