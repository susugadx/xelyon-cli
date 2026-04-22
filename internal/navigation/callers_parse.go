package navigation

import (
	goast "go/ast"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

type parsedRipgrepReferenceLine struct {
	AbsPath string
	RelPath string
	Line    int
	Snippet string
	IsTest  bool
}

type referenceClassification struct {
	Scope        string
	Class        ast.MatchClass
	NodeType     string
	SelectorKind string
	ReceiverType string
}

// parseRipgrepLine は "file:line:content" 行を parse して分類情報付き参照に変換する。
func parseRipgrepLine(line, symbol string, cache *referenceParseCache) *Reference {
	parsed, ok := parseRipgrepReferenceLine(line)
	if !ok {
		return nil
	}

	classification := classifyParsedReferenceLine(parsed, symbol, cache)
	classification = applySnippetCompletionHints(parsed.Snippet, symbol, classification)
	return buildReferenceFromParsedLine(parsed, classification)
}

// parseRipgrepReferenceLine は ripgrep 1 行を参照の基本形へ変換する。
func parseRipgrepReferenceLine(line string) (parsedRipgrepReferenceLine, bool) {
	firstColon := strings.Index(line, ":")
	if firstColon < 0 {
		return parsedRipgrepReferenceLine{}, false
	}
	rest := line[firstColon+1:]
	secondColon := strings.Index(rest, ":")
	if secondColon < 0 {
		return parsedRipgrepReferenceLine{}, false
	}

	filePath := line[:firstColon]
	lineNum, err := strconv.Atoi(rest[:secondColon])
	if err != nil || lineNum <= 0 {
		return parsedRipgrepReferenceLine{}, false
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}
	return parsedRipgrepReferenceLine{
		AbsPath: absPath,
		RelPath: toRelativePath(absPath),
		Line:    lineNum,
		Snippet: strings.TrimSpace(rest[secondColon+1:]),
		IsTest:  strings.HasSuffix(filePath, "_test.go"),
	}, true
}

// classifyParsedReferenceLine は parse 済み参照に AST ベース分類を適用する。
func classifyParsedReferenceLine(parsed parsedRipgrepReferenceLine, symbol string, cache *referenceParseCache) referenceClassification {
	classification := referenceClassification{Class: ast.ClassUnknown}
	if symbol == "" || cache == nil || !ast.IsSupportedFile(parsed.AbsPath) {
		return classification
	}

	cf := cache.get(parsed.AbsPath)
	if cf == nil {
		return classification
	}

	classification = classifyParsedLineWithTreeSitter(cf, parsed, symbol, classification)
	classification = classifyParsedLineWithGoASTHints(cf, parsed, symbol, classification)
	return classification
}

// classifyParsedLineWithTreeSitter は tree-sitter 分類結果を反映する。
func classifyParsedLineWithTreeSitter(cf *cachedFile, parsed parsedRipgrepReferenceLine, symbol string, current referenceClassification) referenceClassification {
	if pf := cf.ensureTreeSitter(parsed.AbsPath); pf != nil {
		if info, err := ast.ClassifyLineWithParsed(pf, parsed.Line, symbol); err == nil && info != nil {
			current.Scope = info.Scope
			current.Class = info.Class
			current.NodeType = info.NodeType
			current.SelectorKind = info.SelectorKind
			current.ReceiverType = info.ReceiverType
		}
	}
	return current
}

// classifyParsedLineWithGoASTHints は go/parser 分類をヒントとして補完適用する。
func classifyParsedLineWithGoASTHints(cf *cachedFile, parsed parsedRipgrepReferenceLine, symbol string, current referenceClassification) referenceClassification {
	if goFile, goFSet, goImports := cf.ensureGoParser(parsed.AbsPath); goFile != nil {
		return applyGoParserReferenceHints(goFile, goFSet, goImports, parsed.Line, symbol, current)
	}
	return current
}

// applySnippetCompletionHints は snippet から不足した分類情報を補完する。
func applySnippetCompletionHints(snippet, symbol string, current referenceClassification) referenceClassification {
	current.Class, current.NodeType, current.SelectorKind, current.ReceiverType = applySnippetReferenceHints(
		snippet,
		symbol,
		current.Class,
		current.NodeType,
		current.SelectorKind,
		current.ReceiverType,
	)
	return current
}

// buildReferenceFromParsedLine は parse + classify 結果から最終参照構造体を組み立てる。
func buildReferenceFromParsedLine(parsed parsedRipgrepReferenceLine, classification referenceClassification) *Reference {
	return &Reference{
		File:         parsed.RelPath,
		ResolvedPath: cleanNavigationResolvedPath(parsed.AbsPath),
		Line:         parsed.Line,
		Scope:        classification.Scope,
		Snippet:      parsed.Snippet,
		IsTest:       parsed.IsTest,
		Class:        classification.Class,
		NodeType:     classification.NodeType,
		SelectorKind: classification.SelectorKind,
		ReceiverType: classification.ReceiverType,
	}
}

func applyGoParserReferenceHints(file *goast.File, fset *token.FileSet, imports map[string]bool, line int, symbol string, current referenceClassification) referenceClassification {
	fallback, ok := classifyLineWithGoAST(file, fset, imports, line, symbol)
	if !ok {
		return current
	}

	if (current.Scope == "" || current.Scope == "package-level") && fallback.Scope != "" {
		current.Scope = fallback.Scope
	}
	if fallback.Class == ast.ClassDef {
		current.Class = ast.ClassDef
	} else if current.Class == ast.ClassUnknown && fallback.Class != ast.ClassUnknown {
		current.Class = fallback.Class
	}
	if current.NodeType == "" && fallback.NodeType != "" {
		current.NodeType = fallback.NodeType
	}
	if (current.SelectorKind == "" || current.SelectorKind == "unknown") && fallback.SelectorKind != "" {
		current.SelectorKind = fallback.SelectorKind
	}
	if current.ReceiverType == "" && fallback.ReceiverType != "" {
		current.ReceiverType = fallback.ReceiverType
	}

	return current
}
