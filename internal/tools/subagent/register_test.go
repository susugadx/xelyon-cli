package subagent

import (
	"context"
	"encoding/json"
	"reflect"
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

type spawnToolStub struct {
	ctx             context.Context
	message         string
	taskType        string
	model           string
	reasoningEffort string
	provider        api.Provider
	cfg             *config.Config
}

func (s *spawnToolStub) Spawn(ctx context.Context, message, taskType, model, reasoningEffort string, provider api.Provider, cfg *config.Config) (string, error) {
	s.ctx = ctx
	s.message = message
	s.taskType = taskType
	s.model = model
	s.reasoningEffort = reasoningEffort
	s.provider = provider
	s.cfg = cfg
	return "sub-boundary", nil
}

type spawnToolRuntimeContextStub struct {
	spawnToolStub
	runtimeCtx SpawnRuntimeContext
}

func (s *spawnToolRuntimeContextStub) SpawnWithRuntimeContext(ctx context.Context, message, taskType, model, reasoningEffort string, provider api.Provider, cfg *config.Config, runtimeCtx SpawnRuntimeContext) (string, error) {
	s.runtimeCtx = runtimeCtx
	return s.Spawn(ctx, message, taskType, model, reasoningEffort, provider, cfg)
}

type waitToolStub struct {
	ids       []string
	timeoutMs int
	response  WaitResponse
}

func (s *waitToolStub) Wait(ids []string, timeoutMs int) WaitResponse {
	s.ids = append([]string(nil), ids...)
	s.timeoutMs = timeoutMs
	return s.response
}

// TestSpawnAgentToolParameters は schema を確認します。
func TestSpawnAgentToolParameters(t *testing.T) {
	tool := NewSpawnAgentTool(NewManager())
	params := tool.Parameters()

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties map, got %T", params["properties"])
	}
	if len(properties) != 4 {
		t.Fatalf("len(properties) = %d, want 4", len(properties))
	}
	if _, ok := properties["message"]; !ok {
		t.Fatal("message parameter is missing")
	}
	if _, ok := properties["task_type"]; !ok {
		t.Fatal("task_type parameter is missing")
	}
	if _, ok := properties["model"]; !ok {
		t.Fatal("model parameter is missing")
	}
	if _, ok := properties["reasoning_effort"]; !ok {
		t.Fatal("reasoning_effort parameter is missing")
	}
	reasoningEffort, ok := properties["reasoning_effort"].(map[string]interface{})
	if !ok {
		t.Fatalf("reasoning_effort schema = %T, want map[string]interface{}", properties["reasoning_effort"])
	}
	if got, want := reasoningEffort["enum"], []string{"off", "low", "medium", "high", "xhigh"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("reasoning_effort enum = %#v, want %#v", got, want)
	}
	if got := params["additionalProperties"]; got != false {
		t.Fatalf("additionalProperties = %#v, want false", got)
	}
}

func TestReasoningEffortSchemaEnumMatchesRuntime(t *testing.T) {
	for _, effort := range []string{"off", "low", "medium", "high", "xhigh"} {
		cfg := config.DefaultConfig()
		if err := applyReasoningEffort(cfg, effort); err != nil {
			t.Fatalf("applyReasoningEffort(%q) error = %v", effort, err)
		}
		if effort == "off" {
			if cfg.Thinking.Enabled {
				t.Fatalf("applyReasoningEffort(%q) enabled thinking, want disabled", effort)
			}
			continue
		}
		if !cfg.Thinking.Enabled || cfg.Thinking.Level != effort {
			t.Fatalf("applyReasoningEffort(%q) thinking = enabled:%t level:%q", effort, cfg.Thinking.Enabled, cfg.Thinking.Level)
		}
	}

	if err := applyReasoningEffort(config.DefaultConfig(), "highest"); err == nil {
		t.Fatal("applyReasoningEffort(highest) error = nil, want invalid reasoning_effort error")
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

func TestSpawnAgentToolRun_UsesSpawnerBoundary(t *testing.T) {
	stub := &spawnToolStub{}
	tool := NewSpawnAgentTool(stub)
	execCfg := config.DefaultConfig()
	execCtx := tools.ExecutionContext{
		Context:  context.Background(),
		Provider: &registerTestProvider{},
		Config:   execCfg,
		Stdin:    strings.NewReader(""),
	}

	result, change, err := tool.Run(execCtx, map[string]string{
		"message":          "scope check",
		"task_type":        "edit",
		"model":            "gpt-5.4-mini",
		"reasoning_effort": "high",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if change != nil {
		t.Fatal("change should be nil")
	}
	if stub.ctx == nil {
		t.Fatal("expected context to be forwarded")
	}
	if stub.message != "scope check" || stub.taskType != "edit" {
		t.Fatalf("unexpected spawn args: message=%q taskType=%q", stub.message, stub.taskType)
	}
	if stub.model != "gpt-5.4-mini" || stub.reasoningEffort != "high" {
		t.Fatalf("unexpected model args: model=%q effort=%q", stub.model, stub.reasoningEffort)
	}
	if stub.provider == nil || stub.cfg != execCfg {
		t.Fatal("provider/config should be passed through to spawner")
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON result: %v", err)
	}
	if parsed["agent_id"] != "sub-boundary" || parsed["status"] != "running" {
		t.Fatalf("unexpected parsed payload: %+v", parsed)
	}
}

func TestSpawnAgentToolRun_ForwardsCurrentModelToRuntimeAwareSpawner(t *testing.T) {
	stub := &spawnToolRuntimeContextStub{}
	tool := NewSpawnAgentTool(stub)
	execCtx := tools.ExecutionContext{
		Context:  context.Background(),
		Provider: &registerTestProvider{},
		Model:    "corp-current-deployment",
		Config:   config.DefaultConfig(),
		Stdin:    strings.NewReader(""),
	}

	result, _, err := tool.Run(execCtx, map[string]string{"message": "scope check"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(result, "sub-boundary") {
		t.Fatalf("Run() = %q, want sub-boundary response", result)
	}
	if stub.runtimeCtx.CurrentModel != "corp-current-deployment" {
		t.Fatalf("CurrentModel forwarded = %q, want parent current model", stub.runtimeCtx.CurrentModel)
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
	if len(properties) != 2 {
		t.Fatalf("len(properties) = %d, want 2", len(properties))
	}
	if _, ok := properties["ids"]; !ok {
		t.Fatal("ids parameter is missing")
	}
	if _, ok := properties["timeout_ms"]; !ok {
		t.Fatal("timeout_ms parameter is missing")
	}
	if got := params["additionalProperties"]; got != false {
		t.Fatalf("additionalProperties = %#v, want false", got)
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
	id, err := manager.Spawn(context.Background(), "inspect files", "", "", "", &registerTestProvider{}, config.DefaultConfig())
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

func TestWaitAgentToolRun_UsesWaiterBoundary(t *testing.T) {
	stub := &waitToolStub{
		response: WaitResponse{
			Status: "completed",
			Results: []WaitResult{
				{AgentID: "sub-001", Status: "completed", Output: "ok"},
			},
		},
	}
	tool := NewWaitAgentTool(stub)

	result, change, err := tool.Run(tools.ExecutionContext{}, map[string]string{
		"ids":        `["sub-001"]`,
		"timeout_ms": "70000",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if change != nil {
		t.Fatal("change should be nil")
	}
	if len(stub.ids) != 1 || stub.ids[0] != "sub-001" {
		t.Fatalf("unexpected waiter ids: %+v", stub.ids)
	}
	if stub.timeoutMs != 70000 {
		t.Fatalf("timeoutMs = %d, want 70000", stub.timeoutMs)
	}

	var parsed WaitResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON result: %v", err)
	}
	if parsed.Status != "completed" || len(parsed.Results) != 1 || parsed.Results[0].Output != "ok" {
		t.Fatalf("unexpected parsed response: %+v", parsed)
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
	id, err := manager.Spawn(context.Background(), "inspect files", "", "", "", &registerTestProvider{}, config.DefaultConfig())
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
