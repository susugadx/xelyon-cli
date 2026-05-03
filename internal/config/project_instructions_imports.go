package config

import (
	"path/filepath"
	"strings"
)

type instructionImportExpansionResult struct {
	Content       []byte
	ImportedPaths []string
	Warnings      []ProjectInstructionWarning
}

func expandInstructionImports(opts instructionFileLoadOptions, data []byte) instructionImportExpansionResult {
	return expandInstructionImportsWithCollector(opts, data, nil)
}

func expandInstructionImportsWithCollector(opts instructionFileLoadOptions, data []byte, collector []string) instructionImportExpansionResult {
	if len(data) == 0 {
		return instructionImportExpansionResult{Content: data, ImportedPaths: collector}
	}
	lines := strings.Split(string(data), "\n")
	var out strings.Builder
	importedPaths := collector
	var warnings []ProjectInstructionWarning
	for i, line := range lines {
		replacedLine, lineImportedPaths, lineWarnings := expandInstructionImportLine(opts, line)
		importedPaths = append(importedPaths, lineImportedPaths...)
		warnings = append(warnings, lineWarnings...)
		out.WriteString(replacedLine)
		if i < len(lines)-1 {
			out.WriteByte('\n')
		}
	}
	return instructionImportExpansionResult{
		Content:       []byte(out.String()),
		ImportedPaths: importedPaths,
		Warnings:      warnings,
	}
}

func expandInstructionImportLine(opts instructionFileLoadOptions, line string) (string, []string, []ProjectInstructionWarning) {
	expandedPath, ok := resolveInstructionImportPath(opts, line)
	if !ok {
		return line, nil, nil
	}

	importedPaths := []string{expandedPath}
	childOpts := opts.withImportPath(expandedPath)
	importedData, _, loadResult, loaded := loadInstructionSource(childOpts)
	if !loaded {
		warnings := convertImportLoadResultToWarnings(loadResult)
		return line, importedPaths, warnings
	}

	childExpanded := expandInstructionImportsWithCollector(childOpts, importedData, importedPaths)
	return string(childExpanded.Content), childExpanded.ImportedPaths, childExpanded.Warnings
}

func resolveInstructionImportPath(opts instructionFileLoadOptions, line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "@") {
		return "", false
	}
	importPath := strings.TrimSpace(strings.TrimPrefix(trimmed, "@"))
	if importPath == "" {
		return "", false
	}

	expandedPath := expandUserPath(importPath)
	if !filepath.IsAbs(expandedPath) {
		expandedPath = filepath.Join(filepath.Dir(opts.FullPath), filepath.FromSlash(expandedPath))
	}
	expandedPath = filepath.Clean(expandedPath)
	if containsString(opts.Traversal.ImportStack, expandedPath) {
		return "", false
	}
	if !opts.Policy.RootBoundary.ContainsPath(expandedPath) {
		return "", false
	}
	return expandedPath, true
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func convertImportLoadResultToWarnings(result instructionLoadResult) []ProjectInstructionWarning {
	warnings := append([]ProjectInstructionWarning(nil), result.Warnings...)
	message := strings.TrimSpace(result.Warning)
	if message == "" {
		return warnings
	}
	warnings = append(warnings, ProjectInstructionWarning{
		Code:    ProjectInstructionWarningImportLoadSkipped,
		Message: message,
	})
	return warnings
}
