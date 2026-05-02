package agent

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

type projectInstructionApplyOptions struct {
	showStatus       bool
	injectProjectMap bool
	projectMapInput  string
}

func loadProjectInstructionBundle(cfg *config.Config) *config.ProjectInstructionBundle {
	bundle, err := config.LoadProjectInstructionBundle(cfg)
	if err != nil {
		return nil
	}
	return bundle
}

func (a *Agent) loadProjectInstructionBundleCached(forceReload bool) *config.ProjectInstructionBundle {
	if a == nil {
		return nil
	}
	if decision := a.projectInstructionBundleCacheDecision(forceReload); decision.reuse {
		return a.projectInstructionBundle
	} else {
		bundle := loadProjectInstructionBundle(a.cfg())
		a.projectInstructionBundle = bundle
		a.projectInstructionBundleLoaded = true
		if decision.cacheKey != "" {
			a.projectInstructionBundleKey = decision.cacheKey
		} else {
			a.projectInstructionBundleKey = a.currentProjectInstructionBundleCacheKey()
		}
		return bundle
	}
}

type projectInstructionBundleCacheDecision struct {
	reuse    bool
	cacheKey string
}

func (a *Agent) projectInstructionBundleCacheDecision(forceReload bool) projectInstructionBundleCacheDecision {
	if a == nil || forceReload || !a.projectInstructionBundleLoaded {
		return projectInstructionBundleCacheDecision{}
	}
	cacheKey := a.currentProjectInstructionBundleCacheKey()
	return projectInstructionBundleCacheDecision{
		reuse:    cacheKey == a.projectInstructionBundleKey,
		cacheKey: cacheKey,
	}
}

func (a *Agent) invalidateProjectInstructionBundleCache() {
	if a == nil {
		return
	}
	a.projectInstructionBundle = nil
	a.projectInstructionBundleLoaded = false
	a.projectInstructionBundleKey = ""
}

func (a *Agent) currentProjectInstructionBundleCacheKey() string {
	if a == nil {
		return ""
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return config.ComputeProjectInstructionBundleFingerprintForDir(a.cfg(), cwd, a.projectInstructionBundle)
}

func (a *Agent) projectInstructionBundleIfLoaded() *config.ProjectInstructionBundle {
	if a == nil || !a.projectInstructionBundleLoaded {
		return nil
	}
	return a.projectInstructionBundle
}

func buildProjectInstructionBlock(bundle *config.ProjectInstructionBundle, input string) string {
	if bundle == nil {
		return ""
	}

	selection := config.ProjectPromptSelection{}
	if bundle.ProjectConfig != nil {
		selection = config.SelectProjectPromptSelection(bundle.ProjectConfig, input)
	}

	return prompt.BuildProjectInstructionBlock(prompt.ProjectInstructionBlockInput{
		HasProjectConfig: bundle.ProjectConfig != nil,
		MandatoryRules:   selection.Rules,
		ProjectContexts:  selection.Contexts,
		ProjectGuidance:  toPromptInstructionEntries(bundle.ProjectGuidance),
		GlobalGuidance:   toPromptInstructionEntries(bundle.GlobalGuidance),
		Warnings:         bundle.WarningMessages(),
	})
}

func toPromptInstructionEntries(files []config.InstructionFile) []prompt.ProjectInstructionEntry {
	if len(files) == 0 {
		return nil
	}
	entries := make([]prompt.ProjectInstructionEntry, 0, len(files))
	for _, file := range files {
		entries = append(entries, prompt.ProjectInstructionEntry{
			Label:    file.Label,
			Content:  file.Content,
			Strength: string(file.Strength),
		})
	}
	return entries
}

// injectProjectInstructionBundle は bundle を SystemPrompt に注入する。
func injectProjectInstructionBundle(systemPrompt string, bundle *config.ProjectInstructionBundle, input string) string {
	systemPrompt = prompt.StripProjectConfigSections(systemPrompt)
	if bundle == nil {
		return systemPrompt
	}

	projectBlock := buildProjectInstructionBlock(bundle, input)
	if projectBlock == "" {
		return systemPrompt
	}
	return prompt.InjectProjectConfigBlock(systemPrompt, projectBlock)
}

// initializeProjectInstructions はプロジェクト instruction 初期化を行う統一ヘルパー。
// bundle 読み込み + SystemPrompt 注入 + final checks 解決 + UI 表示 + project map 注入を行う。
func initializeProjectInstructions(agent *Agent, opts projectInstructionApplyOptions) {
	if agent == nil {
		return
	}
	pm := agent.promptManager()
	if pm == nil {
		return
	}
	pm.InitializeProjectInstructions(opts)
}

// applyProjectInstructionBundle は bundle を final checks / 表示へ適用する。
func applyProjectInstructionBundle(agent *Agent, bundle *config.ProjectInstructionBundle, showStatus bool) {
	if bundle == nil {
		return
	}

	applyFinalChecksFromBundle(agent, bundle)

	if !showStatus {
		return
	}

	renderProjectInstructionStatus(agent, bundle)
}

func applyFinalChecksFromBundle(agent *Agent, bundle *config.ProjectInstructionBundle) {
	if agent == nil || bundle == nil || bundle.ProjectConfig == nil {
		return
	}
	if resolved := config.ResolveFinalChecks(agent.cfg(), bundle.ProjectConfig); resolved != nil {
		cfg := agent.cfg()
		cfg.FinalChecks = *resolved
	}
}

func renderProjectInstructionStatus(agent *Agent, bundle *config.ProjectInstructionBundle) {
	if agent == nil || bundle == nil {
		return
	}
	if bundle.ProjectConfig != nil {
		green.Fprintln(agent.output(), "📋 xelyon.yaml loaded")
	}
	if labels := joinInstructionLabels(bundle.ProjectGuidance); labels != "" {
		green.Fprintf(agent.output(), "📋 Project guidance loaded: %s\n", labels)
	}
	if labels := joinInstructionLabels(bundle.GlobalGuidance); labels != "" {
		green.Fprintf(agent.output(), "📋 Global guidance loaded: %s\n", labels)
	}
	for _, warning := range bundle.WarningMessages() {
		if strings.TrimSpace(warning) == "" {
			continue
		}
		yellow.Fprintf(agent.output(), "⚠️  %s\n", warning)
	}
}

func joinInstructionLabels(files []config.InstructionFile) string {
	if len(files) == 0 {
		return ""
	}
	labels := make([]string, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		label := strings.TrimSpace(file.Label)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}
	return strings.Join(labels, ", ")
}

// rebuildSystemPromptForCurrentProvider は現在の provider/model に合わせて
// SystemPrompt をベースから再構築する。
func (a *Agent) rebuildSystemPromptForCurrentProvider() {
	a.promptManager().RebuildSystemPromptForCurrentProvider()
}

func estimateProjectInstructionTokens(bundle *config.ProjectInstructionBundle) int {
	if bundle == nil {
		return 0
	}
	return token.EstimateTokenCount(buildProjectInstructionBlock(bundle, ""))
}

func (a *Agent) refreshProjectPrompt(input string) {
	a.promptManager().RefreshProjectPrompt(input)
}

func (a *Agent) refreshProjectPromptIfDirty(input string) {
	a.promptManager().RefreshProjectPromptIfDirty(input)
}

func (a *Agent) shouldRefreshProjectPrompt(input string) bool {
	return a.promptManager().ShouldRefreshProjectPrompt(input)
}
