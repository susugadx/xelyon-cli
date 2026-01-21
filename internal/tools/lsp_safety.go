package tools

import (
	"context"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

// SymbolInfo は削除対象のシンボル情報
type SymbolInfo struct {
	Name   string
	Line   int    // 1-indexed
	Column int    // 1-indexed
	Kind   string // func, type, var, const, class, etc.
}

// ReferenceInfo は参照情報
type ReferenceInfo struct {
	Symbol   string
	FilePath string
	Line     int  // 1-indexed
	IsLocal  bool // 同一ファイル内の参照
}

// symbolPatterns は言語ごとのシンボル抽出パターン
var symbolPatterns = map[string][]*regexp.Regexp{
	"go": {
		regexp.MustCompile(`func\s+(\w+)\s*\(`),                   // func FuncName(
		regexp.MustCompile(`func\s+\([^)]+\)\s*(\w+)\s*\(`),       // func (r *R) MethodName(
		regexp.MustCompile(`type\s+(\w+)\s+(?:struct|interface)`), // type TypeName struct/interface
		regexp.MustCompile(`var\s+(\w+)\s+`),                      // var VarName
		regexp.MustCompile(`const\s+(\w+)\s*=`),                   // const ConstName =
	},
	"typescript": {
		regexp.MustCompile(`function\s+(\w+)\s*[(<]`),                        // function funcName( or funcName<
		regexp.MustCompile(`(?:export\s+)?class\s+(\w+)`),                    // class ClassName
		regexp.MustCompile(`(?:export\s+)?interface\s+(\w+)`),                // interface InterfaceName
		regexp.MustCompile(`(?:export\s+)?(?:const|let|var)\s+(\w+)\s*[=:]`), // const constName =
		regexp.MustCompile(`(?:export\s+)?type\s+(\w+)\s*=`),                 // type TypeName =
	},
	"javascript": {
		regexp.MustCompile(`function\s+(\w+)\s*\(`),                       // function funcName(
		regexp.MustCompile(`class\s+(\w+)`),                               // class ClassName
		regexp.MustCompile(`(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=`), // const constName =
	},
	"python": {
		regexp.MustCompile(`def\s+(\w+)\s*\(`), // def func_name(
		regexp.MustCompile(`class\s+(\w+)`),    // class ClassName
	},
	"rust": {
		regexp.MustCompile(`fn\s+(\w+)\s*[(<]`),           // fn func_name( or func_name<
		regexp.MustCompile(`(?:pub\s+)?struct\s+(\w+)`),   // struct StructName
		regexp.MustCompile(`(?:pub\s+)?enum\s+(\w+)`),     // enum EnumName
		regexp.MustCompile(`(?:pub\s+)?trait\s+(\w+)`),    // trait TraitName
		regexp.MustCompile(`(?:pub\s+)?type\s+(\w+)\s*=`), // type TypeName =
	},
}

// ExtractSymbolsFromContent はファイル内容からシンボル（関数/型/変数）を抽出
func ExtractSymbolsFromContent(content, language string) []SymbolInfo {
	patterns, ok := symbolPatterns[language]
	if !ok {
		return nil
	}

	var symbols []SymbolInfo
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		for _, pattern := range patterns {
			matches := pattern.FindStringSubmatchIndex(line)
			if len(matches) >= 4 {
				// matches[2:4] はキャプチャグループの位置
				name := line[matches[2]:matches[3]]
				symbols = append(symbols, SymbolInfo{
					Name:   name,
					Line:   lineNum + 1, // 1-indexed
					Column: matches[2] + 1,
					Kind:   detectKind(pattern.String()),
				})
			}
		}
	}

	return symbols
}

// detectKind はパターンからシンボルの種類を推定
func detectKind(pattern string) string {
	if strings.Contains(pattern, "func") || strings.Contains(pattern, "function") || strings.Contains(pattern, "def") || strings.Contains(pattern, "fn") {
		return "function"
	}
	if strings.Contains(pattern, "class") || strings.Contains(pattern, "struct") || strings.Contains(pattern, "interface") || strings.Contains(pattern, "trait") {
		return "type"
	}
	if strings.Contains(pattern, "type") {
		return "type"
	}
	if strings.Contains(pattern, "const") || strings.Contains(pattern, "var") || strings.Contains(pattern, "let") {
		return "variable"
	}
	if strings.Contains(pattern, "enum") {
		return "enum"
	}
	return "symbol"
}

// CheckReferencesBeforeDelete はLSPで参照をチェック
// 戻り値: 参照リスト, 外部参照あり?, エラー
func CheckReferencesBeforeDelete(filePath string, symbols []SymbolInfo) ([]ReferenceInfo, bool, error) {
	if LSPClient == nil {
		return nil, false, nil
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	var allRefs []ReferenceInfo
	hasExternal := false

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, sym := range symbols {
		// LSPで参照を検索
		locations, err := LSPClient.FindReferences(ctx, absPath, sym.Line, sym.Column, true)
		if err != nil {
			continue // エラーは無視して続行
		}

		for _, loc := range locations {
			refPath := lsp.URIToFile(loc.URI)
			refAbsPath, _ := filepath.Abs(refPath)

			isLocal := refAbsPath == absPath

			ref := ReferenceInfo{
				Symbol:   sym.Name,
				FilePath: refPath,
				Line:     loc.Range.Start.Line + 1, // 1-indexed
				IsLocal:  isLocal,
			}
			allRefs = append(allRefs, ref)

			if !isLocal {
				hasExternal = true
			}
		}
	}

	return allRefs, hasExternal, nil
}

// ExtractSymbolsFromOldStr は削除される文字列からシンボルを抽出
func ExtractSymbolsFromOldStr(oldStr, language string) []SymbolInfo {
	return ExtractSymbolsFromContent(oldStr, language)
}

// GetExternalReferences は外部参照のみをフィルタして返す
func GetExternalReferences(refs []ReferenceInfo) []ReferenceInfo {
	var external []ReferenceInfo
	for _, ref := range refs {
		if !ref.IsLocal {
			external = append(external, ref)
		}
	}
	return external
}

// FormatReferenceWarning は参照警告メッセージをフォーマット
func FormatReferenceWarning(refs []ReferenceInfo) string {
	external := GetExternalReferences(refs)
	if len(external) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("⚠️ Warning: The following symbols are referenced elsewhere:\n")

	// シンボルごとにグループ化
	symbolRefs := make(map[string][]ReferenceInfo)
	for _, ref := range external {
		symbolRefs[ref.Symbol] = append(symbolRefs[ref.Symbol], ref)
	}

	for symbol, symbolRefList := range symbolRefs {
		sb.WriteString("   " + symbol + ":\n")
		shown := 0
		for _, ref := range symbolRefList {
			if shown >= 5 {
				sb.WriteString("      ... and more\n")
				break
			}
			sb.WriteString("      - " + ref.FilePath + ":" + strconv.Itoa(ref.Line) + "\n")
			shown++
		}
	}

	return sb.String()
}
