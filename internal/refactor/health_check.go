package refactor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HealthCheckResult はコード健全性チェックの結果
type HealthCheckResult struct {
	FilePath         string
	HasWarning       bool
	FileLines        int
	MaxFileLines     int
	LongFunctions    []LongFunctionInfo
	MaxFunctionLines int
	Suggestions      []string
}

// LongFunctionInfo は長い関数の情報
type LongFunctionInfo struct {
	Name      string
	Lines     int
	LineStart int
}

// HealthCheckConfig はチェック設定
type HealthCheckConfig struct {
	Enabled          bool
	MaxFileLines     int
	MaxFunctionLines int
	CheckFileSize    bool
	CheckFuncSize    bool
	CheckDuplication bool
}

// DefaultHealthCheckConfig はデフォルト設定を返す
func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Enabled:          true,
		MaxFileLines:     300,
		MaxFunctionLines: 50,
		CheckFileSize:    true,
		CheckFuncSize:    true,
		CheckDuplication: false,
	}
}

// CheckFileHealth はファイルの健全性をチェックする
func CheckFileHealth(filePath string, config HealthCheckConfig) *HealthCheckResult {
	if !config.Enabled {
		return nil
	}

	result := &HealthCheckResult{
		FilePath:         filePath,
		MaxFileLines:     config.MaxFileLines,
		MaxFunctionLines: config.MaxFunctionLines,
	}

	// ファイル読み込み
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	code := string(content)
	lines := strings.Split(code, "\n")
	result.FileLines = len(lines)

	// ファイルサイズチェック
	if config.CheckFileSize && result.FileLines >= config.MaxFileLines {
		result.HasWarning = true
		result.Suggestions = append(result.Suggestions,
			fmt.Sprintf("File has %d lines (limit: %d). Consider splitting into multiple files.",
				result.FileLines, config.MaxFileLines))
	}

	// 関数サイズチェック
	if config.CheckFuncSize {
		ext := strings.ToLower(filepath.Ext(filePath))
		funcs := extractFunctions(code, ext)
		for _, f := range funcs {
			if f.lines >= config.MaxFunctionLines {
				result.HasWarning = true
				result.LongFunctions = append(result.LongFunctions, LongFunctionInfo{
					Name:      f.name,
					Lines:     f.lines,
					LineStart: f.lineStart,
				})
				result.Suggestions = append(result.Suggestions,
					fmt.Sprintf("Function '%s' has %d lines (limit: %d). Consider extracting helper methods.",
						f.name, f.lines, config.MaxFunctionLines))
			}
		}
	}

	return result
}

// FormatHealthWarning はヘルスチェック結果を警告メッセージとしてフォーマットする
func FormatHealthWarning(result *HealthCheckResult) string {
	if result == nil || !result.HasWarning {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n⚠️  Code Health Warning: %s\n", result.FilePath))

	if result.FileLines >= result.MaxFileLines {
		sb.WriteString(fmt.Sprintf("   📦 File size: %d lines (limit: %d)\n",
			result.FileLines, result.MaxFileLines))
	}

	for _, f := range result.LongFunctions {
		sb.WriteString(fmt.Sprintf("   📏 Long function: %s() - %d lines (limit: %d)\n",
			f.Name, f.Lines, result.MaxFunctionLines))
	}

	sb.WriteString("\n💡 Suggestions:\n")
	for _, s := range result.Suggestions {
		sb.WriteString(fmt.Sprintf("   • %s\n", s))
	}

	return sb.String()
}

// ShouldCheckHealth はファイルが健全性チェック対象かを判定
func ShouldCheckHealth(filePath string) bool {
	return isSourceFile(filePath)
}
