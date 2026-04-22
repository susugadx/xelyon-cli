package navigation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

// findSymbolColumn は候補シンボル定義行の 1-origin 列番号を返す。
func findSymbolColumn(cand SymbolCandidate) (int, error) {
	absPath := candidateAbsPath(cand)
	if absPath == "" {
		return 1, fmt.Errorf("empty symbol path")
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return 1, err
	}

	lines := strings.Split(string(content), "\n")
	if cand.Line < 1 || cand.Line > len(lines) {
		return 1, fmt.Errorf("line %d out of range for %s", cand.Line, cand.File)
	}

	name := cand.Name
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	line := lines[cand.Line-1]
	if idx := strings.LastIndex(line, name); idx >= 0 {
		return idx + 1, nil
	}
	return 1, nil
}

// findEnclosingFunction は file/line を包含する関数スコープ名を返す。
func findEnclosingFunction(filePath string, line int) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	symbols, err := ast.ExtractSymbols(absPath)
	if err != nil {
		return "package-level"
	}
	for _, s := range symbols {
		if line < s.Line || line > s.EndLine {
			continue
		}
		switch s.Kind {
		case ast.SymbolFunction:
			return "func " + s.Name
		case ast.SymbolMethod:
			return "method " + s.Name
		}
	}
	return "package-level"
}

// findTypeNameAtLine は指定行で宣言される型名を推定して返す。
func findTypeNameAtLine(filePath string, line int) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	symbols, err := ast.ExtractSymbols(absPath)
	if err == nil {
		for _, s := range symbols {
			if s.Line != line {
				continue
			}
			switch s.Kind {
			case ast.SymbolType, ast.SymbolStruct, ast.SymbolInterface, ast.SymbolClass, ast.SymbolEnum, ast.SymbolTrait, ast.SymbolImpl:
				if s.Name != "" {
					return s.Name
				}
			}
		}
	}

	base := filepath.Base(absPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// isTestFile は Go のテストファイルかどうかを判定する。
func isTestFile(filePath string) bool {
	return strings.HasSuffix(filePath, "_test.go")
}

// readLineSnippet は指定行のテキストを trim して返す。
func readLineSnippet(filePath string, line int) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	if line < 1 || line > len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[line-1])
}

// classifyLineByAST は単一行を AST ヒューリスティックで分類する。
func classifyLineByAST(filePath string, line int, symbol string) ast.MatchClass {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	src, err := os.ReadFile(absPath)
	if err != nil {
		return ast.ClassUnknown
	}

	info, err := ast.ClassifyLine(absPath, src, line, symbol)
	if err != nil || info == nil {
		return ast.ClassUnknown
	}
	return info.Class
}
