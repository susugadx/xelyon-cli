package config

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

var referencedPathPattern = regexp.MustCompile("`[^`]+`|\"[^\"]+\"|'[^']+'|[A-Za-z0-9_./\\\\:-]+")

// ProjectPromptSelection は legacy rules/context の互換表示用解決結果。
// 通常の SystemPrompt 注入では AGENTS.md などの guidance を使い、この結果は使わない。
type ProjectPromptSelection struct {
	Rules    []string
	Contexts []string
}

// ResolveSharedIgnorePatterns は repomap/list_dir/search_code で共通利用する ignore パターンを返す。
func ResolveSharedIgnorePatterns(globalCfg *Config, projectCfg *ProjectConfig, extraPatterns ...string) []string {
	patterns := pathmatch.DefaultIgnorePatterns()

	if globalCfg != nil {
		patterns = append(patterns, globalCfg.ListDir.AdditionalIgnoreDirs...)
		patterns = append(patterns, globalCfg.ProjectMap.AdditionalIgnoreDirs...)
	}
	if projectCfg != nil {
		patterns = append(patterns, projectCfg.Ignore.Patterns...)
	}
	patterns = append(patterns, extraPatterns...)

	return pathmatch.NormalizePatterns(patterns)
}

// SelectProjectPromptSelection は入力内容に一致する legacy rules/context だけを解決する。
func SelectProjectPromptSelection(projectCfg *ProjectConfig, input string) ProjectPromptSelection {
	if projectCfg == nil {
		return ProjectPromptSelection{}
	}

	selection := ProjectPromptSelection{
		Rules: append([]string(nil), projectCfg.Rules...),
	}
	if context := strings.TrimSpace(projectCfg.Context); context != "" {
		selection.Contexts = append(selection.Contexts, context)
	}

	paths := ExtractReferencedProjectPathsForRoot(input, projectRootFromConfig(projectCfg))
	for _, block := range projectCfg.Conditional {
		if !matchesConditionalBlock(block, paths) {
			continue
		}
		selection.Rules = append(selection.Rules, block.Rules...)
		if context := formatConditionalContext(block); context != "" {
			selection.Contexts = append(selection.Contexts, context)
		}
	}

	selection.Rules = dedupeNonEmpty(selection.Rules)
	selection.Contexts = dedupeNonEmpty(selection.Contexts)
	return selection
}

// ExtractReferencedProjectPaths はユーザー入力からパスらしいトークンを抽出する。
func ExtractReferencedProjectPaths(input string) []string {
	return ExtractReferencedProjectPathsForRoot(input, "")
}

// ExtractReferencedProjectPathsForRoot は project root を基準にパスらしいトークンを抽出する。
func ExtractReferencedProjectPathsForRoot(input string, projectRoot string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	candidates := referencedPathPattern.FindAllString(input, -1)
	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if path := cleanReferencedPath(candidate, projectRoot); path != "" {
			paths = append(paths, path)
		}
	}
	return dedupeNonEmpty(paths)
}

func matchesConditionalBlock(block ProjectConditionalBlock, paths []string) bool {
	if len(block.Paths) == 0 {
		return true
	}
	if len(paths) == 0 {
		return false
	}

	matcher := pathmatch.NewMatcher(block.Paths)
	for _, candidate := range paths {
		if matcher.Match(candidate, false) || matcher.Match(candidate, true) {
			return true
		}
	}
	return false
}

func formatConditionalContext(block ProjectConditionalBlock) string {
	context := strings.TrimSpace(block.Context)
	if context == "" {
		return ""
	}
	if strings.TrimSpace(block.Name) == "" {
		return context
	}
	return "### " + strings.TrimSpace(block.Name) + "\n" + context
}

func cleanReferencedPath(candidate string, projectRoot string) string {
	candidate = strings.TrimSpace(candidate)
	candidate = strings.Trim(candidate, "`\"'")
	candidate = strings.TrimSuffix(candidate, ":")
	if idx := strings.Index(candidate, "://"); idx >= 0 {
		return ""
	}

	if strings.Contains(candidate, ":") {
		if pathPart, _, found := strings.Cut(candidate, ":"); found && looksLikeLineReference(candidate) {
			candidate = pathPart
		}
	}

	if rel, ok := relativizeReferencedPath(candidate, projectRoot); ok {
		candidate = rel
	} else {
		return ""
	}
	candidate = strings.TrimPrefix(candidate, "./")
	if candidate == "" {
		return ""
	}
	candidate = filepath.ToSlash(filepath.Clean(candidate))
	if candidate == "." || candidate == "" {
		return ""
	}
	if !strings.Contains(candidate, "/") && filepath.Ext(candidate) == "" && !isKnownTopLevelProjectFile(candidate, projectRoot) {
		return ""
	}
	return candidate
}

func relativizeReferencedPath(candidate string, projectRoot string) (string, bool) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "", false
	}

	if projectRoot != "" && filepath.IsAbs(candidate) {
		rel, err := filepath.Rel(projectRoot, candidate)
		if err != nil {
			return "", false
		}
		rel = filepath.ToSlash(filepath.Clean(rel))
		if rel == ".." || strings.HasPrefix(rel, "../") {
			return "", false
		}
		return rel, true
	}

	if filepath.IsAbs(candidate) {
		return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(candidate)), "/"), true
	}

	return candidate, true
}

func isKnownTopLevelProjectFile(candidate string, projectRoot string) bool {
	if projectRoot != "" {
		if _, err := os.Stat(filepath.Join(projectRoot, candidate)); err == nil {
			return true
		}
	}

	switch strings.ToLower(candidate) {
	case "makefile", "dockerfile", "containerfile", "justfile", "procfile", "gemfile", "rakefile", "brewfile", "vagrantfile", "earthfile", "license", "copying", "notice", "readme":
		return true
	default:
		return false
	}
}

func projectRootFromConfig(projectCfg *ProjectConfig) string {
	if projectCfg == nil || strings.TrimSpace(projectCfg.FilePath) == "" {
		return ""
	}
	return filepath.Dir(projectCfg.FilePath)
}

func looksLikeLineReference(value string) bool {
	_, rest, found := strings.Cut(value, ":")
	if !found {
		return false
	}
	if rest == "" {
		return false
	}
	for _, ch := range rest {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func dedupeNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(result, value) {
			continue
		}
		result = append(result, value)
	}
	return result
}
