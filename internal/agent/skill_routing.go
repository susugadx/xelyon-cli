package agent

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/skills/router"
	"github.com/susugadx/xelyon-cli/internal/skills/usageledger"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

const (
	skillRouterSignalTotalTimeout = 750 * time.Millisecond
	skillRouterGitStatusTimeout   = 500 * time.Millisecond
	skillRouterTouchedPathCap     = 200
)

type skillRouterInputOptions struct {
	command       string
	requestedMode string
	readOnly      bool
}

func (a *Agent) skillRouterInput(ctx context.Context, taskText, command string) router.Input {
	return a.skillRouterInputWithOptions(ctx, taskText, skillRouterInputOptions{command: command})
}

func (a *Agent) skillRouterInputWithOptions(ctx context.Context, taskText string, opts skillRouterInputOptions) router.Input {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, skillRouterSignalTotalTimeout)
	defer cancel()

	input := router.Input{
		TaskText:              taskText,
		Command:               opts.command,
		RequestedMode:         opts.requestedMode,
		ReadOnly:              opts.readOnly,
		PromptCatalogMaxItems: promptSkillCatalogMaxEntries,
	}
	paths, diagnostics := collectSkillRouterTouchedPaths(ctx, a.invocationCWD())
	input.TouchedPaths = paths
	input.SignalDiagnostics = diagnostics
	return input
}

func collectSkillRouterTouchedPaths(ctx context.Context, cwd string) ([]string, []string) {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return nil, []string{"cwd unavailable; skipped git status signals"}
	}
	gitCtx, cancel := context.WithTimeout(ctx, skillRouterGitStatusTimeout)
	defer cancel()
	cmd := exec.CommandContext(gitCtx, "git", "-C", cwd, "status", "--porcelain")
	out, err := cmd.Output()
	if gitCtx.Err() != nil {
		return nil, []string{"git status timed out; using partial routing signals"}
	}
	if err != nil {
		return nil, []string{"git status unavailable; using task text and catalog metadata only"}
	}
	paths, capped := parseSkillRouterStatusPaths(string(out), skillRouterTouchedPathCap)
	if capped {
		return paths, []string{"touched paths capped at 200; routing used partial path signals"}
	}
	return paths, nil
}

func parseSkillRouterStatusPaths(output string, cap int) ([]string, bool) {
	if cap <= 0 {
		return nil, false
	}
	seen := map[string]struct{}{}
	paths := make([]string, 0)
	capped := false
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if path == "" {
			continue
		}
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = strings.TrimSpace(parts[len(parts)-1])
		}
		path = filepath.ToSlash(path)
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		if len(paths) >= cap {
			capped = true
			break
		}
		paths = append(paths, path)
	}
	return paths, capped
}

func (a *Agent) skillRouterProjectRoot() string {
	root, _ := a.skillRouterProjectRootOK()
	return root
}

func (a *Agent) skillRouterProjectRootOK() (string, bool) {
	if a == nil {
		return "", false
	}
	if root, ok := usageledger.ResolveProjectRoot(usageledger.ProjectRootOptions{
		Config:             a.cfg(),
		ProjectMapRootPath: a.projectMapRootPath,
		InvocationCWD:      a.invocationCWD(),
	}); ok {
		return root, true
	}
	return "", false
}

func (a *Agent) skillUsageLedgerStoreForProject(enabled bool) (*usageledger.Store, bool) {
	cfg := a.cfg()
	projectRoot, ok := a.skillRouterProjectRootOK()
	if !ok {
		return nil, false
	}
	retentionDays := config.EffectiveSkillsRouterUsageRetentionDays(cfg)
	return usageledger.NewStore(usageledger.Options{
		ProjectRoot:   projectRoot,
		Enabled:       enabled,
		RetentionDays: retentionDays,
	}), true
}

func (a *Agent) skillUsageLedgerStoreForAll(enabled bool) *usageledger.Store {
	cfg := a.cfg()
	retentionDays := config.EffectiveSkillsRouterUsageRetentionDays(cfg)
	return usageledger.NewStore(usageledger.Options{
		Enabled:       enabled,
		RetentionDays: retentionDays,
	})
}

func (a *Agent) recordSkillRecommendation(rec router.Recommendation) {
	cfg := a.cfg()
	if cfg == nil || !cfg.Skills.Router.UsageLedger {
		return
	}
	summaries := usageSummariesFromRecommendation(rec, router.DefaultRuntimeHintLimits())
	if len(summaries) == 0 {
		return
	}
	store, ok := a.skillUsageLedgerStoreForProject(true)
	if !ok {
		return
	}
	_ = store.Append(usageledger.Record{
		Type:        "recommendation",
		Recommended: summaries,
		Policy: usageledger.PolicySnapshot{
			Enabled:         cfg.Skills.Router.Enabled,
			Activation:      string(config.NormalizeSkillsRouterActivation(cfg.Skills.Router.Activation)),
			PrimaryLimit:    router.DefaultRuntimeHintLimits().Primary,
			SupportingLimit: router.DefaultRuntimeHintLimits().Supporting,
			ConflictLimit:   router.DefaultRuntimeHintLimits().Conflict,
			MaybeLimit:      router.DefaultRuntimeHintLimits().Maybe,
		},
	})
}

func (a *Agent) recordSkillActivationFromToolResult(toolCall *tools.ToolCall, result string, resultErr bool) {
	if a == nil || toolCall == nil || toolCall.Tool != "activate_skill" {
		return
	}
	if resultErr || tools.IsErrorResult(result) {
		return
	}
	name := strings.TrimSpace(toolCall.Args["name"])
	if name == "" {
		return
	}
	a.recordSkillActivation(name)
}

func (a *Agent) recordSkillActivation(name string) {
	cfg := a.cfg()
	if cfg == nil || !cfg.Skills.Router.UsageLedger {
		return
	}
	store, ok := a.skillUsageLedgerStoreForProject(true)
	if !ok {
		return
	}
	_ = store.Append(usageledger.Record{
		Type:      "activation",
		Activated: []string{name},
		Policy: usageledger.PolicySnapshot{
			Enabled:    cfg.Skills.Router.Enabled,
			Activation: string(config.NormalizeSkillsRouterActivation(cfg.Skills.Router.Activation)),
		},
	})
}

func (a *Agent) normalModeSystemPromptForRequest(ctx context.Context, taskText string, recordUsage bool) string {
	return a.normalModeSystemPromptForRequestWithDirectives(ctx, taskText, recordUsage, a.pendingRuntimeDirectives())
}

func (a *Agent) normalModeSystemPromptForRequestWithDirectives(ctx context.Context, taskText string, recordUsage bool, runtimeDirectives []string) string {
	effectivePrompt := prompt.StripPlanningReferences(a.SystemPrompt)
	effectivePrompt = a.injectSkillRouterRuntimeHintWithOptions(ctx, effectivePrompt, taskText, skillRouterInputOptions{command: "chat"}, recordUsage)
	directives := append([]string{normalModeBaseRuntimeDirective}, runtimeDirectives...)
	return appendRuntimeDirectivesToSystemPrompt(effectivePrompt, directives...)
}

func (a *Agent) planModeSystemPromptForInvestigationRequest(ctx context.Context, taskText string, recordUsage bool) string {
	return a.injectSkillRouterRuntimeHintWithOptions(ctx, a.SystemPrompt, taskText, skillRouterInputOptions{
		command:       "/plan",
		requestedMode: "plan",
	}, recordUsage)
}

func usageSummariesFromRecommendation(rec router.Recommendation, limits router.HintLimits) []usageledger.SkillSummary {
	var summaries []usageledger.SkillSummary
	appendCandidates := func(candidates []router.Candidate, limit int) {
		if limit <= 0 {
			return
		}
		count := 0
		for _, candidate := range candidates {
			if candidate.Confidence != router.ConfidenceHigh && candidate.Confidence != router.ConfidenceMedium {
				continue
			}
			summaries = append(summaries, usageledger.SkillSummary{
				Name:       candidate.Name,
				Category:   string(candidate.Category),
				Score:      candidate.Score,
				Confidence: string(candidate.Confidence),
				Activation: string(candidate.Activation),
			})
			count++
			if count >= limit {
				return
			}
		}
	}
	appendCandidates(rec.Primary, limits.Primary)
	appendCandidates(rec.Supporting, limits.Supporting)
	return summaries
}

func (a *Agent) injectSkillRouterRuntimeHint(ctx context.Context, systemPrompt, taskText string, recordUsage bool) string {
	return a.injectSkillRouterRuntimeHintWithOptions(ctx, systemPrompt, taskText, skillRouterInputOptions{command: "chat"}, recordUsage)
}

func (a *Agent) injectSkillRouterRuntimeHintWithOptions(ctx context.Context, systemPrompt, taskText string, opts skillRouterInputOptions, recordUsage bool) string {
	cfg := a.cfg()
	if !config.SkillsRouterRuntimeHintEnabled(cfg) {
		return stripSkillRouterRuntimeHintFromSystemPrompt(systemPrompt)
	}
	catalog := a.loadSkillCatalog()
	input := a.skillRouterInputWithOptions(ctx, taskText, opts)
	rec := router.Recommend(catalog, input)
	if recordUsage {
		a.recordSkillRecommendation(rec)
	}
	return injectSkillRouterRuntimeHintIntoSystemPrompt(systemPrompt, rec, router.DefaultRuntimeHintLimits(), cfg)
}

func injectSkillRouterRuntimeHintIntoSystemPrompt(systemPrompt string, rec router.Recommendation, limits router.HintLimits, cfg *config.Config) string {
	base := router.StripRuntimeHint(systemPrompt)
	block := router.FormatRuntimeHint(rec, limits)
	if strings.TrimSpace(block) == "" {
		return normalizeSkillRouterRuntimeHintBase(base)
	}
	if skillRouterRuntimeHintShouldUseDynamicLayout(base, cfg) {
		layout := api.SplitSystemPromptLayout(base)
		layout.AppendDynamic(block)
		return layout.Compose()
	}
	return strings.TrimRight(base, "\n") + "\n\n" + block
}

func stripSkillRouterRuntimeHintFromSystemPrompt(systemPrompt string) string {
	return normalizeSkillRouterRuntimeHintBase(router.StripRuntimeHint(systemPrompt))
}

func normalizeSkillRouterRuntimeHintBase(systemPrompt string) string {
	if strings.Contains(systemPrompt, api.SystemPromptCacheBoundary) {
		return api.SplitSystemPromptLayout(systemPrompt).Compose()
	}
	return strings.TrimRight(systemPrompt, "\n")
}

func skillRouterRuntimeHintShouldUseDynamicLayout(systemPrompt string, cfg *config.Config) bool {
	return strings.Contains(systemPrompt, api.SystemPromptCacheBoundary) || cfg == nil || cfg.PromptCache.Enabled
}
