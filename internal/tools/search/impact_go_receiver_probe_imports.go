package search

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func probeReceiverIsQualified(receiver string) bool {
	_, qualified := splitProbeReceiverQualifier(receiver)
	return qualified
}

func splitProbeReceiverQualifier(receiver string) (string, bool) {
	receiver = strings.TrimSpace(receiver)
	for strings.HasPrefix(receiver, "(") && strings.HasSuffix(receiver, ")") && len(receiver) > 1 {
		receiver = strings.TrimSpace(receiver[1 : len(receiver)-1])
	}
	if idx := strings.LastIndexAny(receiver, " \t"); idx >= 0 {
		receiver = strings.TrimSpace(receiver[idx+1:])
	}
	receiver = strings.TrimSpace(strings.TrimPrefix(receiver, "*"))
	receiver = stripProbeTypeArguments(receiver)
	if idx := strings.LastIndex(receiver, "."); idx >= 0 {
		return strings.TrimSpace(receiver[idx+1:]), true
	}
	return receiver, false
}

func splitProbeReceiverImportQualifier(receiver string) (string, string, bool) {
	receiver = strings.TrimSpace(receiver)
	for strings.HasPrefix(receiver, "(") && strings.HasSuffix(receiver, ")") && len(receiver) > 1 {
		receiver = strings.TrimSpace(receiver[1 : len(receiver)-1])
	}
	if idx := strings.LastIndexAny(receiver, " \t"); idx >= 0 {
		receiver = strings.TrimSpace(receiver[idx+1:])
	}
	receiver = strings.TrimSpace(strings.TrimPrefix(receiver, "*"))
	receiver = stripProbeTypeArguments(receiver)
	if idx := strings.LastIndex(receiver, "."); idx >= 0 {
		return strings.TrimSpace(receiver[:idx]), strings.TrimSpace(receiver[idx+1:]), true
	}
	return "", receiver, false
}

func resolveQualifiedReceiverImportPath(qualifier, currentPath string, currentSrc []byte, opts SearchOptions) string {
	qualifier = strings.TrimSpace(qualifier)
	currentPath = strings.TrimSpace(currentPath)
	if qualifier == "" || currentPath == "" || len(currentSrc) == 0 {
		return ""
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, currentPath, currentSrc, parser.ImportsOnly)
	if err != nil || file == nil {
		return ""
	}
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
		if name != "" && name == qualifier {
			return importPath
		}
	}
	return ""
}

func defaultQualifiedReceiverImportName(importPath, currentPath string, opts SearchOptions) string {
	basePath := structuredGoImpactImportResolveBasePath(opts, currentPath)
	if localDir, _ := resolveLocalImportPathHint(basePath, importPath); localDir != "" {
		return goPackageNameInDir(localDir)
	}
	return ""
}

func resolveLocalImportPathHint(rootPath, importPath string) (string, string) {
	rootPath = normalizeStructuredGoImpactProbeBase(rootPath)
	importPath = strings.Trim(strings.TrimSpace(importPath), "/")
	if rootPath == "" || importPath == "" {
		return "", ""
	}

	for dir := rootPath; dir != ""; {
		if localDir := resolveLocalImportPathHintAtRoot(dir, importPath); localDir != "" {
			return localDir, dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", ""
}

func resolveLocalImportPathHintAtRoot(rootPath, importPath string) string {
	modulePath := goModulePathAtRoot(rootPath)
	if modulePath == "" {
		return resolveDomainlessLocalImportPathHintAtRoot(rootPath, importPath)
	}
	relPath := ""
	switch {
	case importPath == modulePath:
		relPath = ""
	case strings.HasPrefix(importPath, modulePath+"/"):
		relPath = strings.TrimPrefix(importPath, modulePath+"/")
	default:
		return ""
	}
	candidate := rootPath
	if relPath != "" {
		candidate = filepath.Join(rootPath, filepath.FromSlash(relPath))
	}
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return ""
}

func resolveDomainlessLocalImportPathHintAtRoot(rootPath, importPath string) string {
	parts := strings.Split(importPath, "/")
	if len(parts) < 2 || strings.Contains(parts[0], ".") {
		return ""
	}
	candidate := filepath.Join(rootPath, filepath.Join(parts[1:]...))
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return ""
}

func structuredGoImpactImportResolveBasePath(opts SearchOptions, currentPath string) string {
	candidates := []string{
		strings.TrimSpace(opts.ProjectMapRootPath),
		"",
		strings.TrimSpace(currentPath),
		strings.TrimSpace(opts.Path),
		strings.TrimSpace(opts.InvocationCWD),
	}
	if opts.ProjectMap != nil {
		candidates[1] = strings.TrimSpace(opts.ProjectMap.RootPath)
	}
	for _, candidate := range candidates {
		if base := normalizeStructuredGoImpactProbeBase(candidate); base != "" {
			return base
		}
	}
	return ""
}

func normalizeStructuredGoImpactProbeBase(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return filepath.Dir(path)
	}
	return filepath.Clean(path)
}

func goModulePathAtRoot(rootPath string) string {
	goModPath := filepath.Join(rootPath, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func goPackageNameInDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
		if err != nil || file == nil || file.Name == nil {
			continue
		}
		pkg := strings.TrimSpace(file.Name.Name)
		if pkg != "" {
			return pkg
		}
	}
	return ""
}

func stripProbeTypeArguments(receiver string) string {
	if receiver == "" {
		return ""
	}

	var b strings.Builder
	depth := 0
	for _, r := range receiver {
		switch r {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func extractProbeMethodReceiver(signature string) string {
	signature = strings.TrimSpace(signature)
	if !strings.HasPrefix(signature, "func") {
		return ""
	}

	rest := strings.TrimSpace(strings.TrimPrefix(signature, "func"))
	if !strings.HasPrefix(rest, "(") {
		return ""
	}

	closeIdx := strings.Index(rest, ")")
	if closeIdx <= 1 {
		return ""
	}

	receiverSpec := strings.TrimSpace(rest[1:closeIdx])
	if receiverSpec == "" {
		return ""
	}
	fields := strings.Fields(receiverSpec)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}
