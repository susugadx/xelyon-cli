package setup

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// Options は setup checklist の入力状態を表す。
type Options struct {
	Config                       *config.Config
	CWD                          string
	Provider                     string
	Model                        string
	ProjectConfigInstructionMode ProjectConfigInstructionMode
}

// ProjectConfigInstructionMode は xelyon.yaml 不在時の案内 surface を表す。
type ProjectConfigInstructionMode string

const (
	// ProjectConfigInstructionManual は CLI/classic 向けに手動作成だけを案内する。
	ProjectConfigInstructionManual ProjectConfigInstructionMode = ""
	// ProjectConfigInstructionTUI は TUI 向けに /project も案内する。
	ProjectConfigInstructionTUI ProjectConfigInstructionMode = "tui"
)

// Report は first-run setup と project recommendation の診断結果を表す。
type Report struct {
	Provider ProviderStatus
	Global   []Item
	Project  []Item
}

// ProviderStatus は provider credential の診断結果を表す。
type ProviderStatus struct {
	Provider     string
	DisplayName  string
	Ready        bool
	Detail       string
	Instructions []string
}

// Item は setup checklist の 1 項目を表す。
type Item struct {
	Key         string
	Status      string
	Message     string
	Detail      string
	Instruction string
}

var (
	lookPath               = exec.LookPath
	getwd                  = os.Getwd
	detectProjectLanguages = lsp.DetectProjectLanguages
	loadProjectConfig      = config.LoadProjectConfigForDirWithError
	resolveProjectRoot     = config.ResolveProjectInstructionProjectRootForDir
)

// BuildReport は setup checklist の状態を作る。
func BuildReport(opts Options) Report {
	cfg := opts.Config
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	provider := resolveSetupProviderName(opts.Provider, cfg.DefaultProvider)
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = cfg.GetSelectedModelForProvider(provider)
	}

	report := Report{
		Provider: ProviderCredentialStatus(provider),
	}
	providerInstruction := ""
	if !report.Provider.Ready {
		providerInstruction = firstInstruction(report.Provider.Instructions)
	}
	report.Global = append(report.Global,
		Item{Key: "provider", Status: statusLabel(report.Provider.Ready), Message: "Provider credential", Detail: report.Provider.Detail, Instruction: providerInstruction},
		defaultModelItem(cfg, provider, model),
		toolAvailabilityItem("rg", "ripgrep", "Install ripgrep for faster project map and search_code."),
		toolAvailabilityItem("git", "git", "Install git to enable repository-aware context."),
	)

	report.Project = projectItems(cfg, opts.CWD, opts.ProjectConfigInstructionMode)
	return report
}

// Render は setup checklist を user-facing text として出力する。
func Render(out io.Writer, report Report) {
	if out == nil {
		return
	}
	fmt.Fprintln(out, "XELYON setup")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Global setup")
	for _, item := range report.Global {
		writeItem(out, item)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Project recommendations")
	for _, item := range report.Project {
		writeItem(out, item)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "XELYON does not store API keys. Use environment variables, `xelyon auth openai-subscription login`, or local Ollama.")
}

// RenderString は setup checklist を文字列で返す。
func RenderString(report Report) string {
	var b strings.Builder
	Render(&b, report)
	return b.String()
}

// ProviderCredentialStatus は provider descriptor と既存 auth state から credential 状態を返す。
func ProviderCredentialStatus(provider string) ProviderStatus {
	provider = setupProviderName(provider)

	entry, ok := config.ProviderCatalogEntryFor(provider)
	if !ok {
		return ProviderStatus{
			Provider: provider,
			Ready:    false,
			Detail:   fmt.Sprintf("unknown provider %q", provider),
		}
	}

	display := provider
	if provider == "openai_subscription" {
		display = "OpenAI Subscription"
	}
	status := ProviderStatus{
		Provider:     provider,
		DisplayName:  display,
		Instructions: append([]string(nil), entry.SetupInstructions...),
	}

	switch provider {
	case "openai_subscription":
		authStatus := openaisubscription.ReadSubscriptionAuthStatus(openaisubscription.DefaultSubscriptionAuthConfig())
		status.Ready = authStatus.AllowsRequestAttempt()
		if authStatus.LoggedIn {
			status.Detail = "logged in"
		} else if strings.TrimSpace(authStatus.Message) != "" {
			status.Detail = authStatus.Message
		} else {
			status.Detail = "login required"
		}
		return status
	case "ollama":
		status.Ready = true
		status.Detail = "local Ollama endpoint; OLLAMA_BASE_URL is optional"
		return status
	case "bedrock":
		status.Ready = true
		status.Detail = "uses AWS credential chain at request time"
		return status
	}

	if config.ProviderHasAvailableCredential(provider) {
		status.Ready = true
		status.Detail = "credential configured"
		return status
	}

	envSets := entry.CredentialEnvVarSets
	if len(envSets) == 0 && entry.APIKeyEnv != "" {
		envSets = [][]string{{entry.APIKeyEnv}}
	}
	if len(envSets) > 0 {
		status.Detail = "missing " + formatEnvSets(envSets)
	} else {
		status.Detail = "credential setup required"
	}
	return status
}

func resolveSetupProviderName(requested, fallback string) string {
	if provider := setupProviderName(requested); provider != "" {
		return provider
	}
	if provider := setupProviderName(fallback); provider != "" {
		return provider
	}
	return "deepseek"
}

func setupProviderName(provider string) string {
	normalized := config.NormalizeProviderName(provider)
	if strings.TrimSpace(normalized) == "" {
		return ""
	}
	canonical := config.CanonicalProviderName(normalized)
	if strings.TrimSpace(canonical) != "" {
		if _, ok := config.ProviderCatalogEntryFor(canonical); ok {
			return canonical
		}
	}
	return normalized
}

// ProviderSetupRequired は provider が request 前 setup を必要としているか返す。
func ProviderSetupRequired(provider string) bool {
	return !ProviderCredentialStatus(provider).Ready
}

// ProviderSetupRequiredMessage は provider 未設定時の短い案内を返す。
func ProviderSetupRequiredMessage(provider string) string {
	status := ProviderCredentialStatus(provider)
	lines := []string{
		fmt.Sprintf("provider setup required for %s: %s", status.Provider, status.Detail),
	}
	for _, instruction := range status.Instructions {
		lines = append(lines, "  "+instruction)
	}
	lines = append(lines, "  xelyon setup")
	return strings.Join(lines, "\n")
}

func defaultModelItem(cfg *config.Config, provider, model string) Item {
	item := Item{
		Key:     "default_model",
		Status:  "ok",
		Message: "Default provider/model",
		Detail:  fmt.Sprintf("%s / %s", provider, model),
	}
	if err := validateSetupDefaultModel(cfg, provider, model); err != nil {
		item.Status = "todo"
		item.Detail = fmt.Sprintf("%s / %s: %s", provider, model, err)
		item.Instruction = defaultModelInstruction(provider)
	}
	return item
}

func validateSetupDefaultModel(cfg *config.Config, provider, model string) error {
	if config.CanonicalProviderName(provider) != "azure" {
		return nil
	}
	providerConfigKey := config.ActiveProviderConfigKey(provider)
	if providerConfigKey == "" {
		providerConfigKey = "azure"
	}
	return config.ValidateAzureDeploymentSelection(cfg, providerConfigKey, model, false)
}

func defaultModelInstruction(provider string) string {
	if config.CanonicalProviderName(provider) == "azure" {
		return "Set provider_models.azure.default_model to your Azure deployment name, or run /provider azure <deployment> in TUI."
	}
	return ""
}

func projectItems(cfg *config.Config, cwd string, instructionMode ProjectConfigInstructionMode) []Item {
	if strings.TrimSpace(cwd) == "" {
		if current, err := getwd(); err == nil {
			cwd = current
		}
	}
	if strings.TrimSpace(cwd) == "" {
		return []Item{{Key: "project_root", Status: "skip", Message: "Project root", Detail: "cwd unavailable"}}
	}

	projectCfg, err := loadProjectConfig(cwd)
	if err != nil {
		return []Item{{Key: "xelyon_yaml", Status: "warn", Message: "xelyon.yaml", Detail: err.Error()}}
	}
	if projectCfg == nil {
		items := []Item{
			{Key: "xelyon_yaml", Status: "todo", Message: "xelyon.yaml", Detail: "not found", Instruction: missingProjectConfigInstruction(instructionMode)},
		}
		if root, ok := resolveProjectRoot(cfg, cwd); ok {
			items = append(items, lspRecommendationItems(cfg, root)...)
		} else {
			items = append(items, Item{Key: "lsp", Status: "skip", Message: "LSP recommendations", Detail: "requires project root"})
		}
		items = append(items, finalChecksItem(cfg, nil))
		return items
	}

	root := filepath.Dir(projectCfg.FilePath)
	items := []Item{
		{Key: "xelyon_yaml", Status: "ok", Message: "xelyon.yaml", Detail: projectCfg.FilePath},
	}
	items = append(items, lspRecommendationItems(cfg, root)...)
	items = append(items, finalChecksItem(cfg, projectCfg))
	return items
}

func missingProjectConfigInstruction(mode ProjectConfigInstructionMode) string {
	if mode == ProjectConfigInstructionTUI {
		return "Run /project in TUI or create xelyon.yaml when XELYON-specific repo settings are needed."
	}
	return "Create xelyon.yaml when XELYON-specific repo settings are needed."
}

func lspRecommendationItems(cfg *config.Config, root string) []Item {
	if cfg == nil || !cfg.LSP.Enabled {
		return []Item{{Key: "lsp", Status: "skip", Message: "LSP recommendations", Detail: "lsp.enabled is false"}}
	}
	if cfg.LSP.SkipInstallPrompt {
		return []Item{{Key: "lsp", Status: "skip", Message: "LSP recommendations", Detail: "lsp.skip_install_prompt is true"}}
	}
	languages, err := detectProjectLanguages(root)
	if err != nil {
		return []Item{{Key: "lsp", Status: "warn", Message: "LSP recommendations", Detail: err.Error()}}
	}
	if len(languages) == 0 {
		return []Item{{Key: "lsp", Status: "ok", Message: "LSP recommendations", Detail: "no configured project languages detected"}}
	}

	var items []Item
	for _, language := range languages {
		serverCfg, ok := cfg.LSP.Servers[language.ServerKey]
		if !ok || serverCfg.Disabled || serverCfg.Command == "" {
			continue
		}
		if _, err := lookPath(serverCfg.Command); err == nil {
			continue
		}
		info, ok := lsp.GetInstallInfo(language.ServerKey)
		if !ok || len(info.Commands) == 0 {
			continue
		}
		items = append(items, Item{
			Key:         "lsp." + language.ServerKey,
			Status:      "todo",
			Message:     "LSP server " + language.ServerKey,
			Detail:      fmt.Sprintf("%d file(s) detected", language.FileCount),
			Instruction: info.Commands[0],
		})
	}
	if len(items) == 0 {
		return []Item{{Key: "lsp", Status: "ok", Message: "LSP recommendations", Detail: "detected LSP servers are installed or disabled"}}
	}
	return items
}

func finalChecksItem(globalCfg *config.Config, projectCfg *config.ProjectConfig) Item {
	finalChecks := config.ResolveFinalChecks(globalCfg, projectCfg)
	if finalChecks == nil || len(finalChecks.Commands) == 0 {
		return Item{
			Key:         "final_checks",
			Status:      "todo",
			Message:     "Final checks",
			Detail:      "not configured",
			Instruction: "Add final_checks.commands to xelyon.yaml or ~/.xelyon/config.yaml.",
		}
	}
	return Item{
		Key:     "final_checks",
		Status:  "ok",
		Message: "Final checks",
		Detail:  strings.Join(finalChecks.Commands, " && "),
	}
}

func toolAvailabilityItem(command, label, instruction string) Item {
	if command == "rg" && common.IsRipgrepAvailable() {
		return Item{Key: command, Status: "ok", Message: label, Detail: "available"}
	}
	if command != "rg" {
		if _, err := lookPath(command); err == nil {
			return Item{Key: command, Status: "ok", Message: label, Detail: "available"}
		}
	}
	return Item{Key: command, Status: "todo", Message: label, Detail: "not found", Instruction: instruction}
}

func writeItem(out io.Writer, item Item) {
	fmt.Fprintf(out, "- [%s] %s", item.Status, item.Message)
	if strings.TrimSpace(item.Detail) != "" {
		fmt.Fprintf(out, ": %s", item.Detail)
	}
	fmt.Fprintln(out)
	if strings.TrimSpace(item.Instruction) != "" {
		fmt.Fprintf(out, "  %s\n", item.Instruction)
	}
}

func statusLabel(ok bool) string {
	if ok {
		return "ok"
	}
	return "todo"
}

func firstInstruction(instructions []string) string {
	for _, instruction := range instructions {
		if strings.TrimSpace(instruction) != "" {
			return instruction
		}
	}
	return ""
}

func formatEnvSets(envSets [][]string) string {
	var parts []string
	for _, envSet := range envSets {
		var names []string
		for _, envName := range envSet {
			if strings.TrimSpace(envName) != "" {
				names = append(names, envName)
			}
		}
		if len(names) > 0 {
			parts = append(parts, strings.Join(names, " + "))
		}
	}
	return strings.Join(parts, " or ")
}
