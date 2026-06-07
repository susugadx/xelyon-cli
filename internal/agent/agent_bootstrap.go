package agent

import (
	"context"
	"io"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/i18n"
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcptool"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/tools"

	// Subpackage imports - trigger init() for tool registration
	_ "github.com/susugadx/xelyon-cli/internal/tools/applypatch"
	toolsdev "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	_ "github.com/susugadx/xelyon-cli/internal/tools/gathercontext"
	_ "github.com/susugadx/xelyon-cli/internal/tools/planning"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
	_ "github.com/susugadx/xelyon-cli/internal/tools/skills"
)

// NewAgent は新しい Agent を作成する。
func NewAgent(model string, provider api.Provider, headless bool) *Agent {
	return NewAgentWithRuntime(model, provider, headless, NewAgentRuntimeWithConfig(config.DefaultConfig()))
}

// NewAgentWithRuntime は runtime を指定して新しい Agent を作成する。
func NewAgentWithRuntime(model string, provider api.Provider, headless bool, runtime *AgentRuntime) *Agent {
	runtime = normalizeAgentRuntime(runtime)
	runtime.refreshInvocationCWD()
	runtime.ensureTaskLedger()
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
	lspClient := newAgentLSPClient(cfg, runtime.effectiveInvocationCWD(), errOut)
	providerRuntimeName := providerRuntimeNameFromProvider(provider)
	providerConfigKey := providerConfigKeyFromProvider(provider)
	toolVisibility := resolveToolVisibilityPolicyWithConfig(providerRuntimeName, model, cfg, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true})

	configureMCPTools(provider, mcpManager.GetTools(), errOut)
	runtime.effectiveRegistry().SetExcludedTools(toolVisibility.excluded())

	systemPrompt := buildAgentSystemPrompt(provider, model, cfg, mcpManager, runtime.effectiveInvocationCWD())
	toolCache := runtime.effectiveToolCache()

	agent := &Agent{
		Model:             model,
		CurrentModel:      model,
		CurrentProvider:   provider,
		ProviderName:      providerRuntimeName,
		ProviderConfigKey: providerConfigKey,
		Runtime:           runtime,
		History:           []api.Message{},
		mcpManager:        mcpManager,
		lspClient:         lspClient,
		SystemPrompt:      systemPrompt,
		Stats:             NewSessionStats(providerRuntimeName, model),
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
	agent.promptMgr = newPromptManager(agent)
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
		mcptool.RegisterToRegistry(registry, manager, mcpToolDefinitions(manager.GetTools()))
	}

	return manager
}

func buildAgentSystemPrompt(provider api.Provider, model string, cfg *config.Config, manager *mcp.Manager, invocationCWD string) string {
	providerName := providerRuntimeNameFromProvider(provider)
	systemPrompt := prompt.GetSystemPromptForProviderWithConfig(providerName, model, cfg)
	if manager != nil && len(manager.GetTools()) > 0 {
		systemPrompt += buildMCPToolsPrompt(manager)
	}
	systemPrompt = injectSkillCatalogPrompt(systemPrompt, invocationCWD)

	// プロバイダー別プレフィックスを Workflow Rules の直前に注入
	return prompt.BuildProviderSystemPromptWithConfig(systemPrompt, providerName, model, cfg)
}

func newAgentLSPClient(cfg *config.Config, invocationCWD string, errOut io.Writer) *lsp.Client {
	if cfg == nil || !cfg.LSP.Enabled {
		return nil
	}

	cwd := invocationCWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return nil
		}
	}
	clientRoot := cwd
	warmupRoot, hasProjectRoot := resolveLSPStartupProjectRoot(cfg, cwd)
	if hasProjectRoot {
		clientRoot = warmupRoot
	}

	client := lsp.NewClient(clientRoot)
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

	if hasProjectRoot && !shouldSkipLSPWarmup() {
		go warmupLSPClient(client, warmupRoot, servers, errOut)
	}

	return client
}

func resolveLSPStartupProjectRoot(cfg *config.Config, cwd string) (string, bool) {
	return config.ResolveProjectInstructionProjectRootForDir(cfg, cwd)
}

func setUsageReporter(agent *Agent, provider api.Provider) {
	reporter, ok := provider.(api.UsageReporter)
	if !ok {
		return
	}
	reporter.SetUsageCallback(func(u api.Usage) {
		agent.statsMu.Lock()
		defer agent.statsMu.Unlock()
		agent.Stats.AddUsageForConfig(agent.cfg(), u)
	})
}

// shouldSkipLSPWarmup は環境変数指定時に warm up を無効化する。
func shouldSkipLSPWarmup() bool {
	return os.Getenv("XELYON_DISABLE_LSP_WARMUP") == "1"
}
