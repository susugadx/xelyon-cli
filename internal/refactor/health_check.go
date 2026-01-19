package refactor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// isSourceFile はソースファイルかどうかを判定
func isSourceFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	sourceExts := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true,
		".jsx": true, ".tsx": true, ".java": true, ".rs": true,
		".c": true, ".cpp": true, ".h": true, ".hpp": true,
	}
	return sourceExts[ext]
}

// funcInfo holds information about a function.
type funcInfo struct {
	name      string
	lineStart int
	lineEnd   int
	lines     int
}

// extractFunctions extracts function information from source code.
func extractFunctions(content string, ext string) []funcInfo {
	var funcs []funcInfo
	lines := strings.Split(content, "\n")

	var funcPattern *regexp.Regexp
	switch ext {
	case ".go":
		funcPattern = regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(`)
	case ".js", ".ts", ".jsx", ".tsx":
		funcPattern = regexp.MustCompile(`(?:function\s+(\w+)|(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?(?:function|\([^)]*\)\s*=>|\w+\s*=>))`)
	case ".py":
		funcPattern = regexp.MustCompile(`^def\s+(\w+)\s*\(`)
	case ".java":
		funcPattern = regexp.MustCompile(`(?:public|private|protected)?\s*(?:static\s+)?(?:\w+\s+)?(\w+)\s*\([^)]*\)\s*(?:throws\s+\w+)?\s*\{`)
	case ".rs":
		funcPattern = regexp.MustCompile(`^(?:pub\s+)?fn\s+(\w+)`)
	default:
		return nil
	}

	braceCount := 0
	inFunc := false
	var currentFunc funcInfo

	for i, line := range lines {
		lineNum := i + 1

		if !inFunc {
			matches := funcPattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				// Find the first non-empty capture group
				name := ""
				for _, m := range matches[1:] {
					if m != "" {
						name = m
						break
					}
				}
				if name != "" {
					currentFunc = funcInfo{
						name:      name,
						lineStart: lineNum,
					}
					inFunc = true
					braceCount = strings.Count(line, "{") - strings.Count(line, "}")
					if braceCount <= 0 && strings.Contains(line, "{") {
						// Single-line function
						currentFunc.lineEnd = lineNum
						currentFunc.lines = 1
						funcs = append(funcs, currentFunc)
						inFunc = false
					}
				}
			}
		} else {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				currentFunc.lineEnd = lineNum
				currentFunc.lines = currentFunc.lineEnd - currentFunc.lineStart + 1
				funcs = append(funcs, currentFunc)
				inFunc = false
			}
		}
	}

	return funcs
}
