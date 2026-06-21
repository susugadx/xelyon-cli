package config

import (
	"fmt"
	"os"
	"strings"
)

// ProjectInstructionBundle は project instruction 注入用の統合データ。
// xelyon.yaml の structured config と AGENTS/CLAUDE guidance を分離して保持する。
type ProjectInstructionBundle struct {
	RootPath   string
	RootSource ProjectInstructionRootSource

	ProjectConfig *ProjectConfig

	ProjectGuidance []InstructionFile
	GlobalGuidance  []InstructionFile

	ProjectGuidanceStatus []InstructionFileStatus
	GlobalGuidanceStatus  []InstructionFileStatus

	WarningEntries []ProjectInstructionWarning
}

// ProjectInstructionRootSource は RootPath を決めた根拠を表す。
type ProjectInstructionRootSource string

const (
	// ProjectInstructionRootSourceProjectConfig は xelyon.yaml の場所を root とした状態。
	ProjectInstructionRootSourceProjectConfig ProjectInstructionRootSource = "project_config"
	// ProjectInstructionRootSourceGit は Git repository root を root とした状態。
	ProjectInstructionRootSourceGit ProjectInstructionRootSource = "git"
	// ProjectInstructionRootSourceGuidance は project guidance ファイルの場所を root とした状態。
	ProjectInstructionRootSourceGuidance ProjectInstructionRootSource = "guidance"
	// ProjectInstructionRootSourceFallbackCWD は project root 根拠がなく cwd に fallback した状態。
	ProjectInstructionRootSourceFallbackCWD ProjectInstructionRootSource = "fallback_cwd"
)

// HasProjectRoot は bundle が明示的な project root 根拠を持つかを返す。
func (b *ProjectInstructionBundle) HasProjectRoot() bool {
	if b == nil || strings.TrimSpace(b.RootPath) == "" {
		return false
	}
	return b.RootSource.hasProjectRoot()
}

// ProjectRootPath は project-level scan に使ってよい root path を返す。
func (b *ProjectInstructionBundle) ProjectRootPath() (string, bool) {
	if !b.HasProjectRoot() {
		return "", false
	}
	return b.RootPath, true
}

func (s ProjectInstructionRootSource) hasProjectRoot() bool {
	switch s {
	case ProjectInstructionRootSourceProjectConfig, ProjectInstructionRootSourceGit, ProjectInstructionRootSourceGuidance:
		return true
	default:
		return false
	}
}

// ProjectInstructionWarningCode は guidance 読み込み時の warning 種別。
type ProjectInstructionWarningCode string

const (
	ProjectInstructionWarningInvalidProjectGuidancePath ProjectInstructionWarningCode = "invalid_project_guidance_path"
	ProjectInstructionWarningLoadSkipped                ProjectInstructionWarningCode = "load_skipped"
	ProjectInstructionWarningImportLoadSkipped          ProjectInstructionWarningCode = "import_load_skipped"
	ProjectInstructionWarningDeprecatedFallbackMode     ProjectInstructionWarningCode = "deprecated_project_mode_fallback"
)

const projectModeFallbackDeprecationMessage = "agent_instructions.project.mode=fallback is deprecated; XELYON now treats it as AGENTS-first guidance loading. Use mode=always to keep this behavior or mode=off to disable project guidance."

// ProjectInstructionWarning は guidance 読み込み時の型付き warning。
type ProjectInstructionWarning struct {
	Code    ProjectInstructionWarningCode
	Message string
}

// InstructionStrength は guidance の優先度カテゴリ。
type InstructionStrength string

const (
	InstructionStrengthProjectGuidance InstructionStrength = "project_guidance"
	InstructionStrengthAdvisory        InstructionStrength = "advisory"
)

// InstructionFile は読み込んだ guidance ファイルの内容。
type InstructionFile struct {
	Path            string
	Label           string
	Scope           string
	RepositoryScope string
	Strength        InstructionStrength
	Content         string
	Truncated       bool
	GitTracked      bool
}

// InstructionFileStatus は prompt に注入しない guidance 候補の状態表示用情報。
type InstructionFileStatus struct {
	Path   string
	Label  string
	Scope  string
	Status InstructionFileStatusKind
}

// InstructionFileStatusKind は guidance 候補の状態種別。
type InstructionFileStatusKind string

const (
	// InstructionFileStatusEmpty は guidance ファイルが存在するが空である状態。
	InstructionFileStatusEmpty InstructionFileStatusKind = "empty"
)

// LoadProjectInstructionBundle は現在 cwd を基準に instruction bundle を解決する。
func LoadProjectInstructionBundle(cfg *Config) (*ProjectInstructionBundle, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return LoadProjectInstructionBundleForDir(cfg, cwd)
}

// LoadProjectInstructionBundleForDir は指定ディレクトリを基準に instruction bundle を解決する。
func LoadProjectInstructionBundleForDir(cfg *Config, cwd string) (*ProjectInstructionBundle, error) {
	return LoadProjectInstructionBundleForDirWithInputPaths(cfg, cwd, nil)
}

// LoadProjectInstructionBundleForDirWithInputPaths は指定ディレクトリと
// 入力から参照された repo-relative path を基準に instruction bundle を解決する。
func LoadProjectInstructionBundleForDirWithInputPaths(cfg *Config, cwd string, inputPaths []string) (*ProjectInstructionBundle, error) {
	if strings.TrimSpace(cwd) == "" {
		return nil, fmt.Errorf("cwd is empty")
	}

	cfgForLoad := cfg
	if cfgForLoad == nil {
		cfgForLoad = DefaultConfig()
	}

	projectCfg, err := loadProjectConfigForDir(cwd)
	if err != nil {
		return nil, err
	}

	gitRoot := findGitRoot(cwd)
	root := resolveBundleRoot(cwd, projectCfg, gitRoot, cfgForLoad.AgentInstructions)
	bundle := &ProjectInstructionBundle{
		ProjectConfig: projectCfg,
		RootPath:      root.RootPath,
		RootSource:    root.Source,
	}

	mode := normalizeAgentInstructionProjectMode(cfgForLoad.AgentInstructions.Project.Mode)
	budget := newInstructionByteBudget(cfgForLoad.AgentInstructions)
	appendDeprecatedProjectModeFallbackWarning(bundle, mode)

	if shouldLoadProjectGuidance(mode) {
		strength := resolveProjectGuidanceStrength()
		bundle.ProjectGuidance = loadProjectGuidanceFiles(bundle, cfgForLoad.AgentInstructions, gitRoot, strength, &budget, cwd, inputPaths)
	}

	if cfgForLoad.AgentInstructions.Global.Enabled {
		bundle.GlobalGuidance = loadGlobalGuidanceFiles(bundle, cfgForLoad.AgentInstructions, &budget)
	}

	return bundle, nil
}

func shouldLoadProjectGuidance(mode string) bool {
	switch mode {
	case AgentInstructionProjectModeOff:
		return false
	case AgentInstructionProjectModeAlways, AgentInstructionProjectModeFallback:
		return true
	default:
		return true
	}
}

func appendDeprecatedProjectModeFallbackWarning(bundle *ProjectInstructionBundle, mode string) {
	if mode != AgentInstructionProjectModeFallback {
		return
	}
	appendProjectInstructionWarningMessage(bundle, ProjectInstructionWarningDeprecatedFallbackMode, projectModeFallbackDeprecationMessage)
}

func resolveProjectGuidanceStrength() InstructionStrength {
	return InstructionStrengthProjectGuidance
}

type guidanceLoadPlan struct {
	CandidatePath string
	LoadOptions   instructionFileLoadOptions
	Valid         bool
	Warning       *ProjectInstructionWarning
}

type guidanceLoadPlanResolver func(path string) guidanceLoadPlan

func forEachGuidanceCandidatePath(paths []string, includeLocalFiles bool, visit func(path string) bool) {
	if len(paths) == 0 || visit == nil {
		return
	}
	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		if !includeLocalFiles && isLocalGuidanceFile(path) {
			continue
		}
		if !visit(path) {
			return
		}
	}
}

func buildGuidanceLoadPlans(paths []string, includeLocalFiles bool, resolver guidanceLoadPlanResolver) []guidanceLoadPlan {
	if len(paths) == 0 || resolver == nil {
		return nil
	}
	plans := make([]guidanceLoadPlan, 0, len(paths))
	forEachGuidanceCandidatePath(paths, includeLocalFiles, func(path string) bool {
		plans = append(plans, resolver(path))
		return true
	})
	return plans
}

func loadGuidanceFiles(bundle *ProjectInstructionBundle, budget *instructionByteBudget, plans []guidanceLoadPlan) []InstructionFile {
	var files []InstructionFile
	for _, plan := range plans {
		if budget.exhausted() {
			break
		}
		file, loaded, stop := loadGuidanceFileFromPlan(bundle, budget, plan)
		if stop {
			break
		}
		if !loaded {
			continue
		}
		files = append(files, file)
	}
	return files
}

func loadGuidanceFileFromPlan(bundle *ProjectInstructionBundle, budget *instructionByteBudget, plan guidanceLoadPlan) (file InstructionFile, loaded bool, stop bool) {
	appendProjectInstructionWarning(bundle, plan.Warning)
	if !plan.Valid {
		return InstructionFile{}, false, false
	}

	loadResult := loadInstructionFile(plan.LoadOptions)
	appendProjectInstructionWarnings(bundle, loadResult.Warnings)
	if loadResult.Warning != "" {
		appendProjectInstructionWarningMessage(bundle, ProjectInstructionWarningLoadSkipped, loadResult.Warning)
	}
	if !loadResult.Loaded {
		if loadResult.SkipReason == instructionLoadSkipNoContentInBudget && budget.exhausted() {
			return InstructionFile{}, false, true
		}
		if loadResult.SkipReason == instructionLoadSkipNoContentInBudget {
			appendInstructionFileStatus(bundle, InstructionFileStatus{
				Path:   plan.LoadOptions.FullPath,
				Label:  plan.LoadOptions.DisplayLabel,
				Scope:  plan.LoadOptions.Scope,
				Status: InstructionFileStatusEmpty,
			})
		}
		return InstructionFile{}, false, false
	}
	return loadResult.File, true, false
}

func appendInstructionFileStatus(bundle *ProjectInstructionBundle, status InstructionFileStatus) {
	if bundle == nil || strings.TrimSpace(status.Label) == "" || status.Status == "" {
		return
	}
	target := &bundle.ProjectGuidanceStatus
	if status.Scope == "global" {
		target = &bundle.GlobalGuidanceStatus
	}
	for _, existing := range *target {
		if existing.Label == status.Label && existing.Scope == status.Scope && existing.Status == status.Status {
			return
		}
	}
	*target = append(*target, status)
}

func resolveProjectGuidanceLoadPlan(rootPath, resolvedRootPath, path string, aiCfg AgentInstructionsConfig, gitRoot string, strength InstructionStrength, budget *instructionByteBudget, gitTrackedLookup func(fullPath string) (tracked bool, known bool), repositoryScope string) guidanceLoadPlan {
	fullPath, ok := resolveProjectGuidancePath(rootPath, path)
	if !ok {
		return guidanceLoadPlan{
			CandidatePath: path,
			Valid:         false,
			Warning: &ProjectInstructionWarning{
				Code:    ProjectInstructionWarningInvalidProjectGuidancePath,
				Message: fmt.Sprintf("Skipped invalid project guidance path: %s", path),
			},
		}
	}

	boundary := newInstructionPathBoundary(rootPath, resolvedRootPath)
	return guidanceLoadPlan{
		CandidatePath: path,
		Valid:         true,
		LoadOptions: instructionFileLoadOptions{
			FullPath:        fullPath,
			DisplayLabel:    path,
			Scope:           "project",
			RepositoryScope: normalizeRepositoryInstructionScope(repositoryScope),
			Strength:        strength,
			Policy: instructionFileLoadPolicy{
				RequireGitTracked:    !aiCfg.Project.IncludeGitignored,
				IncludeGitignored:    aiCfg.Project.IncludeGitignored,
				GitRoot:              gitRoot,
				Budget:               budget,
				AllowReadWhenUnknown: true,
				RootBoundary:         boundary,
				ExpandImports:        aiCfg.ExpandImports,
				GitTrackedLookup:     gitTrackedLookup,
			},
		},
	}
}

func resolveGlobalGuidanceLoadPlan(path string, budget *instructionByteBudget, expandImports bool) guidanceLoadPlan {
	expandedPath := expandUserPath(path)
	return guidanceLoadPlan{
		CandidatePath: path,
		Valid:         true,
		LoadOptions: instructionFileLoadOptions{
			FullPath:     expandedPath,
			DisplayLabel: path,
			Scope:        "global",
			Strength:     InstructionStrengthAdvisory,
			Policy: instructionFileLoadPolicy{
				RequireGitTracked: false,
				Budget:            budget,
				ExpandImports:     expandImports,
			},
		},
	}
}

func buildProjectGuidanceLoadPlans(rootPath, cwd string, inputPaths []string, aiCfg AgentInstructionsConfig, gitRoot string, strength InstructionStrength, budget *instructionByteBudget) []guidanceLoadPlan {
	resolvedRootPath, _ := resolvePathForBoundaryComparison(rootPath)
	trackedCache := map[string]struct {
		tracked bool
		known   bool
	}{}
	gitTrackedLookup := func(fullPath string) (tracked bool, known bool) {
		if cached, ok := trackedCache[fullPath]; ok {
			return cached.tracked, cached.known
		}
		tracked, known = isGitTrackedInstructionFile(gitRoot, fullPath)
		trackedCache[fullPath] = struct {
			tracked bool
			known   bool
		}{tracked: tracked, known: known}
		return tracked, known
	}
	plans := buildScopedProjectGuidanceLoadPlans(rootPath, cwd, inputPaths, aiCfg, func(path, repositoryScope string) guidanceLoadPlan {
		return resolveProjectGuidanceLoadPlan(rootPath, resolvedRootPath, path, aiCfg, gitRoot, strength, budget, gitTrackedLookup, repositoryScope)
	})
	return plans
}

func buildGlobalGuidanceLoadPlans(aiCfg AgentInstructionsConfig, budget *instructionByteBudget) []guidanceLoadPlan {
	return buildGuidanceLoadPlans(aiCfg.Global.Files, aiCfg.IncludeLocalFiles, func(path string) guidanceLoadPlan {
		return resolveGlobalGuidanceLoadPlan(path, budget, aiCfg.ExpandImports)
	})
}

func loadProjectGuidanceFiles(bundle *ProjectInstructionBundle, aiCfg AgentInstructionsConfig, gitRoot string, strength InstructionStrength, budget *instructionByteBudget, cwd string, inputPaths []string) []InstructionFile {
	plans := buildProjectGuidanceLoadPlans(bundle.RootPath, cwd, inputPaths, aiCfg, gitRoot, strength, budget)
	return loadGuidanceFiles(bundle, budget, plans)
}

func loadGlobalGuidanceFiles(bundle *ProjectInstructionBundle, aiCfg AgentInstructionsConfig, budget *instructionByteBudget) []InstructionFile {
	plans := buildGlobalGuidanceLoadPlans(aiCfg, budget)
	return loadGuidanceFiles(bundle, budget, plans)
}

func appendProjectInstructionWarning(bundle *ProjectInstructionBundle, warning *ProjectInstructionWarning) {
	if bundle == nil || warning == nil {
		return
	}
	message := strings.TrimSpace(warning.Message)
	if message == "" {
		return
	}
	warning.Message = message
	for _, existing := range bundle.WarningEntries {
		if existing.Code == warning.Code && existing.Message == warning.Message {
			return
		}
	}
	bundle.WarningEntries = append(bundle.WarningEntries, *warning)
}

func appendProjectInstructionWarnings(bundle *ProjectInstructionBundle, warnings []ProjectInstructionWarning) {
	if bundle == nil || len(warnings) == 0 {
		return
	}
	for i := range warnings {
		warning := warnings[i]
		appendProjectInstructionWarning(bundle, &warning)
	}
}

func appendProjectInstructionWarningMessage(bundle *ProjectInstructionBundle, code ProjectInstructionWarningCode, message string) {
	appendProjectInstructionWarning(bundle, &ProjectInstructionWarning{Code: code, Message: message})
}

// WarningMessages は bundle に蓄積された warning メッセージを返す。
func (b *ProjectInstructionBundle) WarningMessages() []string {
	if b == nil || len(b.WarningEntries) == 0 {
		return nil
	}
	messages := make([]string, 0, len(b.WarningEntries))
	for _, warning := range b.WarningEntries {
		message := strings.TrimSpace(warning.Message)
		if message == "" {
			continue
		}
		messages = append(messages, message)
	}
	if len(messages) == 0 {
		return nil
	}
	return messages
}
