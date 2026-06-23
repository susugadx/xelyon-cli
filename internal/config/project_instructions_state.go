package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ComputeProjectInstructionBundleFingerprintForDir は guidance bundle の変更検知キーを返す。
// 実際に bundle/prompt に反映される guidance 結果をベースに算出する。
func ComputeProjectInstructionBundleFingerprintForDir(cfg *Config, cwd string, previous *ProjectInstructionBundle) string {
	return ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, cwd, nil, previous)
}

// ComputeProjectInstructionBundleFingerprintForDirWithInputPaths は guidance bundle の
// 入力 path 選択を含む変更検知キーを返す。
func ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg *Config, cwd string, inputPaths []string, previous *ProjectInstructionBundle) string {
	if strings.TrimSpace(cwd) == "" {
		return ""
	}

	cfgForLoad := cfg
	if cfgForLoad == nil {
		cfgForLoad = DefaultConfig()
	}
	aiCfg := cfgForLoad.AgentInstructions
	builder := newProjectInstructionFingerprintBuilder()

	builder.Add("cwd", normalizedAbsolutePath(cwd))
	builder.Add("project.mode", normalizeAgentInstructionProjectMode(aiCfg.Project.Mode))
	builder.Add("project.include_gitignored", strconv.FormatBool(aiCfg.Project.IncludeGitignored))
	builder.Add("global.enabled", strconv.FormatBool(aiCfg.Global.Enabled))
	builder.Add("include_local_files", strconv.FormatBool(aiCfg.IncludeLocalFiles))
	builder.Add("expand_imports", strconv.FormatBool(aiCfg.ExpandImports))
	builder.Add("max_file_bytes", strconv.Itoa(aiCfg.MaxFileBytes))
	builder.Add("max_total_bytes", strconv.Itoa(aiCfg.MaxTotalBytes))

	xelyonPath, hasProjectConfigPath, _ := ResolveProjectConfigPathForDir(cwd)
	if !hasProjectConfigPath {
		xelyonPath = ""
	}
	builder.Add("xelyon", fileStampForFingerprint(xelyonPath))

	gitRoot := findGitRoot(cwd)
	builder.Add("git_root", normalizedAbsolutePath(gitRoot))

	var projectCfg *ProjectConfig
	if strings.TrimSpace(xelyonPath) != "" {
		projectCfg = &ProjectConfig{FilePath: xelyonPath}
	}
	root := resolveBundleRoot(cwd, projectCfg, gitRoot, aiCfg)
	if strings.TrimSpace(root.RootPath) == "" && previous != nil {
		root.RootPath = previous.RootPath
		root.Source = previous.RootSource
	}
	builder.Add("root", normalizedAbsolutePath(root.RootPath))
	builder.Add("root_source", string(root.Source))
	appendProjectInstructionInputScopesToFingerprint(builder, projectInstructionInputDirectoryScopes(root.RootPath, inputPaths))

	fingerprintBundle := loadGuidanceBundleForFingerprint(aiCfg, projectCfg, root.RootPath, root.Source, gitRoot, cwd, inputPaths)
	appendLoadedGuidanceToFingerprint(builder, "project", fingerprintBundle.ProjectGuidance)
	appendLoadedGuidanceToFingerprint(builder, "global", fingerprintBundle.GlobalGuidance)
	appendGuidanceWarningsToFingerprint(builder, fingerprintBundle.WarningEntries)

	return builder.Sum()
}

func loadGuidanceBundleForFingerprint(aiCfg AgentInstructionsConfig, projectCfg *ProjectConfig, rootPath string, rootSource ProjectInstructionRootSource, gitRoot string, cwd string, inputPaths []string) *ProjectInstructionBundle {
	bundle := &ProjectInstructionBundle{
		ProjectConfig: projectCfg,
		RootPath:      rootPath,
		RootSource:    rootSource,
	}
	budget := newInstructionByteBudget(aiCfg)
	mode := normalizeAgentInstructionProjectMode(aiCfg.Project.Mode)
	appendDeprecatedProjectModeFallbackWarning(bundle, mode)
	if shouldLoadProjectGuidance(mode) {
		strength := resolveProjectGuidanceStrength()
		bundle.ProjectGuidance = loadProjectGuidanceFiles(bundle, aiCfg, gitRoot, strength, &budget, cwd, inputPaths)
	}
	if aiCfg.Global.Enabled {
		bundle.GlobalGuidance = loadGlobalGuidanceFiles(bundle, aiCfg, &budget)
	}
	return bundle
}

func appendProjectInstructionInputScopesToFingerprint(builder *projectInstructionFingerprintBuilder, inputScopes []string) {
	if builder == nil || len(inputScopes) == 0 {
		return
	}
	next := 0
	for _, inputScope := range inputScopes {
		if normalizeRepositoryInstructionScope(inputScope) == "." {
			continue
		}
		builder.Add("input_scope."+strconv.Itoa(next), strings.TrimSpace(inputScope))
		next++
	}
}

func appendLoadedGuidanceToFingerprint(builder *projectInstructionFingerprintBuilder, scope string, files []InstructionFile) {
	if builder == nil {
		return
	}
	for i, file := range files {
		appendGuidanceFileToFingerprint(builder, scope+"."+strconv.Itoa(i), file)
	}
}

func appendGuidanceWarningsToFingerprint(builder *projectInstructionFingerprintBuilder, warnings []ProjectInstructionWarning) {
	if builder == nil {
		return
	}
	for i, warning := range warnings {
		prefix := "warning." + strconv.Itoa(i)
		builder.Add(prefix+".code", string(warning.Code))
		builder.Add(prefix+".message", strings.TrimSpace(warning.Message))
	}
}

func appendGuidanceFileToFingerprint(builder *projectInstructionFingerprintBuilder, prefix string, file InstructionFile) {
	if builder == nil {
		return
	}
	builder.Add(prefix+".path", normalizedAbsolutePath(file.Path))
	builder.Add(prefix+".label", file.Label)
	builder.Add(prefix+".repository_scope", file.RepositoryScope)
	builder.Add(prefix+".strength", string(file.Strength))
	builder.Add(prefix+".git_tracked", strconv.FormatBool(file.GitTracked))
	builder.Add(prefix+".truncated", strconv.FormatBool(file.Truncated))
	builder.Add(prefix+".content", file.Content)
}

type projectInstructionFingerprintBuilder struct {
	hasher hash.Hash
}

func newProjectInstructionFingerprintBuilder() *projectInstructionFingerprintBuilder {
	return &projectInstructionFingerprintBuilder{hasher: sha256.New()}
}

func (b *projectInstructionFingerprintBuilder) Add(key, value string) {
	if b == nil || b.hasher == nil {
		return
	}
	_, _ = b.hasher.Write([]byte(key))
	_, _ = b.hasher.Write([]byte("="))
	_, _ = b.hasher.Write([]byte(value))
	_, _ = b.hasher.Write([]byte("\x1f"))
}

func (b *projectInstructionFingerprintBuilder) Sum() string {
	if b == nil || b.hasher == nil {
		return ""
	}
	return hex.EncodeToString(b.hasher.Sum(nil))
}

func fileStampForFingerprint(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "-"
	}
	absPath := normalizedAbsolutePath(path)
	info, err := os.Stat(path)
	if err != nil {
		return absPath + "|missing"
	}
	return fmt.Sprintf("%s|%d|%d|%s", absPath, info.Size(), info.ModTime().UnixNano(), info.Mode().String())
}

func normalizedAbsolutePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return filepath.Clean(absPath)
}
