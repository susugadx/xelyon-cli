package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/i18n"
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/tools"

	// Subpackage imports - trigger init() for tool registration
	_ "github.com/susugadx/xelyon-cli/internal/tools/applypatch"
	toolsdev "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	_ "github.com/susugadx/xelyon-cli/internal/tools/gathercontext"
	_ "github.com/susugadx/xelyon-cli/internal/tools/planning"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
)

// NewAgent は新しい Agent を作成する。
func NewAgent(model string, provider api.Provider, headless bool) *Agent {
	return NewAgentWithRuntime(model, provider, headless, NewAgentRuntimeWithConfig(config.DefaultConfig()))
}

// NewAgentWithRuntime は runtime を指定して新しい Agent を作成する。
func NewAgentWithRuntime(model string, provider api.Provider, headless bool, runtime *AgentRuntime) *Agent {
	runtime = normalizeAgentRuntime(runtime)
	cfg := runtime.effectiveConfig()
	api.ApplyRuntimeConfig(provider, cfg)

	runtimeUI := runtime.effectiveUI()
	api.ApplyUIRuntime(provider, runtimeUI)
	out := runtimeUI.Output()
	errOut := runtimeUI.ErrorOutput()

	applyAgentUILanguage(cfg)
	toolsdev.CleanupArtifactsWithWriter(errOut)

	storage := newAgentHistoryStorage(out)
	changeStorage := newAgentChangeStorage(out)
	mcpManager := setupMCPManager(cfg, headless, out, errOut, runtime.effectiveRegistry())
	lspClient := newAgentLSPClient(cfg, errOut)
	toolVisibility := resolveToolVisibilityPolicy(provider.Name(), model, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true})

	configureMCPTools(provider, mcpManager.GetTools(), errOut)
	runtime.effectiveRegistry().SetExcludedTools(toolVisibility.excluded())

	systemPrompt := buildAgentSystemPrompt(provider, model, cfg, mcpManager)
	toolCache := runtime.effectiveToolCache()

	agent := &Agent{
		Model:             model,
		CurrentModel:      model,
		CurrentProvider:   provider,
		ProviderName:      strings.ToLower(provider.Name()),
		ProviderConfigKey: providerConfigKeyFromProvider(provider),
		Runtime:           runtime,
		History:           []api.Message{},
		mcpManager:        mcpManager,
		lspClient:         lspClient,
		SystemPrompt:      systemPrompt,
		Stats:             NewSessionStats(strings.ToLower(provider.Name()), model),
		ToolCache:         toolCache,
		LocatorRegistry:   locator.NewRegistry(),
		status:            statusHolder{status: defaultStatus()},
		agentConversationState: agentConversationState{
			session:     history.NewSession(model),
			storage:     storage,
			lastOutputs: []string{},
		},
		agentWorkspaceState: agentWorkspaceState{
			changeStack:   []tools.FileChange{},
			changeStorage: changeStorage,
		},
	}
	agent.syncSessionRuntimeIdentity()
	setUsageReporter(agent, provider)

	return agent
}

func applyAgentUILanguage(cfg *config.Config) {
	if cfg == nil {
		return
	}
	if cfg.General.UILanguage != "" && cfg.General.UILanguage != "auto" {
		i18n.SetLang(cfg.General.UILanguage)
	}
}

func newAgentHistoryStorage(out io.Writer) *history.Storage {
	storage, err := history.NewStorage()
	if err != nil {
		red.Fprintf(out, "Warning: Failed to initialize history storage: %v\n", err)
		return nil
	}
	return storage
}

func newAgentChangeStorage(out io.Writer) *history.ChangeStorage {
	changeStorage, err := history.NewChangeStorage()
	if err != nil {
		yellow.Fprintf(out, "Warning: Failed to initialize change storage: %v\n", err)
		return nil
	}
	return changeStorage
}

func setupMCPManager(cfg *config.Config, headless bool, out, errOut io.Writer, registry *tools.Registry) *mcp.Manager {
	manager := mcp.NewManager()
	manager.SetOutput(errOut)
	if cfg == nil || !cfg.MCP.Enabled || (headless && !cfg.MCP.Headless) || os.Getenv("XELYON_DISABLE_MCP") == "1" {
		return manager
	}

	if err := manager.LoadConfig(); err != nil {
		yellow.Fprintf(out, "Warning: Failed to load MCP config: %v\n", err)
	}

	if err := manager.Connect(context.Background()); err != nil {
		yellow.Fprintf(out, "Warning: MCP connection error: %v\n", err)
	}

	if len(manager.GetTools()) > 0 {
		manager.RegisterToToolRegistry(registry)
	}

	return manager
}

func buildAgentSystemPrompt(provider api.Provider, model string, cfg *config.Config, manager *mcp.Manager) string {
	systemPrompt := prompt.GetSystemPromptForProvider(provider.Name(), model)
	if manager != nil && len(manager.GetTools()) > 0 {
		systemPrompt += buildMCPToolsPrompt(manager)
	}

	// プロバイダー別プレフィックスを Workflow Rules の直前に注入
	return prompt.BuildProviderSystemPromptWithConfig(systemPrompt, provider.Name(), model, cfg)
}

func newAgentLSPClient(cfg *config.Config, errOut io.Writer) *lsp.Client {
	if cfg == nil || !cfg.LSP.Enabled {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	client := lsp.NewClient(cwd)
	client.SetOutput(errOut)
	client.SetErrorOutput(errOut)

	servers := make(map[string]lsp.ServerConfig, len(cfg.LSP.Servers))
	for lang, serverCfg := range cfg.LSP.Servers {
		servers[lang] = lsp.ServerConfig{
			Command:  serverCfg.Command,
			Args:     serverCfg.Args,
			Disabled: serverCfg.Disabled,
		}
	}
	client.SetConfigs(servers)

	if !shouldSkipLSPWarmup() {
		go warmupLSPClient(client, errOut)
	}

	return client
}

func warmupLSPClient(client *lsp.Client, errOut io.Writer) {
	warmCtx, warmCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer warmCancel()
	if _, err := client.GetServer(warmCtx, "go"); err != nil {
		fmt.Fprintf(errOut, "LSP warm-up: gopls not available (%v)\n", err)
	}
}

func setUsageReporter(agent *Agent, provider api.Provider) {
	reporter, ok := provider.(api.UsageReporter)
	if !ok {
		return
	}
	reporter.SetUsageCallback(func(u api.Usage) {
		agent.statsMu.Lock()
		defer agent.statsMu.Unlock()
		agent.Stats.AddUsage(u)
	})
}

// shouldSkipLSPWarmup は環境変数指定時に warm up を無効化する。
func shouldSkipLSPWarmup() bool {
	return os.Getenv("XELYON_DISABLE_LSP_WARMUP") == "1"
}
