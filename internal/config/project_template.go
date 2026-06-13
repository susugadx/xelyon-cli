package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrProjectConfigExists は xelyon.yaml が既に存在する場合のエラー。
var ErrProjectConfigExists = errors.New("xelyon.yaml already exists")

// ErrProjectAgentInstructionsExists は AGENTS.md が既に存在する場合のエラー。
var ErrProjectAgentInstructionsExists = errors.New("AGENTS.md already exists")

const projectYAMLTemplate = `# XELYON repo config for %s

# Project Map / list_dir / search_code で共有する ignore パターン
# ignore:
#   patterns:
#     - dist
#     - generated
#
# 明示完了時の final checks（省略時は config.yaml の final_checks を使用）
# final_checks:
#   commands:
#     - make ci-check
#   timeout: 600
`

const projectAgentInstructionsTemplate = `# AGENTS.md

## Project Overview
- TODO: このリポジトリの目的を短く書く。

## Commands
- Test: TODO
- Lint: TODO
- Build: TODO

## Working Rules
- 既存の設計、命名、テスト方針に合わせる。
- 変更後は関係するテストまたはチェックを実行する。
`

// CreateProjectConfigTemplate は xelyon.yaml のテンプレートを作成する。
func CreateProjectConfigTemplate(path string, overwrite bool) error {
	if path == "" {
		path = "xelyon.yaml"
	}
	if _, err := os.Stat(path); err == nil && !overwrite {
		return ErrProjectConfigExists
	}

	projectName := projectNameForTemplate(path)
	content := fmt.Sprintf(projectYAMLTemplate, projectName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write xelyon.yaml: %w", err)
	}
	return nil
}

// CreateProjectAgentInstructionsTemplate は repo-local AGENTS.md の雛形を作成する。
func CreateProjectAgentInstructionsTemplate(path string) error {
	if path == "" {
		resolvedPath, err := ResolveProjectAgentInstructionsTemplatePathForDir("")
		if err != nil {
			return err
		}
		path = resolvedPath
	}
	if _, err := os.Stat(path); err == nil {
		return ErrProjectAgentInstructionsExists
	}
	if err := os.WriteFile(path, []byte(projectAgentInstructionsTemplate), 0o644); err != nil {
		return fmt.Errorf("failed to write AGENTS.md: %w", err)
	}
	return nil
}

// CreateDefaultProjectAgentInstructionsTemplate は loader と同じ project root に AGENTS.md の雛形を作成する。
func CreateDefaultProjectAgentInstructionsTemplate() (string, error) {
	path, err := ResolveProjectAgentInstructionsTemplatePathForDir("")
	if err != nil {
		return "", err
	}
	return path, CreateProjectAgentInstructionsTemplate(path)
}

// ResolveProjectAgentInstructionsTemplatePathForDir は /init の既定 AGENTS.md 作成先を返す。
func ResolveProjectAgentInstructionsTemplatePathForDir(cwd string) (string, error) {
	dir, err := resolveProjectSearchDir(cwd)
	if err != nil {
		return "", err
	}
	projectCfg, err := projectConfigRootOnlyForDir(dir)
	if err != nil {
		return "", err
	}
	gitRoot := findGitRoot(dir)
	root := resolveBundleRoot(dir, projectCfg, gitRoot, DefaultConfig().AgentInstructions)
	if strings.TrimSpace(root.RootPath) == "" {
		root.RootPath = dir
	}
	return filepath.Join(root.RootPath, "AGENTS.md"), nil
}

func projectConfigRootOnlyForDir(cwd string) (*ProjectConfig, error) {
	path, ok, err := ResolveProjectConfigPathForDir(cwd)
	if err != nil || !ok {
		return nil, err
	}
	return &ProjectConfig{FilePath: path}, nil
}

func projectNameForTemplate(path string) string {
	if path == "" || path == "xelyon.yaml" {
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Base(cwd)
		}
		return "project"
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Base(cwd)
		}
		return "project"
	}
	return filepath.Base(dir)
}
