package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfig はプロジェクト固有の設定（xelyon.yaml）
type ProjectConfig struct {
	Context string       `yaml:"context"`         // AI に注入するプロジェクトコンテキスト
	Rules   []string     `yaml:"rules"`           // 必須ルール（番号付きで system prompt に注入）
	Hooks   *HooksConfig `yaml:"hooks,omitempty"` // 完了時フック（config.yaml の hooks を上書き）

	FilePath string `yaml:"-"` // ロード元ファイルパス
	IsLegacy bool   `yaml:"-"` // XELYON.md フォールバックの場合 true
}

// LoadProjectConfig はプロジェクト設定をロードする。
// cwd から親方向に xelyon.yaml を探索し、なければ XELYON.md にフォールバックする。
// どちらも見つからない場合は nil を返す。
func LoadProjectConfig() *ProjectConfig {
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}

	// 1. xelyon.yaml を探索
	if path := findFileUpward(dir, "xelyon.yaml"); path != "" {
		pc, err := loadProjectConfigFromYAML(path)
		if err != nil {
			// パースエラー時は nil を返す（呼び出し元で警告表示）
			return nil
		}
		return pc
	}

	// 2. XELYON.md フォールバック
	if path := findFileUpward(dir, "XELYON.md"); path != "" {
		return loadProjectConfigFromMD(path)
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
	pc.IsLegacy = false
	return pc, nil
}

// loadProjectConfigFromMD は XELYON.md の全文を Context に格納して返す。
func loadProjectConfigFromMD(path string) *ProjectConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	return &ProjectConfig{
		Context:  string(data),
		FilePath: path,
		IsLegacy: true,
	}
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
	if globalCfg != nil && len(globalCfg.Hooks.OnCompletion) > 0 {
		return &globalCfg.Hooks
	}

	return nil
}
