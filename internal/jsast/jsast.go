package jsast

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
	codeast "github.com/susugadx/xelyon-cli/internal/ast"
)

type Language string

const (
	LanguageJavaScript Language = "javascript"
	LanguageTypeScript Language = "typescript"
	LanguageTSX        Language = "tsx"
)

type Symbol struct {
	Name      string
	Kind      string
	Signature string
	Line      int
	EndLine   int
	Character int
	Exported  bool
}

type MatchInfo struct {
	Class codeast.MatchClass
	Scope string
}

type ParsedFile struct {
	tree    *gotreesitter.Tree
	src     []byte
	lang    Language
	grammar *gotreesitter.Language
}

var jsTreeSitterMu sync.Mutex

func Supports(path string) bool {
	_, ok := languageForPath(path)
	return ok
}

func ParseBytes(path string, src []byte) (*ParsedFile, error) {
	lang, ok := languageForPath(path)
	if !ok {
		return nil, fmt.Errorf("unsupported JS family language: %s", path)
	}

	grammar := treeSitterLanguage(lang)
	if grammar == nil {
		return nil, fmt.Errorf("unsupported JS family language: %s", path)
	}
	tree, err := parseJSFamilySource(grammar, src)
	if err != nil {
		return nil, err
	}
	if tree == nil {
		return nil, fmt.Errorf("parse returned nil tree")
	}
	return &ParsedFile{tree: tree, src: src, lang: lang, grammar: grammar}, nil
}

func parseJSFamilySource(grammar *gotreesitter.Language, src []byte) (*gotreesitter.Tree, error) {
	parser := gotreesitter.NewParser(grammar)
	jsTreeSitterMu.Lock()
	tree, err := parser.Parse(src)
	jsTreeSitterMu.Unlock()
	if err != nil {
		return nil, err
	}
	normalized := normalizeJSFamilySourceForParse(src)
	if len(normalized) == 0 || bytesEqual(normalized, src) || jsFamilyTreeUsable(tree) {
		return tree, nil
	}
	// gotreesitter の TSX grammar が async arrow で error root になる場合だけ、
	// byte offset を保った正規化 source で再 parse する。
	parser = gotreesitter.NewParser(grammar)
	jsTreeSitterMu.Lock()
	normalizedTree, normalizedErr := parser.Parse(normalized)
	jsTreeSitterMu.Unlock()
	if normalizedErr != nil || !jsFamilyTreeUsable(normalizedTree) {
		if normalizedTree != nil {
			normalizedTree.Release()
		}
		return tree, nil
	}
	if tree != nil {
		tree.Release()
	}
	return normalizedTree, nil
}

func jsFamilyTreeUsable(tree *gotreesitter.Tree) bool {
	if tree == nil || tree.RootNode() == nil {
		return false
	}
	root := tree.RootNode()
	return root.IsNamed() && root.NamedChildCount() > 0 && !root.HasError()
}

func (p *ParsedFile) Close() {
	if p != nil && p.tree != nil {
		p.tree.Release()
		p.tree = nil
	}
}

func ExtractSymbols(path string, src []byte) ([]Symbol, error) {
	parsed, err := ParseBytes(path, src)
	if err != nil {
		return nil, err
	}
	defer parsed.Close()
	return ExtractSymbolsWithParsed(parsed), nil
}

func ExtractSymbolsWithParsed(parsed *ParsedFile) []Symbol {
	if parsed == nil || parsed.tree == nil {
		return nil
	}
	var symbols []Symbol
	seen := make(map[string]struct{})
	walkNamed(parsed.tree.RootNode(), func(node *gotreesitter.Node) {
		if symbol, ok := symbolFromNode(parsed, node); ok {
			appendSymbol(&symbols, seen, symbol)
		}
	})
	for _, symbol := range fallbackSymbolsFromSourceWithOptions(parsed, fallbackSymbolOptions{includeTypeBodyMembers: !jsFamilyTreeUsable(parsed.tree)}) {
		if symbolAtLineExists(symbols, symbol) {
			continue
		}
		appendSymbol(&symbols, seen, symbol)
	}
	sort.SliceStable(symbols, func(i, j int) bool {
		if symbols[i].Line != symbols[j].Line {
			return symbols[i].Line < symbols[j].Line
		}
		if symbols[i].EndLine != symbols[j].EndLine {
			return symbols[i].EndLine < symbols[j].EndLine
		}
		return symbols[i].Name < symbols[j].Name
	})
	return symbols
}

func ClassifyLine(path string, src []byte, line int, targetName string) (*MatchInfo, error) {
	parsed, err := ParseBytes(path, src)
	if err != nil {
		return nil, err
	}
	defer parsed.Close()
	return ClassifyLineWithParsed(parsed, line, targetName)
}

func ClassifyLineWithParsed(parsed *ParsedFile, line int, targetName string) (*MatchInfo, error) {
	if parsed == nil || parsed.tree == nil {
		return nil, fmt.Errorf("ParsedFile is nil")
	}
	if line <= 0 {
		return nil, fmt.Errorf("line must be >= 1: %d", line)
	}
	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		return nil, fmt.Errorf("targetName is required")
	}

	startByte, endByte, ok := lineByteRange(parsed.src, line)
	if !ok {
		return &MatchInfo{Class: codeast.ClassUnknown, Scope: "package-level"}, nil
	}

	root := parsed.tree.RootNode()
	occurrences := findIdentifierOccurrences(parsed.src[startByte:endByte], targetName)
	var best *MatchInfo
	for _, offset := range occurrences {
		absStart := startByte + offset
		absEnd := absStart + uint(len(targetName))
		if sourceInCommentAt(parsed.src, absStart) {
			info := &MatchInfo{Class: codeast.ClassComment, Scope: "package-level"}
			if best == nil || matchClassPriority(info.Class) > matchClassPriority(best.Class) {
				best = info
			}
			continue
		}
		node := root.DescendantForByteRange(uint32(absStart), uint32(absEnd))
		if node == nil {
			continue
		}
		class := classifyNode(parsed, root, node, uint32(absStart), uint32(absEnd), targetName)
		info := &MatchInfo{
			Class: class,
			Scope: findEnclosingScope(parsed, node),
		}
		if best == nil || matchClassPriority(info.Class) > matchClassPriority(best.Class) {
			best = info
		}
	}
	if best != nil {
		return best, nil
	}
	return &MatchInfo{Class: codeast.ClassUnknown, Scope: "package-level"}, nil
}

func ClassifyRangeWithParsed(parsed *ParsedFile, line, character, endLine, endCharacter int, targetName string) (*MatchInfo, error) {
	if parsed == nil || parsed.tree == nil {
		return nil, fmt.Errorf("ParsedFile is nil")
	}
	startByte, endByte, ok := byteRangeForLSPRange(parsed.src, line, character, endLine, endCharacter)
	if !ok {
		return &MatchInfo{Class: codeast.ClassUnknown, Scope: "package-level"}, nil
	}
	root := parsed.tree.RootNode()
	node := root.DescendantForByteRange(uint32(startByte), uint32(endByte))
	if node == nil {
		return &MatchInfo{Class: codeast.ClassUnknown, Scope: "package-level"}, nil
	}
	classifyName := strings.TrimSpace(nodeText(parsed, node))
	if classifyName == "" {
		classifyName = strings.TrimSpace(targetName)
	}
	return &MatchInfo{
		Class: classifyNode(parsed, root, node, uint32(startByte), uint32(endByte), classifyName),
		Scope: findEnclosingScope(parsed, node),
	}, nil
}

func languageForPath(path string) (Language, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js", ".mjs", ".cjs":
		return LanguageJavaScript, true
	case ".ts":
		return LanguageTypeScript, true
	case ".tsx", ".jsx":
		return LanguageTSX, true
	default:
		return "", false
	}
}

func treeSitterLanguage(lang Language) *gotreesitter.Language {
	switch lang {
	case LanguageJavaScript, LanguageTypeScript, LanguageTSX:
		// gotreesitter の JS/TS grammar は core build で一部構文の tree が崩れるため、
		// JS family は最も広く parse できる TSX grammar を共通 owner にする。
		return grammars.TsxLanguage()
	default:
		return nil
	}
}

func walkNamed(node *gotreesitter.Node, visit func(*gotreesitter.Node)) {
	if node == nil {
		return
	}
	visit(node)
	for i := 0; i < node.NamedChildCount(); i++ {
		walkNamed(node.NamedChild(i), visit)
	}
}

func appendSymbol(symbols *[]Symbol, seen map[string]struct{}, symbol Symbol) {
	if symbol.Name == "" || symbol.Line <= 0 {
		return
	}
	key := fmt.Sprintf("%d:%s:%s", symbol.Line, symbol.Name, symbol.Kind)
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*symbols = append(*symbols, symbol)
}

func symbolAtLineExists(symbols []Symbol, candidate Symbol) bool {
	for _, symbol := range symbols {
		if symbol.Line == candidate.Line && symbol.Name == candidate.Name {
			return true
		}
	}
	return false
}
