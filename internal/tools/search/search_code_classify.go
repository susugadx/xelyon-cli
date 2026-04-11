package search

import (
	"os"
	"strings"

	internalast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

var controlFlowKeywords = map[string]bool{
	"if": true, "else": true, "for": true, "while": true,
	"switch": true, "case": true, "return": true, "break": true,
	"continue": true, "throw": true, "try": true, "catch": true,
	"finally": true, "select": true, "range": true, "yield": true, "await": true,
}

// classifyMatch はマッチ行の内容から種別を判定する（言語非依存の汎用パターン）
func classifyMatch(line string) MatchType {
	trimmed := strings.TrimSpace(line)

	if common.IsCommentLine(trimmed) {
		return MatchTypeComment
	}

	importKeywords := []string{"import ", "from ", "use ", "include ", "using "}
	for _, kw := range importKeywords {
		if strings.HasPrefix(trimmed, kw) {
			return MatchTypeImport
		}
	}
	if strings.Contains(trimmed, "require(") {
		return MatchTypeImport
	}

	strippedTrimmed := common.StripModifiers(trimmed)

	defKeywords := []string{
		"func ", "fn ", "def ", "function ", "sub ", "method ",
		"type ", "class ", "struct ", "interface ", "enum ", "trait ", "impl ",
		"const ", "var ", "let ", "static ", "pub ", "export ",
		"module ", "namespace ", "package ",
	}
	for _, kw := range defKeywords {
		if strings.HasPrefix(strippedTrimmed, kw) {
			return MatchTypeDefinition
		}
	}

	parts := strings.Fields(strippedTrimmed)
	if len(parts) > 0 {
		firstWord := parts[0]
		if controlFlowKeywords[firstWord] {
			return MatchTypeRef
		}

		if len(parts) >= 2 {
			rest := strings.TrimSpace(strippedTrimmed[len(firstWord):])
			parenIdx := strings.Index(rest, "(")
			if parenIdx > 0 {
				idCandidate := strings.TrimSpace(rest[:parenIdx])
				if isValidIdentifier(idCandidate) {
					return MatchTypeDefinition
				}
			}
		}
	}

	if hasAssignment(line) {
		return MatchTypeAssignment
	}

	return MatchTypeRef
}

// reclassifyWithAST は Go ファイルの検索結果を AST ベースで再分類する。
// 非 Go ファイル、または targetName を抽出できない regex パターンはスキップする。
func reclassifyWithAST(results []SearchResult, pattern string, isRegex bool, opts SearchOptions) {
	targetName := extractTargetName(pattern, isRegex)
	if targetName == "" {
		return
	}

	for i := range results {
		r := &results[i]
		absPath := absoluteAffectedFilePath(r.FilePath, opts, affectedFileSourceText)
		if absPath == "" {
			absPath = r.FilePath
		}
		if !internalast.IsSupportedFile(absPath) {
			continue
		}

		src, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		for j := range r.Matches {
			m := &r.Matches[j]
			if !m.IsMatch {
				continue
			}

			info, err := internalast.ClassifyLine(absPath, src, m.LineNum, targetName)
			if err != nil {
				continue
			}

			m.Type = matchClassToType(info.Class)
			if info.Scope != "" && info.Scope != "package-level" && info.Class != internalast.ClassDef {
				m.Block = &BlockInfo{Name: info.Scope}
			}
		}
	}
}

// extractTargetName は pattern から AST 分類用のターゲット名を抽出する。
func extractTargetName(pattern string, isRegex bool) string {
	if pattern == "" {
		return ""
	}
	if !isRegex {
		return pattern
	}
	if strings.ContainsAny(pattern, `\.^$*+?()[]{}|`) {
		return ""
	}
	return pattern
}

// matchClassToType は AST の MatchClass を search_code の MatchType に変換する。
func matchClassToType(class internalast.MatchClass) MatchType {
	switch class {
	case internalast.ClassDef:
		return MatchTypeDefinition
	case internalast.ClassImport:
		return MatchTypeImport
	case internalast.ClassCall:
		return MatchTypeCall
	case internalast.ClassComment:
		return MatchTypeComment
	case internalast.ClassString:
		return MatchTypeString
	case internalast.ClassRef:
		return MatchTypeRef
	default:
		return MatchTypeRef
	}
}

// isValidIdentifier は関数名などの識別子として妥当か判定する（ジェネリクス <> [] 対応）
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	idx := strings.IndexAny(s, "<[")
	if idx >= 0 {
		s = s[:idx]
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	for i, r := range s {
		if i == 0 {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && r != '_' && r < 0x80 {
				return false
			}
		} else {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r < 0x80 {
				return false
			}
		}
	}
	return true
}

// hasAssignment は行に代入演算子が含まれるか判定する（比較演算子・文字列内を除外）
func hasAssignment(line string) bool {
	s := common.StripQuoted(line)
	for _, op := range []string{"===", "!==", "==", "!=", ">=", "<=", "=>"} {
		s = strings.ReplaceAll(s, op, "")
	}
	return strings.Contains(s, ":=") || strings.Contains(s, "=")
}
