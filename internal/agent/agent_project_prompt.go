package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

var (
	// 引用付き path は slash を含むものを優先的に取る。
	projectMapQuotedPathPattern = regexp.MustCompile("[\"'`]([^\"'`]+(?:[\\\\/][^\"'`]+)+)[\"'`]")
	// slash を含まない quoted filename は dedicated pattern で扱う。
	// 'design spec.md' のような空白付き filename は他パターンではまたげない。
	projectMapQuotedFilenamePattern = regexp.MustCompile(`["']([^"']+\.[a-zA-Z0-9]{1,10})["']`)
	projectMapInputPathPatterns     = []*regexp.Regexp{
		projectMapQuotedPathPattern,
		regexp.MustCompile(`\b([A-Za-z]:[\\/][^\s"'` + "`" + `]+)\b`),
		regexp.MustCompile(`\b((?:[\w.-]+[\\/])+[\w./\\-]*)\b`),
		projectMapQuotedFilenamePattern,
		regexp.MustCompile(`\b((?:[\w.-]+/)*[\w.-]+\.[a-zA-Z0-9]{1,10})\b`),
		regexp.MustCompile(`(/[^\s"']+)`),
	}
)

const projectMapFocusMaxPaths = 5

type projectMapInputMatch struct {
	candidate string
	start     int
	end       int
}

func loadProjectConfig() *config.ProjectConfig {
	return config.LoadProjectConfig()
}

// injectProjectConfig は ProjectConfig を SystemPrompt に注入する。
// 入力内容に一致した rules/context のみを注入する。
func injectProjectConfig(systemPrompt string, pc *config.ProjectConfig, input string) string {
	systemPrompt = prompt.StripProjectConfigSections(systemPrompt)
	if pc == nil {
		return systemPrompt
	}

	selection := config.SelectProjectPromptSelection(pc, input)
	projectBlock := prompt.BuildProjectConfigBlock(selection.Rules, selection.Contexts)
	return prompt.InjectProjectConfigBlock(systemPrompt, projectBlock)
}

// applyProjectConfig はプロジェクト設定をエージェントに適用する統一ヘルパー。
// SystemPrompt 注入 + final checks 解決 + UI 表示を行う。
func applyProjectConfig(agent *Agent, pc *config.ProjectConfig) {
	if pc == nil {
		return
	}

	// 1. System prompt 注入
	agent.SystemPrompt = injectProjectConfig(agent.SystemPrompt, pc, "")

	// 2. final checks 解決（xelyon.yaml 優先、config.yaml フォールバック）
	if resolved := config.ResolveFinalChecks(agent.cfg(), pc); resolved != nil {
		cfg := agent.cfg()
		cfg.FinalChecks = *resolved
	}

	// 3. UI 表示
	green.Fprintln(agent.output(), "📋 xelyon.yaml loaded")
}

type projectMapInjectionContext struct {
	pm          *repomap.ProjectMap
	rebuilt     bool
	maxTokens   int
	baseKey     string
	focusPaths  []string
	focusKey    string
	fileCount   int
	symbolCount int
}

type projectMapSectionBuild struct {
	baseSection       string
	focusSection      string
	projectMapPrompt  string
	effectiveFocusKey string
}

// injectProjectMap はプロジェクト構造マップをシステムプロンプトに注入する。
func injectProjectMap(agent *Agent, input string) {
	if agent == nil {
		return
	}

	resetProjectMapPromptSection(agent)

	injectionCtx, ok := prepareProjectMapInjection(agent, input)
	if !ok {
		return
	}

	if applyProjectMapCachedSection(agent, injectionCtx) {
		return
	}

	build, ok := buildProjectMapSection(agent, injectionCtx)
	if !ok {
		resetProjectMapCachedSections(agent)
		agent.projectMapDirty = false
		return
	}

	applyProjectMapBuiltSection(agent, injectionCtx, build)
	if injectionCtx.rebuilt {
		green.Fprintf(agent.output(), "🗺️  Project map loaded (%d files, %d symbols)\n", agent.projectMapFileCount, agent.projectMapSymbolCount)
	}
}

func resetProjectMapPromptSection(agent *Agent) {
	agent.SystemPrompt = stripProjectMapSection(agent.SystemPrompt)
	agent.projectMapFileCount = 0
	agent.projectMapSymbolCount = 0
}

func resetProjectMapCachedSections(agent *Agent) {
	agent.projectMapBaseSection = ""
	agent.projectMapFocusSection = ""
	agent.projectMapSection = ""
	agent.projectMapBaseKey = ""
	agent.projectMapFocusKey = ""
}

func prepareProjectMapInjection(agent *Agent, input string) (projectMapInjectionContext, bool) {
	cfg := agent.cfg()
	if !cfg.ProjectMap.Enabled {
		return projectMapInjectionContext{}, false
	}
	if !common.IsRipgrepAvailable() {
		return projectMapInjectionContext{}, false
	}

	cwd, err := os.Getwd()
	if err != nil {
		return projectMapInjectionContext{}, false
	}

	pc := loadProjectConfig()
	rootPath := cwd
	if pc != nil && strings.TrimSpace(pc.FilePath) != "" {
		rootPath = filepath.Dir(pc.FilePath)
	}
	ignorePatterns := config.ResolveSharedIgnorePatterns(cfg, pc)
	ignoreKey := strings.Join(ignorePatterns, "\x00")

	pm, rebuilt := ensureProjectMap(agent, rootPath, ignorePatterns, ignoreKey)
	if pm == nil {
		return projectMapInjectionContext{}, false
	}

	fileCount := pm.GetFileCount()
	symbolCount := pm.GetSymbolCount()
	maxTokens := calcProjectMapBudget(agent, cfg, fileCount, symbolCount)
	pm.MaxTokens = maxTokens

	focusPaths := extractProjectMapFocusPaths(cwd, rootPath, input, projectMapFocusMaxPaths)
	return projectMapInjectionContext{
		pm:          pm,
		rebuilt:     rebuilt,
		maxTokens:   maxTokens,
		baseKey:     buildProjectMapBaseKey(agent, cfg, maxTokens, fileCount, symbolCount),
		focusPaths:  focusPaths,
		focusKey:    buildProjectMapFocusKey(focusPaths),
		fileCount:   fileCount,
		symbolCount: symbolCount,
	}, true
}

func applyProjectMapCachedSection(agent *Agent, injectionCtx projectMapInjectionContext) bool {
	if injectionCtx.rebuilt ||
		agent.projectMapBaseSection == "" ||
		agent.projectMapBaseKey != injectionCtx.baseKey ||
		agent.projectMapFocusKey != injectionCtx.focusKey ||
		agent.projectMapSection == "" ||
		token.EstimateTokenCount(agent.projectMapBaseSection) > injectionCtx.maxTokens ||
		token.EstimateTokenCount(agent.projectMapSection) > injectionCtx.maxTokens {
		return false
	}

	agent.SystemPrompt = appendProjectMapSection(agent.SystemPrompt, agent.projectMapSection)
	agent.projectMapFileCount = injectionCtx.fileCount
	agent.projectMapSymbolCount = injectionCtx.symbolCount
	agent.projectMapDirty = false
	return true
}

func buildProjectMapSection(agent *Agent, injectionCtx projectMapInjectionContext) (projectMapSectionBuild, bool) {
	baseSection := agent.projectMapBaseSection
	if injectionCtx.rebuilt || agent.projectMapBaseKey != injectionCtx.baseKey || strings.TrimSpace(baseSection) == "" {
		baseSection = injectionCtx.pm.GenerateManifest(nil)
	}

	focusSection := renderProjectMapFocusOverlay(injectionCtx.focusPaths)
	projectMapPrompt := composeProjectMapPromptSection(baseSection, focusSection)
	if projectMapPrompt != "" && token.EstimateTokenCount(projectMapPrompt) > injectionCtx.maxTokens {
		projectMapPrompt = composeProjectMapPromptSection(baseSection, "")
		focusSection = ""
	}
	if projectMapPrompt == "" {
		return projectMapSectionBuild{}, false
	}

	focusCount := countProjectMapFocusLines(focusSection)
	if focusCount > len(injectionCtx.focusPaths) {
		focusCount = len(injectionCtx.focusPaths)
	}

	return projectMapSectionBuild{
		baseSection:       baseSection,
		focusSection:      focusSection,
		projectMapPrompt:  projectMapPrompt,
		effectiveFocusKey: buildProjectMapFocusKey(injectionCtx.focusPaths[:focusCount]),
	}, true
}

func applyProjectMapBuiltSection(agent *Agent, injectionCtx projectMapInjectionContext, build projectMapSectionBuild) {
	agent.SystemPrompt = appendProjectMapSection(agent.SystemPrompt, build.projectMapPrompt)
	agent.projectMapFileCount = injectionCtx.fileCount
	agent.projectMapSymbolCount = injectionCtx.symbolCount
	agent.projectMapBaseSection = build.baseSection
	agent.projectMapFocusSection = build.focusSection
	agent.projectMapSection = build.projectMapPrompt
	agent.projectMapBaseKey = injectionCtx.baseKey
	agent.projectMapFocusKey = build.effectiveFocusKey
	agent.projectMapDirty = false
}

func buildProjectMapBaseKey(agent *Agent, cfg *config.Config, maxTokens, fileCount, symbolCount int) string {
	stateKey := ""
	if agent != nil {
		stateKey = agent.projectMapStateKey
	}
	contextWindow := 0
	if agent != nil {
		contextWindow = token.GetModelTokenLimit(agent.CurrentModel)
	}
	if contextWindow <= 0 {
		contextWindow = 128000
	}

	ratio := config.NormalizeProjectMapContextRatio(0)
	if cfg != nil {
		ratio = config.NormalizeProjectMapContextRatio(cfg.ProjectMap.ContextRatio)
	}
	effectiveRatio := effectiveProjectMapContextRatio(ratio, fileCount, symbolCount)

	return fmt.Sprintf("%s\x00budget:%d\x00ctx:%d\x00ratio:%.6f", stateKey, maxTokens, contextWindow, effectiveRatio)
}

func buildProjectMapFocusKey(paths []string) string {
	return strings.Join(dedupeProjectMapPriorityPaths(paths), "\x00")
}

func extractProjectMapFocusPaths(cwd, rootPath, input string, limit int) []string {
	if limit <= 0 {
		return nil
	}

	paths := dedupeProjectMapPriorityPaths(projectMapPriorityPathsFromInput(cwd, rootPath, extractProjectMapPathsFromInput(input), limit))
	if len(paths) == 0 {
		return nil
	}
	return paths
}

func extractProjectMapPathsFromInput(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	pathSet := make(map[string]struct{})
	var accepted []projectMapInputMatch
	var paths []string
	for _, pattern := range projectMapInputPathPatterns {
		matches := pattern.FindAllStringSubmatchIndex(input, -1)
		for _, match := range matches {
			if len(match) < 4 {
				continue
			}
			start, end := match[2], match[3]
			if shouldSkipProjectMapInputMatch(accepted, start, end) {
				continue
			}
			candidate := cleanProjectMapInputPathCandidate(input[start:end])
			if candidate == "" {
				continue
			}
			if _, ok := pathSet[candidate]; ok {
				continue
			}
			pathSet[candidate] = struct{}{}
			accepted = append(accepted, projectMapInputMatch{
				candidate: candidate,
				start:     start,
				end:       end,
			})
			paths = append(paths, candidate)
		}
	}
	return filterProjectMapInputCandidates(paths)
}

func shouldSkipProjectMapInputMatch(accepted []projectMapInputMatch, start, end int) bool {
	for _, existing := range accepted {
		if start >= existing.start && end <= existing.end {
			return true
		}
	}
	return false
}

func cleanProjectMapInputPathCandidate(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	candidate = strings.Trim(candidate, ".,:;!?()[]{}<>")
	candidate = strings.Trim(candidate, "\"'`")
	candidate = strings.ReplaceAll(candidate, "\\", "/")
	if candidate == "" {
		return ""
	}
	if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
		return ""
	}
	if !strings.Contains(candidate, "/") && !strings.Contains(candidate, ".") {
		return ""
	}
	return candidate
}

func filterProjectMapInputCandidates(candidates []string) []string {
	if len(candidates) == 0 {
		return nil
	}

	type normalizedCandidate struct {
		original   string
		normalized string
	}

	normalized := make([]normalizedCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		cleaned := cleanProjectMapInputPathCandidate(candidate)
		if cleaned == "" {
			continue
		}
		normalized = append(normalized, normalizedCandidate{
			original:   cleaned,
			normalized: strings.ToLower(cleaned),
		})
	}

	filtered := make([]string, 0, len(normalized))
	for i, candidate := range normalized {
		if candidate.original == "" {
			continue
		}
		if strings.Contains(candidate.original, "/") {
			filtered = append(filtered, candidate.original)
			continue
		}

		shadowed := false
		for j, other := range normalized {
			if i == j {
				continue
			}
			if !strings.Contains(other.original, "/") {
				continue
			}
			if strings.HasSuffix(other.normalized, "/"+candidate.normalized) {
				shadowed = true
				break
			}
		}
		if !shadowed {
			filtered = append(filtered, candidate.original)
		}
	}

	return dedupeProjectMapPriorityPaths(filtered)
}

func projectMapPriorityPathsFromInput(cwd, rootPath string, candidates []string, limit int) []string {
	if limit <= 0 {
		return nil
	}

	capHint := len(candidates)
	if capHint > limit {
		capHint = limit
	}
	normalized := make([]string, 0, capHint)
	for _, candidate := range candidates {
		path, ok := resolveProjectMapInputCandidate(cwd, rootPath, candidate)
		if !ok {
			continue
		}
		normalized = append(normalized, path)
		if len(normalized) >= limit {
			break
		}
	}
	return normalized
}

func resolveProjectMapInputCandidate(cwd, rootPath, candidate string) (string, bool) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" || rootPath == "" {
		return "", false
	}
	if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
		return "", false
	}
	if filepath.IsAbs(candidate) {
		absPath := filepath.Clean(candidate)
		if !projectMapPathExists(absPath) {
			return "", false
		}
		return canonicalizeProjectMapPriorityPath(rootPath, absPath)
	}
	if isWindowsAbsoluteProjectMapPath(candidate) {
		absPath := filepath.Clean(windowsAbsoluteProjectMapPathToLocal(candidate))
		if !projectMapPathExists(absPath) {
			return "", false
		}
		return canonicalizeProjectMapPriorityPath(rootPath, absPath)
	}

	sessionAbs := filepath.Clean(filepath.Join(cwd, filepath.FromSlash(candidate)))
	rootAbs := filepath.Clean(filepath.Join(rootPath, filepath.FromSlash(candidate)))

	sessionExists := projectMapPathExists(sessionAbs)
	rootExists := projectMapPathExists(rootAbs)

	switch {
	case rootExists && (looksRepoRelativeProjectMapPath(candidate) || !sessionExists):
		return canonicalizeProjectMapPriorityPath(rootPath, rootAbs)
	case sessionExists:
		return canonicalizeProjectMapPriorityPath(rootPath, sessionAbs)
	case rootExists:
		return canonicalizeProjectMapPriorityPath(rootPath, rootAbs)
	default:
		return "", false
	}
}

func canonicalizeProjectMapPriorityPath(rootPath, absPath string) (string, bool) {
	if rootPath == "" || absPath == "" {
		return "", false
	}

	rootAbs, err := filepath.Abs(rootPath)
	if err != nil {
		return "", false
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", false
	}

	relPath, err := filepath.Rel(rootAbs, absPath)
	if err != nil {
		return "", false
	}
	if relPath == "." {
		return "", false
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", false
	}

	return filepath.ToSlash(filepath.Clean(relPath)), true
}

func projectMapPathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func looksRepoRelativeProjectMapPath(candidate string) bool {
	candidate = filepath.ToSlash(strings.TrimSpace(candidate))
	if candidate == "" {
		return false
	}
	if strings.HasPrefix(candidate, "./") || strings.HasPrefix(candidate, "../") {
		return false
	}
	return strings.Contains(candidate, "/")
}

func isWindowsAbsoluteProjectMapPath(candidate string) bool {
	if len(candidate) < 4 {
		return false
	}
	if (candidate[0] < 'A' || candidate[0] > 'Z') && (candidate[0] < 'a' || candidate[0] > 'z') {
		return false
	}
	return candidate[1] == ':' && candidate[2] == '/'
}

func windowsAbsoluteProjectMapPathToLocal(candidate string) string {
	if !isWindowsAbsoluteProjectMapPath(candidate) {
		return candidate
	}
	return candidate[2:]
}

func renderProjectMapFocusOverlay(paths []string) string {
	paths = dedupeProjectMapPriorityPaths(paths)
	if len(paths) == 0 {
		return ""
	}
	if len(paths) > projectMapFocusMaxPaths {
		paths = paths[:projectMapFocusMaxPaths]
	}

	var b strings.Builder
	b.WriteString("Focus files for current task:\n")
	for _, path := range paths {
		b.WriteString("- ")
		b.WriteString(path)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func composeProjectMapPromptSection(baseSection, focusSection string) string {
	baseSection = strings.TrimRight(baseSection, "\n")
	focusSection = strings.TrimRight(focusSection, "\n")

	switch {
	case baseSection == "":
		if focusSection == "" {
			return ""
		}
		return "## Project Map\n\n" + focusSection
	case focusSection == "":
		return baseSection
	default:
		return baseSection + "\n\n" + focusSection
	}
}

func countProjectMapFocusLines(section string) int {
	if strings.TrimSpace(section) == "" {
		return 0
	}
	count := 0
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "- ") {
			count++
		}
	}
	return count
}

func dedupeProjectMapPriorityPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	deduped := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		deduped = append(deduped, path)
	}
	return deduped
}

func appendProjectMapSection(systemPrompt, section string) string {
	if strings.TrimSpace(section) == "" {
		return systemPrompt
	}

	// Project Map is the most volatile part of the system prompt.
	// Put it behind a cache boundary so Claude can reuse the stable prefix
	// even when the map changes after edits or repo-state updates.
	if !strings.Contains(systemPrompt, api.SystemPromptCacheBoundary) {
		return systemPrompt + api.SystemPromptCacheBoundary + section
	}

	return systemPrompt + "\n\n" + section
}

func ensureProjectMap(agent *Agent, rootPath string, ignorePatterns []string, ignoreKey string) (*repomap.ProjectMap, bool) {
	if agent == nil {
		return nil, false
	}

	if !agent.projectMapDirty &&
		agent.projectMap != nil &&
		agent.projectMapRootPath == rootPath &&
		agent.projectMapIgnoreKey == ignoreKey {
		if stateKey := currentProjectMapStateKey(agent, rootPath); stateKey != "" && agent.projectMapStateKey == stateKey {
			return agent.projectMap, false
		}
	}

	pm := repomap.NewProjectMap(rootPath, 0, ignorePatterns...)
	if err := pm.Build(); err != nil {
		yellow.Fprintf(agent.output(), "⚠️ ProjectMap build failed: %v\n", err)
		return nil, false
	}

	agent.projectMap = pm
	agent.projectMapRootPath = rootPath
	agent.projectMapIgnoreKey = ignoreKey
	agent.projectMapWatchDirs = nil
	if !isGitProjectMapAvailable(rootPath) {
		agent.projectMapWatchDirs = collectProjectMapWatchDirs(rootPath, ignorePatterns)
	}
	agent.projectMapStateKey = currentProjectMapStateKey(agent, rootPath)
	return pm, true
}

func calcProjectMapBudget(agent *Agent, cfg *config.Config, fileCount, symbolCount int) int {
	// コンテキストウィンドウサイズを取得
	contextWindow := token.GetModelTokenLimit(agent.CurrentModel)
	if contextWindow <= 0 {
		contextWindow = 128000 // フォールバック
	}

	ratio := effectiveProjectMapContextRatio(cfg.ProjectMap.ContextRatio, fileCount, symbolCount)
	budgetCap := int(float64(contextWindow) * ratio)

	if budgetCap < 1 {
		return 1
	}

	return budgetCap
}

func effectiveProjectMapContextRatio(baseRatio float64, fileCount, symbolCount int) float64 {
	ratio := config.NormalizeProjectMapContextRatio(baseRatio)

	switch {
	case fileCount >= 400 || symbolCount >= 4000:
		if ratio < 0.04 {
			return 0.04
		}
	case fileCount >= 200 || symbolCount >= 2000:
		if ratio < 0.03 {
			return 0.03
		}
	}

	return ratio
}

// rebuildSystemPromptForCurrentProvider は現在の provider/model に合わせて
// SystemPrompt をベースから再構築する。
func (a *Agent) rebuildSystemPromptForCurrentProvider() {
	a.promptManager().RebuildSystemPromptForCurrentProvider()
}

func estimateProjectConfigTokens(pc *config.ProjectConfig) int {
	if pc == nil {
		return 0
	}

	selection := config.SelectProjectPromptSelection(pc, "")
	return token.EstimateTokenCount(prompt.BuildProjectConfigBlock(selection.Rules, selection.Contexts))
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

func currentProjectMapStateKey(agent *Agent, rootPath string) string {
	head := gitProjectMapHEAD(rootPath)
	status := gitProjectMapStatusDigest(rootPath)
	if head != "" || status != "" {
		return head + ":" + status
	}

	if digest := nonGitProjectMapWatchDigest(rootPath, projectMapWatchDirs(agent), projectMapIgnorePatterns(agent)); digest != "" {
		return "dirs:" + digest
	}

	info, err := os.Stat(rootPath)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("fs:%d", info.ModTime().UTC().UnixNano())
}

func gitProjectMapHEAD(rootPath string) string {
	return gitProjectMapCommandDigest(rootPath, []string{"rev-parse", "HEAD"})
}

func gitProjectMapStatusDigest(rootPath string) string {
	return gitProjectMapCommandDigest(rootPath, []string{"status", "--porcelain"})
}

func gitProjectMapCommandDigest(rootPath string, args []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = rootPath

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}

	sum := sha256.Sum256(bytes.TrimSpace(stdout.Bytes()))
	return hex.EncodeToString(sum[:])
}

func isGitProjectMapAvailable(rootPath string) bool {
	return gitProjectMapHEAD(rootPath) != "" || gitProjectMapStatusDigest(rootPath) != ""
}

func projectMapWatchDirs(agent *Agent) []string {
	if agent == nil || len(agent.projectMapWatchDirs) == 0 {
		return []string{"."}
	}

	dirs := make([]string, len(agent.projectMapWatchDirs))
	copy(dirs, agent.projectMapWatchDirs)
	return dirs
}

func projectMapIgnorePatterns(agent *Agent) []string {
	if agent == nil || agent.projectMapIgnoreKey == "" {
		return nil
	}
	return pathmatch.NormalizePatterns(strings.Split(agent.projectMapIgnoreKey, "\x00"))
}

func nonGitProjectMapWatchDigest(rootPath string, watchDirs []string, ignorePatterns []string) string {
	if len(watchDirs) == 0 {
		return ""
	}

	matcher := pathmatch.NewMatcher(ignorePatterns)
	var state strings.Builder
	for _, relDir := range watchDirs {
		relDir = filepath.Clean(filepath.ToSlash(strings.TrimSpace(relDir)))
		if relDir == "" {
			relDir = "."
		}

		absDir := rootPath
		if relDir != "." {
			absDir = filepath.Join(rootPath, relDir)
		}

		entries, err := os.ReadDir(absDir)
		switch {
		case err != nil:
			state.WriteString(relDir)
			state.WriteString(":missing\n")
		default:
			filtered := 0
			var entryState strings.Builder
			for _, entry := range entries {
				entryRelPath := entry.Name()
				if relDir != "." {
					entryRelPath = filepath.ToSlash(filepath.Join(relDir, entry.Name()))
				}
				if matcher.Match(entryRelPath, entry.IsDir()) {
					continue
				}
				filtered++
				entryState.WriteString(entry.Name())
				if entry.IsDir() {
					entryState.WriteByte('/')
				}
				entryState.WriteByte('\n')
			}
			_, _ = fmt.Fprintf(&state, "%s:%d\n", relDir, filtered)
			state.WriteString(entryState.String())
		}
	}

	sum := sha256.Sum256([]byte(state.String()))
	return hex.EncodeToString(sum[:])
}

func collectProjectMapWatchDirs(rootPath string, ignorePatterns []string) []string {
	matcher := pathmatch.NewMatcher(ignorePatterns)
	dirs := []string{"."}

	_ = filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		relPath, relErr := filepath.Rel(rootPath, path)
		if relErr != nil {
			return nil
		}
		relPath = filepath.Clean(filepath.ToSlash(relPath))
		if relPath == "." {
			return nil
		}
		if matcher.Match(relPath, true) {
			return filepath.SkipDir
		}

		dirs = append(dirs, relPath)
		return nil
	})

	slices.Sort(dirs)
	return slices.Compact(dirs)
}
