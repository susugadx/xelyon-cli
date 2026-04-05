package search

import (
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func symbolCallsPackageHelper(sym, helper packageHelper) bool {
	if symbolCallsDirectHelper(sym, helper) {
		return true
	}
	for _, alias := range helperReferenceAliases(sym, helper) {
		if symbolCallsAlias(sym, alias) {
			return true
		}
	}
	return false
}

func symbolCallsDirectHelper(sym, helper packageHelper) bool {
	body := symbolBodyFromLines(sym.src, sym.sym)
	if sym.parsed == nil {
		for _, line := range findPackageHelperCallLines(body, helper.name, sym.sym.Line) {
			if helper.receiver == "" && bodyLineContainsBareHelperCall(body, line, sym.sym.Line, helper.name) {
				return true
			}
			if helper.receiver != "" && bodyLineContainsMethodHelperCall(body, line, sym.sym.Line, helper.name) {
				return true
			}
		}
		return false
	}
	for _, line := range findPackageHelperCallLines(body, helper.name, sym.sym.Line) {
		info, err := ast.ClassifyLineWithParsed(sym.parsed, line, helper.name)
		if err != nil || info == nil || info.Class != ast.ClassCall {
			continue
		}
		if helper.receiver == "" {
			if info.SelectorKind == "package" || info.SelectorKind == "method" {
				continue
			}
			if info.NodeType == "identifier" || info.SelectorKind == "" {
				return true
			}
			continue
		}
		if info.SelectorKind != "method" {
			continue
		}
		if canonicalProbeReceiver(info.ReceiverType) == helper.receiver {
			return true
		}
	}
	return false
}

func helperReferenceAliases(sym, helper packageHelper) []string {
	body := symbolBodyFromLines(sym.src, sym.sym)
	lines := strings.Split(body, "\n")
	aliases := make(map[string]struct{})
	for idx, line := range lines {
		if !strings.Contains(line, helper.name) {
			continue
		}
		alias := helperAssignedAlias(line)
		if alias == "" {
			continue
		}
		if sym.parsed == nil {
			aliases[alias] = struct{}{}
			continue
		}
		info, err := ast.ClassifyLineWithParsed(sym.parsed, sym.sym.Line+idx, helper.name)
		if err != nil || info == nil {
			continue
		}
		if helper.receiver == "" {
			if info.SelectorKind == "package" || info.SelectorKind == "method" {
				continue
			}
			aliases[alias] = struct{}{}
			continue
		}
		if info.SelectorKind != "method" {
			continue
		}
		if canonicalProbeReceiver(info.ReceiverType) == helper.receiver {
			aliases[alias] = struct{}{}
		}
	}
	return sortedAliasList(aliases)
}

func symbolCallsImportedPackageHelper(sym packageHelper, qualifier string, helper packageHelper) bool {
	if symbolCallsQualifiedHelper(sym, qualifier, helper.name) {
		return true
	}
	for _, alias := range qualifiedHelperReferenceAliases(sym, qualifier, helper.name) {
		if symbolCallsAlias(sym, alias) {
			return true
		}
	}
	return false
}

func symbolCallsQualifiedHelper(sym packageHelper, qualifier, helperName string) bool {
	body := symbolBodyFromLines(sym.src, sym.sym)
	for _, line := range findQualifiedPackageHelperCallLines(body, qualifier, helperName, sym.sym.Line) {
		info, err := ast.ClassifyLineWithParsed(sym.parsed, line, helperName)
		if err != nil || info == nil || info.Class != ast.ClassCall {
			continue
		}
		if info.SelectorKind == "package" {
			return true
		}
	}
	return false
}

func qualifiedHelperReferenceAliases(sym packageHelper, qualifier, helperName string) []string {
	body := symbolBodyFromLines(sym.src, sym.sym)
	lines := strings.Split(body, "\n")
	aliases := make(map[string]struct{})
	for idx, line := range lines {
		if !strings.Contains(line, qualifier) || !strings.Contains(line, helperName) {
			continue
		}
		alias := helperAssignedAlias(line)
		if alias == "" {
			continue
		}
		info, err := ast.ClassifyLineWithParsed(sym.parsed, sym.sym.Line+idx, helperName)
		if err != nil || info == nil {
			continue
		}
		if info.SelectorKind == "package" {
			aliases[alias] = struct{}{}
		}
	}
	return sortedAliasList(aliases)
}

func sortedAliasList(aliases map[string]struct{}) []string {
	if len(aliases) == 0 {
		return nil
	}
	list := make([]string, 0, len(aliases))
	for alias := range aliases {
		list = append(list, alias)
	}
	sort.Strings(list)
	return list
}

func helperAssignedAlias(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	re := regexp.MustCompile(`^(?:var\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*(?::=|=)`)
	match := re.FindStringSubmatch(line)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func symbolCallsAlias(sym packageHelper, alias string) bool {
	body := symbolBodyFromLines(sym.src, sym.sym)
	if sym.parsed == nil {
		for _, line := range findPackageHelperCallLines(body, alias, sym.sym.Line) {
			if bodyLineContainsBareHelperCall(body, line, sym.sym.Line, alias) {
				return true
			}
		}
		return false
	}
	for _, line := range findPackageHelperCallLines(body, alias, sym.sym.Line) {
		info, err := ast.ClassifyLineWithParsed(sym.parsed, line, alias)
		if err != nil || info == nil || info.Class != ast.ClassCall {
			continue
		}
		if info.SelectorKind == "package" || info.SelectorKind == "method" {
			continue
		}
		if info.NodeType == "identifier" || info.SelectorKind == "" {
			return true
		}
	}
	return false
}

func localHelperImportDirs(currentPath string, currentSrc []byte, opts SearchOptions) map[string]string {
	currentPath = strings.TrimSpace(currentPath)
	if currentPath == "" || len(currentSrc) == 0 {
		return nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, currentPath, currentSrc, parser.ImportsOnly)
	if err != nil || file == nil {
		return nil
	}

	basePath := structuredGoImpactImportResolveBasePath(opts, currentPath)
	if basePath == "" {
		return nil
	}

	imports := make(map[string]string)
	for _, spec := range file.Imports {
		if spec == nil || spec.Path == nil {
			continue
		}
		importPath := strings.Trim(spec.Path.Value, "\"")
		if importPath == "" {
			continue
		}
		name := ""
		if spec.Name != nil {
			name = strings.TrimSpace(spec.Name.Name)
		}
		if name == "." || name == "_" {
			continue
		}
		if name == "" {
			name = defaultQualifiedReceiverImportName(importPath, currentPath, opts)
		}
		if name == "" {
			continue
		}
		localDir, _ := resolveLocalImportPathHint(basePath, importPath)
		if localDir == "" {
			continue
		}
		imports[name] = localDir
	}
	return imports
}

func findPackageHelperCallLines(body, name string, startLine int) []int {
	return findHelperReferenceLines(body, `\b`+regexp.QuoteMeta(name)+`\s*\(`, startLine)
}

func findQualifiedPackageHelperCallLines(body, qualifier, name string, startLine int) []int {
	pattern := `\b` + regexp.QuoteMeta(qualifier) + `\s*\.\s*` + regexp.QuoteMeta(name) + `\s*\(`
	return findHelperReferenceLines(body, pattern, startLine)
}

func findHelperReferenceLines(body, pattern string, startLine int) []int {
	if body == "" || pattern == "" || startLine <= 0 {
		return nil
	}

	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringIndex(body, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[int]struct{}, len(matches))
	lines := make([]int, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		line := startLine + strings.Count(body[:match[0]], "\n")
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		lines = append(lines, line)
	}
	sort.Ints(lines)
	return lines
}

func bodyLineContainsBareHelperCall(body string, line, startLine int, name string) bool {
	lineText := helperProbeLineText(body, line, startLine)
	if lineText == "" {
		return false
	}
	re := regexp.MustCompile(`(^|[^.\w])` + regexp.QuoteMeta(name) + `\s*\(`)
	return re.MatchString(lineText)
}

func bodyLineContainsMethodHelperCall(body string, line, startLine int, name string) bool {
	lineText := helperProbeLineText(body, line, startLine)
	if lineText == "" {
		return false
	}
	re := regexp.MustCompile(`\.\s*` + regexp.QuoteMeta(name) + `\s*\(`)
	return re.MatchString(lineText)
}

func helperProbeLineText(body string, line, startLine int) string {
	if body == "" || line < startLine || startLine <= 0 {
		return ""
	}
	lines := strings.Split(body, "\n")
	idx := line - startLine
	if idx < 0 || idx >= len(lines) {
		return ""
	}
	return lines[idx]
}
