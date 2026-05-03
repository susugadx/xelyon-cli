package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
)

const runSkillScriptOutsideSkillScriptsDirectoryError = "script path is outside the skill scripts directory"

// validateResolvedSkillScriptPath は解決済み script path が
// 対象 skill の scripts ディレクトリ配下かを判定する。
func validateResolvedSkillScriptPath(skill skillcatalog.ParsedSkill, resolvedScriptPath string) error {
	scriptRoot, err := resolveSkillScriptsBoundaryRoot(skill)
	if err != nil {
		return err
	}

	resolvedPath, err := canonicalSkillScriptBoundaryPath(resolvedScriptPath)
	if err != nil {
		return fmt.Errorf("failed to resolve script path: %w", err)
	}

	if !pathWithinSkillScriptsRoot(resolvedPath, scriptRoot) {
		return fmt.Errorf("%s: %s", runSkillScriptOutsideSkillScriptsDirectoryError, resolvedPath)
	}
	return nil
}

func resolveSkillScriptsBoundaryRoot(skill skillcatalog.ParsedSkill) (string, error) {
	skillDir := strings.TrimSpace(skill.Directory)
	if skillDir == "" {
		return "", fmt.Errorf("failed to resolve skill scripts root: skill directory is empty")
	}

	scriptRoot, err := canonicalSkillScriptBoundaryPath(filepath.Join(skillDir, "scripts"))
	if err != nil {
		return "", fmt.Errorf("failed to resolve skill scripts root: %w", err)
	}

	info, statErr := os.Stat(scriptRoot)
	if statErr != nil {
		return "", fmt.Errorf("failed to stat skill scripts root: %w", statErr)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("skill scripts root is not a directory")
	}

	return scriptRoot, nil
}

func canonicalSkillScriptBoundaryPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		absPath = resolvedPath
	}
	return filepath.Clean(absPath), nil
}

func pathWithinSkillScriptsRoot(path, root string) bool {
	if path == root {
		return true
	}
	return strings.HasPrefix(path, root+string(filepath.Separator))
}
