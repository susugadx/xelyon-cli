package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
)

func (m *PromptManager) RebuildSystemPromptForCurrentProvider() {
	m.RebuildSystemPrompt(systemPromptRebuildRequest{
		mode:             systemPromptRebuildModeBase,
		bundleMode:       systemPromptBundleResolveForced,
		injectProjectMap: true,
	})
}

func (m *PromptManager) InitializeProjectInstructions(opts projectInstructionApplyOptions) error {
	a := m.agent
	if a == nil {
		return nil
	}
	bundle, err := a.loadProjectInstructionBundleCachedWithError(true)
	if err != nil {
		return err
	}
	if err := applyProjectInstructionBundle(a, bundle, opts.showStatus); err != nil {
		return err
	}
	m.rebuildProjectPromptWithResolvedBundle(opts.projectMapInput, bundle, opts.injectProjectMap)
	return nil
}

func (m *PromptManager) RebuildSystemPrompt(req systemPromptRebuildRequest) {
	a := m.agent
	if a == nil {
		return
	}
	if m.shouldSkipRebuild(req) {
		return
	}

	m.rebuildStaticPrompt(req)
	bundle := m.resolveBundleForRebuild(req)
	m.applyDynamicPromptSections(req, bundle)
}

func (m *PromptManager) shouldSkipRebuild(req systemPromptRebuildRequest) bool {
	a := m.agent
	return a == nil || (req.mode == systemPromptRebuildModeBase && a.CurrentProvider == nil)
}

func (m *PromptManager) rebuildStaticPrompt(req systemPromptRebuildRequest) {
	a := m.agent
	if a == nil {
		return
	}
	switch req.mode {
	case systemPromptRebuildModeBase:
		a.SystemPrompt = m.buildBaseSystemPromptForCurrentProvider()
	case systemPromptRebuildModeProjectOnly:
		m.rebuildProjectOnlyStaticPrompt()
	default:
		a.SystemPrompt = m.buildBaseSystemPromptForCurrentProvider()
	}
}

func (m *PromptManager) applyDynamicPromptSections(req systemPromptRebuildRequest, bundle *config.ProjectInstructionBundle) {
	a := m.agent
	if a == nil {
		return
	}
	if bundle != nil {
		m.InjectProjectInstructions(bundle, req.input)
	}
	if req.injectProjectMap {
		injectProjectMap(a, req.input)
	}
}

func (m *PromptManager) resolveBundleForRebuild(req systemPromptRebuildRequest) *config.ProjectInstructionBundle {
	a := m.agent
	if a == nil {
		return nil
	}
	switch req.bundleMode {
	case systemPromptBundleResolveGiven:
		return req.bundle
	case systemPromptBundleResolveForced:
		return a.loadProjectInstructionBundleCached(true)
	default:
		return a.loadProjectInstructionBundleCached(false)
	}
}

func (m *PromptManager) buildBaseSystemPromptForCurrentProvider() string {
	a := m.agent
	if a == nil || a.CurrentProvider == nil {
		return ""
	}

	planningPrompt := promptplan.BuildPlanningPrompt()
	hadPlanPrompt := strings.Contains(a.SystemPrompt, planningPrompt)
	systemPrompt := m.buildStaticPrompt(promptStaticBuildInput{
		invocationCWD: a.invocationCWD(),
	})

	if hadPlanPrompt {
		systemPrompt += api.SystemPromptCacheBoundary + planningPrompt
	}

	return systemPrompt
}

func (m *PromptManager) rebuildProjectPromptWithResolvedBundle(input string, bundle *config.ProjectInstructionBundle, injectProjectMap bool) {
	m.RebuildSystemPrompt(systemPromptRebuildRequest{
		mode:             systemPromptRebuildModeProjectOnly,
		input:            input,
		bundle:           bundle,
		bundleMode:       systemPromptBundleResolveGiven,
		injectProjectMap: injectProjectMap,
	})
}

// InjectProjectInstructions は現在の SystemPrompt に project instructions を注入する。
func (m *PromptManager) InjectProjectInstructions(bundle *config.ProjectInstructionBundle, input string) {
	a := m.agent
	if a == nil {
		return
	}
	a.SystemPrompt = injectProjectInstructionBundle(a.SystemPrompt, bundle, input)
}

// StripProjectMapSection は SystemPrompt から project map セクションを除去する。
func (m *PromptManager) StripProjectMapSection() {
	a := m.agent
	if a == nil {
		return
	}
	a.SystemPrompt = stripProjectMapSection(a.SystemPrompt)
}

// AppendProjectMapSection は SystemPrompt に project map セクションを追加する。
func (m *PromptManager) AppendProjectMapSection(section string) {
	a := m.agent
	if a == nil {
		return
	}
	a.SystemPrompt = appendProjectMapSection(a.SystemPrompt, section)
}
