package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfig はプロジェクト固有の設定（xelyon.yaml）
type ProjectConfig struct {
	Context     string                    `yaml:"context"`               // AI に注入するプロジェクトコンテキスト
	Rules       []string                  `yaml:"rules"`                 // 必須ルール（番号付きで system prompt に注入）
	Conditional []ProjectConditionalBlock `yaml:"conditional,omitempty"` // 条件付きで注入する rules/context
	Ignore      ProjectIgnoreConfig       `yaml:"ignore,omitempty"`      // repomap/list_dir/search_code で共有する ignore 設定
	Hooks       *HooksConfig              `yaml:"hooks,omitempty"`       // 完了時フック（config.yaml の hooks を上書き）

	FilePath string `yaml:"-"` // ロード元ファイルパス
}

// ProjectConditionalBlock は条件に応じて注入する rules/context のまとまり。
type ProjectConditionalBlock struct {
	Name    string   `yaml:"name,omitempty"`    // 任意の表示名
	Paths   []string `yaml:"paths,omitempty"`   // 対象パス glob
	Rules   []string `yaml:"rules,omitempty"`   // 条件一致時のみ注入するルール
	Context string   `yaml:"context,omitempty"` // 条件一致時のみ注入するコンテキスト
}

// ProjectIgnoreConfig はプロジェクト共通 ignore 設定。
type ProjectIgnoreConfig struct {
	Patterns []string `yaml:"patterns,omitempty"` // ignore 対象のパターン
}

// LoadProjectConfig はプロジェクト設定をロードする。
// cwd から親方向に xelyon.yaml を探索する。見つからない場合は nil を返す。
func LoadProjectConfig() *ProjectConfig {
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}

	if path := findFileUpward(dir, "xelyon.yaml"); path != "" {
		pc, err := loadProjectConfigFromYAML(path)
		if err != nil {
			return nil
		}
		return pc
	}

	return nil
}

// findFileUpward は dir から親方向に filename を探索し、見つかったフルパスを返す。
// 見つからない場合は空文字を返す。
func findFileUpward(dir, filename string) string {
	for {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// loadProjectConfigFromYAML は xelyon.yaml をパースして ProjectConfig を返す。
func loadProjectConfigFromYAML(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pc := &ProjectConfig{}
	if err := yaml.Unmarshal(data, pc); err != nil {
		return nil, err
	}

	pc.FilePath = path
	return pc, nil
}

// SaveProjectConfig は ProjectConfig を xelyon.yaml に保存する。
// pc.FilePath が設定されていればそのパスに、なければ ./xelyon.yaml に保存。
func SaveProjectConfig(pc *ProjectConfig) error {
	path := pc.FilePath
	if path == "" {
		path = "xelyon.yaml"
	}

	data, err := yaml.Marshal(pc)
	if err != nil {
		return fmt.Errorf("failed to marshal project config: %w", err)
	}

	projectName := filepath.Base(filepath.Dir(path))
	if path == "xelyon.yaml" {
		if cwd, err := os.Getwd(); err == nil {
			projectName = filepath.Base(cwd)
		}
	}

	header := fmt.Sprintf("# %s - Project Configuration\n# AI 用コンテキスト。ドキュメントではありません。\n\n", projectName)
	fullData := []byte(header + string(data))

	if err := os.WriteFile(path, fullData, 0644); err != nil {
		return fmt.Errorf("failed to write xelyon.yaml: %w", err)
	}

	return nil
}

// ResolveHooks はプロジェクト設定とグローバル設定から hooks を解決する。
// 優先順位:
//  1. xelyon.yaml に hooks あり → それを使う
//  2. xelyon.yaml に hooks なし → config.yaml の hooks を使う
//  3. どちらもなし → nil
func ResolveHooks(globalCfg *Config, projectCfg *ProjectConfig) *HooksConfig {
	// xelyon.yaml の hooks が設定されている場合はそれを優先
	if projectCfg != nil && projectCfg.Hooks != nil {
		return projectCfg.Hooks
	}

	// config.yaml の hooks にフォールバック
	if globalCfg != nil && (len(globalCfg.Hooks.OnCompletion) > 0 || len(globalCfg.Hooks.OnStepComplete) > 0) {
		return &globalCfg.Hooks
	}

	return nil
}
