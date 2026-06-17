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
	if manager.Events() == nil {
		t.Fatal("event channel should be configured")
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

// TestManagerWait_ErrorOutputPrefersErrorMessage は error result に Response が残っていても
// wait_agent output では失敗理由を優先することを確認します。
func TestManagerWait_ErrorOutputPrefersErrorMessage(t *testing.T) {
	cfg := config.DefaultConfig()
	provider := &managerTestProvider{name: "openai"}
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, _ string, _ api.Provider, _ *config.Config) *RunResult {
			return &RunResult{
				Status:       "error",
				Response:     `{"tool":"read_file","args":{"path":"loop.go"}}`,
				ErrorMessage: "tool loop limit reached (10 iterations)",
			}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &managerTestProvider{name: providerName}, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "looping task", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	response := manager.Wait([]string{id}, 0)
	if response.Status != "error" {
		t.Fatalf("Wait().Status = %q, want error", response.Status)
	}
	if len(response.Results) != 1 {
		t.Fatalf("len(Wait().Results) = %d, want 1", len(response.Results))
	}
	if response.Results[0].Status != "error" {
		t.Fatalf("Wait().Results[0].Status = %q, want error", response.Results[0].Status)
	}
	if response.Results[0].Output != "tool loop limit reached (10 iterations)" {
		t.Fatalf("Wait().Results[0].Output = %q, want error message", response.Results[0].Output)
	}
}

// TestManagerSpawn_EmitsCompletionEvent は完了イベントが発行されることを確認します。
func TestManagerSpawn_EmitsCompletionEvent(t *testing.T) {
	cfg := config.DefaultConfig()
	provider := &managerTestProvider{name: "openai"}
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(ctx context.Context, _ string, _ string, _ api.Provider, _ *config.Config) *RunResult {
			EmitEvent(ctx, SubAgentEvent{Tool: "read_file", Phase: "start", FilePath: "manager.go"})
			return &RunResult{
				Status:         "completed",
				Response:       "ok",
				ToolExecutions: 1,
				DurationMs:     1200,
			}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &managerTestProvider{name: providerName}, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "read files", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	events := manager.Events()
	start := <-events
	if start.AgentID != id {
		t.Fatalf("start.AgentID = %q, want %q", start.AgentID, id)
	}
	if start.Tool != "read_file" || start.Phase != "start" {
		t.Fatalf("unexpected start event: %+v", start)
	}

	completed := <-events
	if completed.Tool != "_completed" || completed.Phase != "end" {
		t.Fatalf("unexpected completion event: %+v", completed)
	}
	if !completed.Success {
		t.Fatal("completion event should be successful")
	}
	if !strings.Contains(completed.Output, "completed") {
		t.Fatalf("completed.Output = %q, want completion summary", completed.Output)
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
