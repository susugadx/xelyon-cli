package subagent

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type managerTestProvider struct {
	name       string
	responseID string
	cleared    bool
}

func (p *managerTestProvider) Name() string { return p.name }

func (p *managerTestProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "", nil
}

func (p *managerTestProvider) SupportsImages() bool { return false }

func (p *managerTestProvider) ChatWithImage(context.Context, string, []api.Message, string, *api.ImageData, string) (string, error) {
	return "", nil
}

func (p *managerTestProvider) IsFunctionCallingEnabled() bool { return true }

func (p *managerTestProvider) ClearCache() {
	p.cleared = true
	p.responseID = ""
}

func (p *managerTestProvider) SetResponseID(id string) {
	p.responseID = id
}

// TestNewManager は既定依存で Manager が生成されることを確認します。
func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
	if manager.runHeadless == nil {
		t.Fatal("runHeadless should be configured")
	}
	if manager.providerFactory == nil {
		t.Fatal("providerFactory should be configured")
	}
}

// TestInferSubAgentModel は provider ごとの既定サブエージェントモデルを確認します。
func TestInferSubAgentModel(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{provider: "openai", want: "gpt-5.4-mini"},
		{provider: "claude", want: "claude-haiku-4-5-20251001"},
		{provider: "anthropic", want: "claude-haiku-4-5-20251001"},
		{provider: "gemini", want: "gemini-3.1-flash-lite-preview"},
		{provider: "deepseek", want: "deepseek-chat"},
		{provider: "groq", want: "llama-3.3-70b-versatile"},
		{provider: "openrouter", want: "openai/gpt-5.4-mini"},
		{provider: "unknown", want: ""},
	}

	for _, tt := range tests {
		if got := inferSubAgentModel(tt.provider); got != tt.want {
			t.Errorf("inferSubAgentModel(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

// TestNormalizeSubAgentModel はプレースホルダ文字列を未設定として扱うことを確認します。
func TestNormalizeSubAgentModel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: ""},
		{input: "   ", want: ""},
		{input: "sub_agent.default_model", want: ""},
		{input: "SUB_AGENT.DEFAULT_MODEL", want: ""},
		{input: "gpt-5.4-mini", want: "gpt-5.4-mini"},
	}

	for _, tt := range tests {
		if got := normalizeSubAgentModel(tt.input); got != tt.want {
			t.Errorf("normalizeSubAgentModel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestManagerSpawnWaitCompleted は spawn -> wait の基本フローを確認します。
func TestManagerSpawnWaitCompleted(t *testing.T) {
	cfg := config.DefaultConfig()
	provider := &managerTestProvider{name: "openai"}
	createdProvider := &managerTestProvider{name: "openai"}

	var gotModel string
	var gotProvider api.Provider
	var gotCfg *config.Config
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, provider api.Provider, cfg *config.Config) *RunResult {
			gotModel = model
			gotProvider = provider
			gotCfg = cfg
			return &RunResult{Status: "completed", Response: "sub-agent done"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "openai" {
				t.Fatalf("ProviderFactory providerName = %q, want openai", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "read files", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	response := manager.Wait([]string{id}, 0)
	if response.Status != "completed" {
		t.Fatalf("Wait().Status = %q, want completed", response.Status)
	}
	if len(response.Results) != 1 {
		t.Fatalf("len(Wait().Results) = %d, want 1", len(response.Results))
	}
	if response.Results[0].Status != "completed" {
		t.Fatalf("Wait().Results[0].Status = %q, want completed", response.Results[0].Status)
	}
	if response.Results[0].Output != "sub-agent done" {
		t.Fatalf("Wait().Results[0].Output = %q, want %q", response.Results[0].Output, "sub-agent done")
	}
	if gotModel != "gpt-5.4-mini" {
		t.Fatalf("resolved model = %q, want gpt-5.4-mini", gotModel)
	}
	if gotProvider != createdProvider {
		t.Fatal("expected fresh provider instance to be used")
	}
	if gotCfg == nil {
		t.Fatal("expected cloned config to be passed to runner")
	}
	if gotCfg.SubAgentPrompt != ExplorePrompt {
		t.Fatal("expected explore prompt to be injected into config for default task type")
	}
	if !gotCfg.Thinking.Enabled {
		t.Fatal("expected explore task type to enable default reasoning")
	}
	if gotCfg.Thinking.Level != "high" {
		t.Fatalf("gotCfg.Thinking.Level = %q, want high", gotCfg.Thinking.Level)
	}
	if !createdProvider.cleared {
		t.Fatal("expected fresh provider cache/response chain to be cleared")
	}
}

// TestManagerSpawn_DefaultModelFollowsMainProvider は DefaultModel 未設定時にメイン provider の最安モデルを選ぶことを確認します。
func TestManagerSpawn_DefaultModelFollowsMainProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.DefaultModel = ""
	provider := &managerTestProvider{name: "claude"}
	createdProvider := &managerTestProvider{name: "claude"}

	var gotModel string
	var gotProvider api.Provider
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, provider api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			gotProvider = provider
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "claude" {
				t.Fatalf("ProviderFactory providerName = %q, want claude", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "read files", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotModel != "claude-haiku-4-5-20251001" {
		t.Fatalf("resolved model = %q, want claude-haiku-4-5-20251001", gotModel)
	}
	if gotProvider != createdProvider {
		t.Fatal("expected fresh provider instance to be used")
	}
}

// TestManagerSpawn_PlaceholderModelFallsBackToMainProvider はプレースホルダ文字列指定時に provider 推定へフォールバックすることを確認します。
func TestManagerSpawn_PlaceholderModelFallsBackToMainProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.DefaultModel = ""
	provider := &managerTestProvider{name: "gemini"}
	createdProvider := &managerTestProvider{name: "gemini"}

	var gotModel string
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, _ api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "gemini" {
				t.Fatalf("ProviderFactory providerName = %q, want gemini", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect files", "", "sub_agent.default_model", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotModel != "gemini-3.1-flash-lite-preview" {
		t.Fatalf("resolved model = %q, want gemini-3.1-flash-lite-preview", gotModel)
	}
}

// TestManagerSpawn_PlaceholderConfigFallsBackToMainProvider は設定値がプレースホルダ文字列でも provider 推定へフォールバックすることを確認します。
func TestManagerSpawn_PlaceholderConfigFallsBackToMainProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.DefaultModel = "sub_agent.default_model"
	provider := &managerTestProvider{name: "claude"}
	createdProvider := &managerTestProvider{name: "claude"}

	var gotModel string
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, _ api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "claude" {
				t.Fatalf("ProviderFactory providerName = %q, want claude", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect files", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotModel != "claude-haiku-4-5-20251001" {
		t.Fatalf("resolved model = %q, want claude-haiku-4-5-20251001", gotModel)
	}
}

// TestManagerSpawn_DefaultModelExplicitOverride は明示設定モデルが provider 推定より優先されることを確認します。
func TestManagerSpawn_DefaultModelExplicitOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.DefaultModel = "gpt-5.4-mini"
	currentProvider := &managerTestProvider{name: "gemini"}
	createdProvider := &managerTestProvider{name: "openai"}

	var gotModel string
	var gotProvider api.Provider
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, provider api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			gotProvider = provider
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "openai" {
				t.Fatalf("ProviderFactory providerName = %q, want openai", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect files", "", "", "", currentProvider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotModel != "gpt-5.4-mini" {
		t.Fatalf("resolved model = %q, want gpt-5.4-mini", gotModel)
	}
	if gotProvider != createdProvider {
		t.Fatal("expected provider factory result to be used")
	}
}

// TestManagerSpawn_DefaultModelUnknownProviderFallback は不明 provider で OpenAI nano にフォールバックすることを確認します。
func TestManagerSpawn_DefaultModelUnknownProviderFallback(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.DefaultModel = ""
	currentProvider := &managerTestProvider{name: "custom"}
	createdProvider := &managerTestProvider{name: "openai"}

	var gotModel string
	var gotProvider api.Provider
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, provider api.Provider, _ *config.Config) *RunResult {
			gotModel = model
			gotProvider = provider
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "openai" {
				t.Fatalf("ProviderFactory providerName = %q, want openai", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect files", "", "", "", currentProvider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotModel != "gpt-5.4-mini" {
		t.Fatalf("resolved model = %q, want gpt-5.4-mini", gotModel)
	}
	if gotProvider != createdProvider {
		t.Fatal("expected fallback provider factory result to be used")
	}
}

// TestManagerSpawnParallel は複数サブエージェントが並列実行されることを確認します。
func TestManagerSpawnParallel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.MaxConcurrent = 2
	provider := &managerTestProvider{name: "openai"}

	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var current int32
	var maxSeen int32

	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, _ string, _ api.Provider, _ *config.Config) *RunResult {
			value := atomic.AddInt32(&current, 1)
			for {
				prev := atomic.LoadInt32(&maxSeen)
				if value <= prev || atomic.CompareAndSwapInt32(&maxSeen, prev, value) {
					break
				}
			}
			started <- struct{}{}
			<-release
			atomic.AddInt32(&current, -1)
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &managerTestProvider{name: providerName}, nil
		},
	})

	idA, err := manager.Spawn(context.Background(), "task A", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn(task A) error = %v", err)
	}
	idB, err := manager.Spawn(context.Background(), "task B", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn(task B) error = %v", err)
	}

	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("sub-agents did not start in time")
		}
	}
	close(release)

	response := manager.Wait([]string{idA, idB}, 0)
	if response.Status != "completed" {
		t.Fatalf("Wait().Status = %q, want completed", response.Status)
	}
	if atomic.LoadInt32(&maxSeen) < 2 {
		t.Fatalf("expected parallel execution, maxSeen = %d", atomic.LoadInt32(&maxSeen))
	}
}

// TestManagerWaitTimeout は timeout を返すことを確認します。
func TestManagerWaitTimeout(t *testing.T) {
	cfg := config.DefaultConfig()
	provider := &managerTestProvider{name: "openai"}
	release := make(chan struct{})

	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, _ string, _ api.Provider, _ *config.Config) *RunResult {
			<-release
			return &RunResult{Status: "completed", Response: "late"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &managerTestProvider{name: providerName}, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "slow task", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	response := manager.Wait([]string{id}, 10)
	if response.Status != "timeout" {
		t.Fatalf("Wait().Status = %q, want timeout", response.Status)
	}
	if response.Results[0].Status != "timeout" {
		t.Fatalf("Wait().Results[0].Status = %q, want timeout", response.Results[0].Status)
	}

	close(release)
	response = manager.Wait([]string{id}, 0)
	if response.Results[0].Status != "completed" {
		t.Fatalf("Wait() after release = %q, want completed", response.Results[0].Status)
	}
}

// TestManagerWaitNotFound は存在しない ID の待機を確認します。
func TestManagerWaitNotFound(t *testing.T) {
	manager := NewManager()
	response := manager.Wait([]string{"sub-999"}, 0)
	if response.Status != "error" {
		t.Fatalf("Wait().Status = %q, want error", response.Status)
	}
	if response.Results[0].Output != "agent not found" {
		t.Fatalf("Wait().Results[0].Output = %q, want %q", response.Results[0].Output, "agent not found")
	}
}

// TestManagerSpawnMaxConcurrent は同時実行上限を超えた場合にエラーを返すことを確認します。
func TestManagerSpawnMaxConcurrent(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SubAgent.MaxConcurrent = 1
	provider := &managerTestProvider{name: "openai"}
	release := make(chan struct{})

	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, _ string, _ api.Provider, _ *config.Config) *RunResult {
			<-release
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &managerTestProvider{name: providerName}, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "first", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn(first) error = %v", err)
	}

	_, err = manager.Spawn(context.Background(), "second", "", "", "", provider, cfg)
	if err == nil {
		t.Fatal("expected max concurrent error")
	}
	if !strings.Contains(err.Error(), "max concurrent") {
		t.Fatalf("unexpected error: %v", err)
	}

	close(release)
	_ = manager.Wait([]string{id}, 0)
}

// TestManagerSpawnSwitchesProviderForModel はモデルに応じた provider 切替を確認します。
func TestManagerSpawnSwitchesProviderForModel(t *testing.T) {
	cfg := config.DefaultConfig()
	currentProvider := &managerTestProvider{name: "openai"}
	createdProvider := &managerTestProvider{name: "claude"}

	var gotProvider api.Provider
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, _ string, provider api.Provider, _ *config.Config) *RunResult {
			gotProvider = provider
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "claude" {
				t.Fatalf("ProviderFactory providerName = %q, want claude", providerName)
			}
			return createdProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "compare", "", "claude-sonnet-4-6", "", currentProvider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotProvider != createdProvider {
		t.Fatal("expected provider factory result to be used")
	}
}

// TestManagerSpawn_SameProviderGetsFreshInstance は同一 provider でも親インスタンスを再利用しないことを確認します。
func TestManagerSpawn_SameProviderGetsFreshInstance(t *testing.T) {
	cfg := config.DefaultConfig()
	parentProvider := &managerTestProvider{name: "openai", responseID: "resp_parent"}
	subProvider := &managerTestProvider{name: "openai", responseID: "resp_stale"}

	var gotProvider api.Provider
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, _ string, provider api.Provider, _ *config.Config) *RunResult {
			gotProvider = provider
			return &RunResult{Status: "completed", Response: "ok"}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			if providerName != "openai" {
				t.Fatalf("ProviderFactory providerName = %q, want openai", providerName)
			}
			return subProvider, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect files", "", "", "", parentProvider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotProvider != subProvider {
		t.Fatal("expected sub-agent to receive the fresh provider instance")
	}
	if parentProvider.responseID != "resp_parent" {
		t.Fatalf("parent responseID = %q, want resp_parent", parentProvider.responseID)
	}
	if subProvider.responseID != "" {
		t.Fatalf("sub provider responseID = %q, want empty", subProvider.responseID)
	}
	if !subProvider.cleared {
		t.Fatal("expected sub provider state to be cleared before execution")
	}
}

// TestManagerGetSummary は集計結果に completed / running / コストが反映されることを確認します。
func TestManagerGetSummary(t *testing.T) {
	cfg := config.DefaultConfig()
	provider := &managerTestProvider{name: "openai"}
	release := make(chan struct{})

	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, message, model string, _ api.Provider, _ *config.Config) *RunResult {
			if message == "running task" {
				<-release
				return &RunResult{Status: "completed", Model: model}
			}
			return &RunResult{
				Status:         "completed",
				Model:          model,
				Response:       "done",
				InputTokens:    120,
				CachedTokens:   40,
				OutputTokens:   30,
				ThinkingTokens: 10,
				Cost:           0.0123,
				ToolExecutions: 2,
				DurationMs:     250,
			}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &managerTestProvider{name: providerName}, nil
		},
	})

	completedID, err := manager.Spawn(context.Background(), "completed task", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn(completed task) error = %v", err)
	}
	if response := manager.Wait([]string{completedID}, 0); response.Status != "completed" {
		t.Fatalf("Wait(completed task).Status = %q, want completed", response.Status)
	}

	runningID, err := manager.Spawn(context.Background(), "running task", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn(running task) error = %v", err)
	}

	summary := manager.GetSummary()
	if summary.TotalSpawned != 2 {
		t.Fatalf("summary.TotalSpawned = %d, want 2", summary.TotalSpawned)
	}
	if summary.TotalCompleted != 1 {
		t.Fatalf("summary.TotalCompleted = %d, want 1", summary.TotalCompleted)
	}
	if summary.TotalRunning != 1 {
		t.Fatalf("summary.TotalRunning = %d, want 1", summary.TotalRunning)
	}
	if summary.TotalInput != 120 {
		t.Fatalf("summary.TotalInput = %d, want 120", summary.TotalInput)
	}
	if summary.TotalCached != 40 {
		t.Fatalf("summary.TotalCached = %d, want 40", summary.TotalCached)
	}
	if summary.TotalOutput != 30 {
		t.Fatalf("summary.TotalOutput = %d, want 30", summary.TotalOutput)
	}
	if summary.TotalThinking != 10 {
		t.Fatalf("summary.TotalThinking = %d, want 10", summary.TotalThinking)
	}
	if summary.TotalCost != 0.0123 {
		t.Fatalf("summary.TotalCost = %f, want 0.0123", summary.TotalCost)
	}
	if summary.TotalTools != 2 {
		t.Fatalf("summary.TotalTools = %d, want 2", summary.TotalTools)
	}
	if len(summary.Agents) != 2 {
		t.Fatalf("len(summary.Agents) = %d, want 2", len(summary.Agents))
	}
	if summary.Agents[0].ID != completedID {
		t.Fatalf("summary.Agents[0].ID = %q, want %q", summary.Agents[0].ID, completedID)
	}
	if summary.Agents[1].ID != runningID {
		t.Fatalf("summary.Agents[1].ID = %q, want %q", summary.Agents[1].ID, runningID)
	}
	if summary.Agents[1].Status != "running" {
		t.Fatalf("summary.Agents[1].Status = %q, want running", summary.Agents[1].Status)
	}
	if summary.Agents[1].Model != "gpt-5.4-mini" {
		t.Fatalf("summary.Agents[1].Model = %q, want gpt-5.4-mini", summary.Agents[1].Model)
	}

	close(release)
	_ = manager.Wait([]string{runningID}, 0)
}

// TestManagerGetSummary_ErrorMessage はエラー詳細がサマリーへ反映されることを確認します。
func TestManagerGetSummary_ErrorMessage(t *testing.T) {
	cfg := config.DefaultConfig()
	provider := &managerTestProvider{name: "openai"}
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, _ api.Provider, _ *config.Config) *RunResult {
			return &RunResult{
				Status:       "error",
				Model:        model,
				ErrorMessage: "provider timeout",
			}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &managerTestProvider{name: providerName}, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "failing task", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	summary := manager.GetSummary()
	if len(summary.Agents) != 1 {
		t.Fatalf("len(summary.Agents) = %d, want 1", len(summary.Agents))
	}
	if summary.Agents[0].ErrorMessage != "provider timeout" {
		t.Fatalf("summary.Agents[0].ErrorMessage = %q, want %q", summary.Agents[0].ErrorMessage, "provider timeout")
	}
}
