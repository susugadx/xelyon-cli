package config

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// ProjectInstructionBundle は project instruction 注入用の統合データ。
// xelyon.yaml の mandatory policy と AGENTS/CLAUDE guidance を分離して保持する。
type ProjectInstructionBundle struct {
	RootPath string

	ProjectConfig *ProjectConfig

	ProjectGuidance []InstructionFile
	GlobalGuidance  []InstructionFile

	Warnings []string
}

// InstructionStrength は guidance の優先度カテゴリ。
type InstructionStrength string

const (
	InstructionStrengthProjectGuidance InstructionStrength = "project_guidance"
	InstructionStrengthAdvisory        InstructionStrength = "advisory"
)

// InstructionFile は読み込んだ guidance ファイルの内容。
type InstructionFile struct {
	Path       string
	Label      string
	Scope      string
	Strength   InstructionStrength
	Content    string
	Truncated  bool
	GitTracked bool
}

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

	bundle := &ProjectInstructionBundle{
		ProjectConfig: projectCfg,
		RootPath:      resolveBundleRootPath(cwd, projectCfg, gitRoot, cfgForLoad.AgentInstructions),
	}

	mode := normalizeInstructionProjectMode(cfgForLoad.AgentInstructions.Project.Mode)
	budget := newInstructionByteBudget(cfgForLoad.AgentInstructions)

	if shouldLoadProjectGuidance(mode, projectCfg != nil) {
		strength := resolveProjectGuidanceStrength(projectCfg != nil)
		bundle.ProjectGuidance = loadProjectGuidanceFiles(bundle, cfgForLoad.AgentInstructions, gitRoot, strength, &budget)
	}

	if cfgForLoad.AgentInstructions.Global.Enabled {
		bundle.GlobalGuidance = loadGlobalGuidanceFiles(cfgForLoad.AgentInstructions, &budget)
	}

	return bundle, nil
}

func loadProjectConfigForDir(cwd string) (*ProjectConfig, error) {
	if path := findFileUpward(cwd, "xelyon.yaml"); path != "" {
		pc, err := loadProjectConfigFromYAML(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", path, err)
		}
		return pc, nil
	}
	return nil, nil
}

func resolveBundleRootPath(cwd string, projectCfg *ProjectConfig, gitRoot string, aiCfg AgentInstructionsConfig) string {
	if projectCfg != nil && strings.TrimSpace(projectCfg.FilePath) != "" {
		return filepath.Dir(projectCfg.FilePath)
	}
	if gitRoot != "" {
		return gitRoot
	}
	if guidanceRoot := findGuidanceRootUpward(cwd, aiCfg.Project.Files, aiCfg.IncludeLocalFiles); guidanceRoot != "" {
		return guidanceRoot
	}
	return cwd
}

func findGuidanceRootUpward(cwd string, files []string, includeLocal bool) string {
	for dir := cwd; ; dir = filepath.Dir(dir) {
		for _, candidate := range files {
			candidate = strings.TrimSpace(candidate)
			if candidate == "" {
				continue
			}
			if !includeLocal && isLocalGuidanceFile(candidate) {
				continue
			}
			fullPath := filepath.Join(dir, filepath.FromSlash(candidate))
			if _, err := os.Stat(fullPath); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return ""
}

func normalizeInstructionProjectMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case "off", "always", "fallback":
		return normalized
	default:
		return "fallback"
	}
}

func shouldLoadProjectGuidance(mode string, hasProjectConfig bool) bool {
	switch mode {
	case "off":
		return false
	case "always":
		return true
	case "fallback":
		return !hasProjectConfig
	default:
		return !hasProjectConfig
	}
}

func resolveProjectGuidanceStrength(hasProjectConfig bool) InstructionStrength {
	if hasProjectConfig {
		return InstructionStrengthAdvisory
	}
	return InstructionStrengthProjectGuidance
}

func loadProjectGuidanceFiles(bundle *ProjectInstructionBundle, aiCfg AgentInstructionsConfig, gitRoot string, strength InstructionStrength, budget *instructionByteBudget) []InstructionFile {
	var files []InstructionFile
	for _, raw := range aiCfg.Project.Files {
		if budget.exhausted() {
			break
		}
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		if !aiCfg.IncludeLocalFiles && isLocalGuidanceFile(path) {
			continue
		}

		fullPath := filepath.Join(bundle.RootPath, filepath.FromSlash(path))
		info, ok := loadInstructionFile(instructionFileLoadOptions{
			FullPath:             fullPath,
			DisplayLabel:         path,
			Scope:                "project",
			Strength:             strength,
			RequireGitTracked:    !aiCfg.Project.IncludeGitignored,
			GitRoot:              gitRoot,
			IncludeGitignored:    aiCfg.Project.IncludeGitignored,
			Budget:               budget,
			AllowReadWhenUnknown: true,
		})
		if !ok {
			continue
		}
		files = append(files, info)
	}
	return files
}

func loadGlobalGuidanceFiles(aiCfg AgentInstructionsConfig, budget *instructionByteBudget) []InstructionFile {
	var files []InstructionFile
	for _, raw := range aiCfg.Global.Files {
		if budget.exhausted() {
			break
		}
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		expandedPath := expandUserPath(path)
		if !aiCfg.IncludeLocalFiles && isLocalGuidanceFile(expandedPath) {
			continue
		}

		info, ok := loadInstructionFile(instructionFileLoadOptions{
			FullPath:          expandedPath,
			DisplayLabel:      path,
			Scope:             "global",
			Strength:          InstructionStrengthAdvisory,
			RequireGitTracked: false,
			Budget:            budget,
		})
		if !ok {
			continue
		}
		files = append(files, info)
	}
	return files
}

type instructionByteBudget struct {
	maxFileBytes  int
	maxTotalBytes int
	usedBytes     int
}

func newInstructionByteBudget(aiCfg AgentInstructionsConfig) instructionByteBudget {
	maxFile := aiCfg.MaxFileBytes
	if maxFile <= 0 {
		maxFile = defaultAgentInstructionsConfig().MaxFileBytes
	}
	maxTotal := aiCfg.MaxTotalBytes
	if maxTotal <= 0 {
		maxTotal = defaultAgentInstructionsConfig().MaxTotalBytes
	}
	return instructionByteBudget{maxFileBytes: maxFile, maxTotalBytes: maxTotal}
}

func (b *instructionByteBudget) exhausted() bool {
	if b == nil {
		return false
	}
	if b.maxTotalBytes <= 0 {
		return false
	}
	return b.usedBytes >= b.maxTotalBytes
}

func (b *instructionByteBudget) remaining() int {
	if b == nil || b.maxTotalBytes <= 0 {
		return int(^uint(0) >> 1)
	}
	remaining := b.maxTotalBytes - b.usedBytes
	if remaining < 0 {
		return 0
	}
	return remaining
}

type instructionFileLoadOptions struct {
	FullPath             string
	DisplayLabel         string
	Scope                string
	Strength             InstructionStrength
	RequireGitTracked    bool
	IncludeGitignored    bool
	GitRoot              string
	Budget               *instructionByteBudget
	AllowReadWhenUnknown bool
}

func loadInstructionFile(opts instructionFileLoadOptions) (InstructionFile, bool) {
	if strings.TrimSpace(opts.FullPath) == "" {
		return InstructionFile{}, false
	}
	if _, err := os.Stat(opts.FullPath); err != nil {
		return InstructionFile{}, false
	}

	gitTracked := false
	if opts.RequireGitTracked {
		tracked, known := isGitTrackedInstructionFile(opts.GitRoot, opts.FullPath)
		if known {
			if !tracked {
				if !opts.IncludeGitignored {
					return InstructionFile{}, false
				}
			} else {
				gitTracked = true
			}
		} else if !opts.AllowReadWhenUnknown {
			return InstructionFile{}, false
		}
	}

	data, err := os.ReadFile(opts.FullPath)
	if err != nil {
		return InstructionFile{}, false
	}

	content, truncated, consumed := applyInstructionContentLimits(data, opts.Budget)
	if consumed <= 0 && strings.TrimSpace(content) == "" {
		return InstructionFile{}, false
	}

	return InstructionFile{
		Path:       opts.FullPath,
		Label:      opts.DisplayLabel,
		Scope:      opts.Scope,
		Strength:   opts.Strength,
		Content:    content,
		Truncated:  truncated,
		GitTracked: gitTracked,
	}, true
}

func applyInstructionContentLimits(data []byte, budget *instructionByteBudget) (content string, truncated bool, consumed int) {
	if budget == nil {
		return string(data), false, len(data)
	}

	limit := len(data)
	var notes []string
	if budget.maxFileBytes > 0 && limit > budget.maxFileBytes {
		limit = budget.maxFileBytes
		truncated = true
		notes = append(notes, fmt.Sprintf("[Content truncated after %d bytes by XELYON agent_instructions.max_file_bytes]", budget.maxFileBytes))
	}

	remaining := budget.remaining()
	if limit > remaining {
		limit = remaining
		truncated = true
		notes = append(notes, fmt.Sprintf("[Content truncated after %d bytes by XELYON agent_instructions.max_total_bytes]", budget.maxTotalBytes))
	}

	if limit < 0 {
		limit = 0
	}

	prefix := truncateValidUTF8ByBytes(data, limit)
	consumed = len(prefix)
	budget.usedBytes += consumed

	content = string(prefix)
	if truncated {
		trimmed := strings.TrimRight(content, "\n")
		if trimmed == "" {
			content = strings.Join(notes, "\n")
		} else {
			content = trimmed + "\n\n" + strings.Join(notes, "\n")
		}
	}
	return content, truncated, consumed
}

func truncateValidUTF8ByBytes(data []byte, maxBytes int) []byte {
	if maxBytes <= 0 {
		return nil
	}
	if len(data) <= maxBytes {
		return data
	}

	chunk := data[:maxBytes]
	if utf8.Valid(chunk) {
		return chunk
	}

	for len(chunk) > 0 && !utf8.Valid(chunk) {
		_, size := utf8.DecodeLastRune(chunk)
		if size <= 0 {
			chunk = chunk[:len(chunk)-1]
			continue
		}
		chunk = chunk[:len(chunk)-size]
	}
	return chunk
}

func isLocalGuidanceFile(path string) bool {
	name := strings.ToLower(filepath.Base(strings.TrimSpace(path)))
	if name == "" {
		return false
	}
	if name == "claude.local.md" || name == "agents.local.md" {
		return true
	}
	return strings.HasSuffix(name, ".local.md")
}

func expandUserPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if path == "~" {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func findGitRoot(cwd string) string {
	if strings.TrimSpace(cwd) == "" {
		return ""
	}
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

func isGitTrackedInstructionFile(gitRoot, fullPath string) (tracked bool, known bool) {
	if strings.TrimSpace(gitRoot) == "" || strings.TrimSpace(fullPath) == "" {
		return false, false
	}
	rel, err := filepath.Rel(gitRoot, fullPath)
	if err != nil {
		return false, false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return false, false
	}
	cmd := exec.Command("git", "-C", gitRoot, "ls-files", "--error-unmatch", "--", rel)
	if err := cmd.Run(); err != nil {
		return false, true
	}
	return true, true
}
