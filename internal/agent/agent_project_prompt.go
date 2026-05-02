package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

func loadProjectInstructionBundle(cfg *config.Config) *config.ProjectInstructionBundle {
	bundle, err := config.LoadProjectInstructionBundle(cfg)
	if err != nil {
		return nil
	}
	return bundle
}

func buildProjectInstructionBlock(bundle *config.ProjectInstructionBundle, input string) string {
	if bundle == nil {
		return ""
	}

	selection := config.ProjectPromptSelection{}
	if bundle.ProjectConfig != nil {
		selection = config.SelectProjectPromptSelection(bundle.ProjectConfig, input)
	}

	projectGuidance := make([]prompt.ProjectInstructionEntry, 0, len(bundle.ProjectGuidance))
	for _, file := range bundle.ProjectGuidance {
		projectGuidance = append(projectGuidance, prompt.ProjectInstructionEntry{
			Label:    file.Label,
			Content:  file.Content,
			Strength: string(file.Strength),
		})
	}

	globalGuidance := make([]prompt.ProjectInstructionEntry, 0, len(bundle.GlobalGuidance))
	for _, file := range bundle.GlobalGuidance {
		globalGuidance = append(globalGuidance, prompt.ProjectInstructionEntry{
			Label:    file.Label,
			Content:  file.Content,
			Strength: string(file.Strength),
		})
	}

	return prompt.BuildProjectInstructionBlock(prompt.ProjectInstructionBlockInput{
		HasProjectConfig: bundle.ProjectConfig != nil,
		MandatoryRules:   selection.Rules,
		ProjectContexts:  selection.Contexts,
		ProjectGuidance:  projectGuidance,
		GlobalGuidance:   globalGuidance,
	})
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

// applyProjectInstructionBundle はプロジェクト instruction をエージェントに適用する統一ヘルパー。
// SystemPrompt 注入 + final checks 解決 + UI 表示を行う。
func applyProjectInstructionBundle(agent *Agent, bundle *config.ProjectInstructionBundle) {
	if bundle == nil {
		return
	}

	// 1. System prompt 注入
	agent.SystemPrompt = injectProjectInstructionBundle(agent.SystemPrompt, bundle, "")

	// 2. final checks 解決（xelyon.yaml があるときだけ project override を反映）
	if bundle.ProjectConfig != nil {
		if resolved := config.ResolveFinalChecks(agent.cfg(), bundle.ProjectConfig); resolved != nil {
			cfg := agent.cfg()
			cfg.FinalChecks = *resolved
		}
	}

	// 3. UI 表示
	if bundle.ProjectConfig != nil {
		green.Fprintln(agent.output(), "📋 xelyon.yaml loaded")
	}
	if labels := joinInstructionLabels(bundle.ProjectGuidance); labels != "" {
		green.Fprintf(agent.output(), "📋 Project guidance loaded: %s\n", labels)
	}
	if labels := joinInstructionLabels(bundle.GlobalGuidance); labels != "" {
		green.Fprintf(agent.output(), "📋 Global guidance loaded: %s\n", labels)
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
