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
// path / mtime / size / mode を主に使い、expand_imports=true の場合のみ import 依存も追跡する。
func ComputeProjectInstructionBundleFingerprintForDir(cfg *Config, cwd string, previous *ProjectInstructionBundle) string {
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

	xelyonPath := findFileUpward(cwd, "xelyon.yaml")
	builder.Add("xelyon", fileStampForFingerprint(xelyonPath))

	gitRoot := findGitRoot(cwd)
	builder.Add("git_root", normalizedAbsolutePath(gitRoot))

	var projectCfg *ProjectConfig
	if strings.TrimSpace(xelyonPath) != "" {
		projectCfg = &ProjectConfig{FilePath: xelyonPath}
	}
	rootPath := resolveBundleRootPath(cwd, projectCfg, gitRoot, aiCfg)
	if strings.TrimSpace(rootPath) == "" && previous != nil {
		rootPath = previous.RootPath
	}
	builder.Add("root", normalizedAbsolutePath(rootPath))

	projectPlans := buildProjectGuidanceLoadPlans(rootPath, aiCfg, gitRoot, resolveProjectGuidanceStrength(projectCfg != nil), nil)
	appendGuidanceLoadPlansToFingerprint(builder, "project", projectPlans)
	globalPlans := buildGlobalGuidanceLoadPlans(aiCfg, nil)
	appendGuidanceLoadPlansToFingerprint(builder, "global", globalPlans)

	return builder.Sum()
}

func appendGuidanceLoadPlansToFingerprint(builder *projectInstructionFingerprintBuilder, scope string, plans []guidanceLoadPlan) {
	if builder == nil {
		return
	}
	for _, plan := range plans {
		builder.Add(scope+".file", plan.CandidatePath)
		if !plan.Valid {
			builder.Add(scope+".path", "invalid")
			if plan.Warning != nil {
				builder.Add(scope+".warning", plan.Warning.Message)
			}
			continue
		}

		builder.Add(scope+".stamp", fileStampForFingerprint(plan.LoadOptions.FullPath))
		for _, importedPath := range collectInstructionImportDependencies(plan.LoadOptions) {
			builder.Add(scope+".import.stamp", fileStampForFingerprint(importedPath))
		}
	}
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
