package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProjectConfig はプロジェクト固有の設定（xelyon.yaml）
type ProjectConfig struct {
	Context     string                    `yaml:"context"`                // AI に注入するプロジェクトコンテキスト
	Rules       []string                  `yaml:"rules"`                  // 必須ルール（番号付きで system prompt に注入）
	Conditional []ProjectConditionalBlock `yaml:"conditional,omitempty"`  // 条件付きで注入する rules/context
	Ignore      ProjectIgnoreConfig       `yaml:"ignore,omitempty"`       // repomap/list_dir/search_code で共有する ignore 設定
	FinalChecks *FinalChecksConfig        `yaml:"final_checks,omitempty"` // 明示完了時 final checks（config.yaml の final_checks を上書き）

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
	pc, err := LoadProjectConfigWithError()
	if err != nil {
		return nil
	}
	return pc
}

// LoadProjectConfigWithError はプロジェクト設定をロードし、読み込み失敗を返す。
// cwd から親方向に xelyon.yaml を探索する。見つからない場合は nil, nil を返す。
func LoadProjectConfigWithError() (*ProjectConfig, error) {
	return LoadProjectConfigForDirWithError("")
}

// LoadProjectConfigForDir は指定 cwd から親方向に xelyon.yaml を探索して読み込む。
// cwd が空の場合は現在の process cwd を使う。見つからない場合は nil を返す。
func LoadProjectConfigForDir(cwd string) *ProjectConfig {
	pc, err := LoadProjectConfigForDirWithError(cwd)
	if err != nil {
		return nil
	}
	return pc
}

// LoadProjectConfigForDirWithError は指定 cwd から親方向に xelyon.yaml を探索して読み込む。
// cwd が空の場合は現在の process cwd を使う。見つからない場合は nil, nil を返す。
func LoadProjectConfigForDirWithError(cwd string) (*ProjectConfig, error) {
	path, ok, err := ResolveProjectConfigPathForDir(cwd)
	if err != nil {
		return nil, err
	}
	if ok {
		pc, err := loadProjectConfigFromYAML(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", path, err)
		}
		return pc, nil
	}

	return nil, nil
}

// ResolveProjectConfigPathForDir は指定 cwd から親方向に xelyon.yaml を探索し、
// 見つかった設定ファイルの絶対パスを返す。
func ResolveProjectConfigPathForDir(cwd string) (string, bool, error) {
	dir, err := resolveProjectSearchDir(cwd)
	if err != nil {
		return "", false, err
	}
	path := findFileUpward(dir, "xelyon.yaml")
	if strings.TrimSpace(path) == "" {
		return "", false, nil
	}
	return normalizeProjectConfigPath(path), true, nil
}

func resolveProjectSearchDir(cwd string) (string, error) {
	dir := strings.TrimSpace(cwd)
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	return normalizeProjectConfigPath(dir), nil
}

func normalizeProjectConfigPath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
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
	finalChecks, err := loadCompatibleFinalChecks(data)
	if err != nil {
		return nil, err
	}
	pc.FinalChecks = finalChecks

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

// ResolveFinalChecks はプロジェクト設定とグローバル設定から final checks を解決する。
// 優先順位:
//  1. xelyon.yaml に final_checks あり → それを使う
//  2. xelyon.yaml に final_checks なし → config.yaml の final_checks を使う
//  3. どちらもなし → nil
func ResolveFinalChecks(globalCfg *Config, projectCfg *ProjectConfig) *FinalChecksConfig {
	// xelyon.yaml の final_checks が設定されている場合はそれを優先
	if projectCfg != nil && projectCfg.FinalChecks != nil {
		return projectCfg.FinalChecks
	}

	// config.yaml の final_checks にフォールバック
	if globalCfg != nil && len(globalCfg.FinalChecks.Commands) > 0 {
		return &globalCfg.FinalChecks
	}

	return nil
}
