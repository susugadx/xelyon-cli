package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/token"
)

type projectInstructionApplyOptions struct {
	showStatus       bool
	injectProjectMap bool
	projectMapInput  string
}

func loadProjectInstructionBundleForCWD(cfg *config.Config, cwd string) *config.ProjectInstructionBundle {
	bundle, _ := loadProjectInstructionBundleForCWDWithError(cfg, cwd)
	return bundle
}

func loadProjectInstructionBundleForCWDWithError(cfg *config.Config, cwd string) (*config.ProjectInstructionBundle, error) {
	return config.LoadProjectInstructionBundleForDir(cfg, cwd)
}

func loadProjectInstructionBundleForCWDAndInputPathsWithError(cfg *config.Config, cwd string, inputPaths []string) (*config.ProjectInstructionBundle, error) {
	return config.LoadProjectInstructionBundleForDirWithInputPaths(cfg, cwd, inputPaths)
}

func (a *Agent) loadProjectInstructionBundleCached(forceReload bool) *config.ProjectInstructionBundle {
	bundle, _ := a.loadProjectInstructionBundleCachedWithError(forceReload)
	return bundle
}

func (a *Agent) loadProjectInstructionBundleCachedWithError(forceReload bool) (*config.ProjectInstructionBundle, error) {
	return a.loadProjectInstructionBundleCachedWithInputWithError(forceReload, "")
}

func (a *Agent) loadProjectInstructionBundleCachedWithInput(forceReload bool, input string) *config.ProjectInstructionBundle {
	bundle, _ := a.loadProjectInstructionBundleCachedWithInputWithError(forceReload, input)
	return bundle
}

func (a *Agent) loadProjectInstructionBundleCachedWithInputWithError(forceReload bool, input string) (*config.ProjectInstructionBundle, error) {
	if a == nil {
		return nil, nil
	}
	cache := newProjectInstructionBundleCache(a)
	if decision := cache.decision(forceReload, input); decision.reuse {
		return a.projectInstructionBundle, nil
	} else {
		inputPaths := projectInstructionInputPathsForAgent(a, input)
		bundle, err := loadProjectInstructionBundleForCWDAndInputPathsWithError(a.cfg(), a.invocationCWD(), inputPaths)
		if err != nil {
			return nil, err
		}
		a.projectInstructionBundle = bundle
		a.projectInstructionBundleLoaded = true
		a.projectInstructionBundleKey = cache.currentKey(input)
		return bundle, nil
	}
}

func (a *Agent) invalidateProjectInstructionBundleCache() {
	newProjectInstructionBundleCache(a).invalidate()
}

func (a *Agent) projectInstructionBundleIfLoaded() *config.ProjectInstructionBundle {
	if a == nil || !a.projectInstructionBundleLoaded {
		return nil
	}
	return a.projectInstructionBundle
}

func buildProjectInstructionBlock(bundle *config.ProjectInstructionBundle) string {
	if bundle == nil {
		return ""
	}

	return prompt.BuildProjectInstructionBlock(prompt.ProjectInstructionBlockInput{
		ProjectGuidance: toPromptInstructionEntries(bundle.ProjectGuidance),
		GlobalGuidance:  toPromptInstructionEntries(bundle.GlobalGuidance),
		Warnings:        bundle.WarningMessages(),
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
			Scope:    file.RepositoryScope,
			Source:   file.Label,
			Content:  file.Content,
			Strength: string(file.Strength),
		})
	}
	return entries
}

// injectProjectInstructionBundle は bundle を SystemPrompt に注入する。
func injectProjectInstructionBundle(systemPrompt string, bundle *config.ProjectInstructionBundle) string {
	systemPrompt = prompt.StripProjectConfigSections(systemPrompt)
	if bundle == nil {
		return systemPrompt
	}

	projectBlock := buildProjectInstructionBlock(bundle)
	if projectBlock == "" {
		return systemPrompt
	}
	return prompt.InjectProjectConfigBlock(systemPrompt, projectBlock)
}

// initializeProjectInstructions はプロジェクト instruction 初期化を行う統一ヘルパー。
// bundle 読み込み + SystemPrompt 注入 + runtime 設定同期 + UI 表示 + project map 注入を行う。
func initializeProjectInstructions(agent *Agent, opts projectInstructionApplyOptions) error {
	if agent == nil {
		return nil
	}
	pm := agent.promptManager()
	if pm == nil {
		return nil
	}
	return pm.InitializeProjectInstructions(opts)
}

// applyProjectInstructionBundle は bundle を runtime 設定 / 表示へ適用する。
func applyProjectInstructionBundle(agent *Agent, bundle *config.ProjectInstructionBundle, showStatus bool) error {
	if bundle == nil {
		return nil
	}

	if err := syncProviderHistoryRuntimeConfigFromProjectConfig(agent.Runtime, bundle.ProjectConfig); err != nil {
		return err
	}
	applyFinalChecksFromBundle(agent, bundle)

	if !showStatus {
		return nil
	}

	renderProjectInstructionStatus(agent, bundle)
	return nil
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
	if labels := joinInstructionLabels(bundle.ProjectGuidance, bundle.ProjectGuidanceStatus); labels != "" {
		green.Fprintf(agent.output(), "📋 Project guidance loaded: %s\n", labels)
	}
	if labels := joinInstructionLabels(bundle.GlobalGuidance, bundle.GlobalGuidanceStatus); labels != "" {
		green.Fprintf(agent.output(), "📋 Global guidance loaded: %s\n", labels)
	}
	for _, warning := range bundle.WarningMessages() {
		if strings.TrimSpace(warning) == "" {
			continue
		}
		yellow.Fprintf(agent.output(), "⚠️  %s\n", warning)
	}
}

func joinInstructionLabels(files []config.InstructionFile, statuses []config.InstructionFileStatus) string {
	if len(files) == 0 && len(statuses) == 0 {
		return ""
	}
	labels := make([]string, 0, len(files)+len(statuses))
	seen := make(map[string]struct{}, len(files)+len(statuses))
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
	for _, status := range statuses {
		label := strings.TrimSpace(status.Label)
		if label == "" {
			continue
		}
		switch status.Status {
		case config.InstructionFileStatusEmpty:
			label += " (empty)"
		default:
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
	return token.EstimateTokenCount(buildProjectInstructionBlock(bundle))
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
