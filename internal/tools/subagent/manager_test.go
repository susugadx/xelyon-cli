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
	name string
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

// TestManagerSpawnWaitCompleted は spawn -> wait の基本フローを確認します。
func TestManagerSpawnWaitCompleted(t *testing.T) {
	cfg := config.DefaultConfig()
	provider := &managerTestProvider{name: "openai"}

	var gotModel string
	var gotProvider api.Provider
	var gotCfg *config.Config
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ string, model string, provider api.Provider, cfg *config.Config) *RunResult {
			gotModel = model
			gotProvider = provider
			gotCfg = cfg
			return &RunResult{Status: "completed", Response: "sub-agent done"}
		},
	})

	id, err := manager.Spawn("read files", "", "", provider, cfg)
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
	if gotModel != "gpt-5.4-nano" {
		t.Fatalf("resolved model = %q, want gpt-5.4-nano", gotModel)
	}
	if gotProvider != provider {
		t.Fatal("expected current provider to be reused")
	}
	if gotCfg == nil {
		t.Fatal("expected cloned config to be passed to runner")
	}
	if gotCfg.SubAgentPrompt != SubAgentSystemPrompt {
		t.Fatal("expected sub-agent prompt to be injected into config")
	}
	if gotCfg.Thinking.Enabled {
		t.Fatal("expected default sub-agent reasoning to be disabled")
	}
}

// TestManagerSpawnParallel は複数サブエージェントが並列実行されることを確認します。
func TestManagerSpawnParallel(t *testing.T) {
	cfg := config.DefaultConfig()
	provider := &managerTestProvider{name: "openai"}

	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var current int32
	var maxSeen int32

	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ string, _ string, _ api.Provider, _ *config.Config) *RunResult {
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
	})

	idA, err := manager.Spawn("task A", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn(task A) error = %v", err)
	}
	idB, err := manager.Spawn("task B", "", "", provider, cfg)
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
		RunHeadless: func(_ string, _ string, _ api.Provider, _ *config.Config) *RunResult {
			<-release
			return &RunResult{Status: "completed", Response: "late"}
		},
	})

	id, err := manager.Spawn("slow task", "", "", provider, cfg)
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
		RunHeadless: func(_ string, _ string, _ api.Provider, _ *config.Config) *RunResult {
			<-release
			return &RunResult{Status: "completed", Response: "ok"}
		},
	})

	id, err := manager.Spawn("first", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn(first) error = %v", err)
	}

	_, err = manager.Spawn("second", "", "", provider, cfg)
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
		RunHeadless: func(_ string, _ string, provider api.Provider, _ *config.Config) *RunResult {
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

	id, err := manager.Spawn("compare", "claude-sonnet-4-6", "", currentProvider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	if gotProvider != createdProvider {
		t.Fatal("expected provider factory result to be used")
	}
}
