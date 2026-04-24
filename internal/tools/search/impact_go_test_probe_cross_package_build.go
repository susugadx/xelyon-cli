package search

import (
	goast "go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func isImportableCrossPackageHelperFile(filePath string, src []byte, allowTestFiles bool) bool {
	if allowTestFiles {
		return true
	}
	if strings.HasSuffix(strings.TrimSpace(filePath), "_test.go") {
		return false
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, parser.PackageClauseOnly)
	if err != nil || file == nil || file.Name == nil {
		return false
	}
	return !strings.HasSuffix(strings.TrimSpace(file.Name.Name), "_test")
}

func newPackageHelper(absPath string, src []byte, parsed *ast.ParsedFile, candidate ast.Symbol) (packageHelper, bool) {
	switch candidate.Kind {
	case ast.SymbolFunction:
		if strings.HasPrefix(candidate.Name, "Test") {
			return packageHelper{}, false
		}
	case ast.SymbolMethod:
	default:
		return packageHelper{}, false
	}

	return packageHelper{
		key:      helperCacheKeyFromFields(absPath, candidate.Name, candidate.Line, candidate.EndLine),
		name:     candidate.Name,
		receiver: canonicalProbeReceiver(extractProbeMethodReceiver(candidate.Signature)),
		abs:      absPath,
		src:      src,
		parsed:   parsed,
		sym:      candidate,
	}, true
}

func fallbackCrossPackageHelpersFromBrokenFile(absPath string, src []byte) []packageHelper {
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, absPath, src, parser.AllErrors)

	helpers := make([]packageHelper, 0, 4)
	seen := make(map[string]struct{})
	add := func(name, receiver string, line, endLine int) {
		helper, ok := newFallbackPackageHelper(absPath, src, name, receiver, line, endLine)
		if !ok {
			return
		}
		if _, exists := seen[helper.key]; exists {
			return
		}
		seen[helper.key] = struct{}{}
		helpers = append(helpers, helper)
	}

	if file != nil {
		for _, decl := range file.Decls {
			fn, ok := decl.(*goast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}
			line := fset.Position(fn.Pos()).Line
			endLine := fset.Position(fn.End()).Line
			add(fn.Name.Name, fallbackReceiverFromFuncDecl(src, fset, fn), line, endLine)
		}
		if len(helpers) > 0 {
			return helpers
		}
	}

	srcText := string(src)
	re := regexp.MustCompile(`(?m)^func\s*(\(([^)]*)\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	for _, match := range re.FindAllStringSubmatchIndex(srcText, -1) {
		if len(match) < 8 {
			continue
		}
		receiver := ""
		if match[4] >= 0 && match[5] >= 0 {
			receiver = fallbackReceiverFromDeclText(srcText[match[4]:match[5]])
		}
		name := strings.TrimSpace(srcText[match[6]:match[7]])
		line := 1 + strings.Count(srcText[:match[0]], "\n")
		add(name, receiver, line, countLines(srcText))
	}
	return helpers
}

func newFallbackPackageHelper(absPath string, src []byte, name, receiver string, line, endLine int) (packageHelper, bool) {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, "Test") {
		return packageHelper{}, false
	}
	if line <= 0 {
		line = 1
	}
	totalLines := countLines(string(src))
	if endLine < line {
		endLine = totalLines
	}

	kind := ast.SymbolFunction
	if receiver != "" {
		kind = ast.SymbolMethod
	}

	return packageHelper{
		key:      helperCacheKeyFromFields(absPath, name, line, endLine),
		name:     name,
		receiver: canonicalProbeReceiver(receiver),
		abs:      absPath,
		src:      src,
		parsed:   nil,
		sym: ast.Symbol{
			Name:     name,
			Kind:     kind,
			Line:     line,
			EndLine:  endLine,
			Exported: isProbeExportedName(name),
		},
	}, true
}

func fallbackReceiverFromFuncDecl(src []byte, fset *token.FileSet, fn *goast.FuncDecl) string {
	if fn == nil || fn.Recv == nil || len(fn.Recv.List) == 0 || fset == nil {
		return ""
	}
	start := fset.Position(fn.Recv.List[0].Type.Pos()).Offset
	end := fset.Position(fn.Recv.List[0].Type.End()).Offset
	if start < 0 || end <= start || end > len(src) {
		return ""
	}
	return string(src[start:end])
}

func fallbackReceiverFromDeclText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func countLines(src string) int {
	if src == "" {
		return 1
	}
	return strings.Count(src, "\n") + 1
}

func isProbeExportedName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	r := rune(name[0])
	return r >= 'A' && r <= 'Z'
}

func structuredGoImpactProbeRootPath(opts SearchOptions, fallback string) string {
	if root := strings.TrimSpace(opts.ProjectMapRootPath); root != "" {
		return root
	}
	path := strings.TrimSpace(opts.Path)
	if path == "" {
		return fallback
	}
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return filepath.Dir(path)
	}
	return path
}
