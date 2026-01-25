package lsp

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	lsplib "github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

var (
	green  = common.Green
	red    = common.Red
	yellow = common.Yellow
	cyan   = common.Cyan
)

// LSPClient is set by agent during initialization
var LSPClient *lsplib.Client

// LSPToolTimeout is the timeout for LSP operations
const LSPToolTimeout = 30 * time.Second

// ===== lsp_references Tool =====

// LSPReferencesTool finds all references to a symbol
type LSPReferencesTool struct{}

func (t *LSPReferencesTool) Name() string { return "lsp_references" }

func (t *LSPReferencesTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	if LSPClient == nil {
		return "LSP not available. Please configure LSP servers in ~/.xelyon/config.yaml", nil, nil
	}

	path := args["path"]
	if path == "" {
		return "Error: path is required", nil, fmt.Errorf("path is required")
	}

	line, err := strconv.Atoi(args["line"])
	if err != nil || line < 1 {
		return "Error: line must be a positive number (1-indexed)", nil, fmt.Errorf("line must be a positive number")
	}

	character, err := strconv.Atoi(args["character"])
	if err != nil || character < 1 {
		return "Error: character must be a positive number (1-indexed)", nil, fmt.Errorf("character must be a positive number")
	}

	// Validate path
	absPath, err := common.ValidatePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}

	// Include declaration by default
	includeDecl := args["include_declaration"] != "false"

	ctx, cancel := context.WithTimeout(context.Background(), LSPToolTimeout)
	defer cancel()

	locations, err := LSPClient.FindReferences(ctx, absPath, line, character, includeDecl)
	if err != nil {
		// Graceful fallback
		return fmt.Sprintf("LSP references not available: %v\nTip: Use search_code to find occurrences instead.", err), nil, nil
	}

	if len(locations) == 0 {
		return "No references found", nil, nil
	}

	// Format output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d references:\n\n", len(locations)))
	for i, loc := range locations {
		if i >= 50 {
			sb.WriteString(fmt.Sprintf("\n... and %d more", len(locations)-50))
			break
		}
		filePath := lsplib.URIToFile(loc.URI)
		sb.WriteString(fmt.Sprintf("  %s:%d:%d\n", filePath, loc.Range.Start.Line+1, loc.Range.Start.Character+1))
	}

	green.Printf("🔍 LSP: Found %d references\n", len(locations))
	return sb.String(), nil, nil
}

// ===== lsp_definition Tool =====

// LSPDefinitionTool finds the definition of a symbol
type LSPDefinitionTool struct{}

func (t *LSPDefinitionTool) Name() string { return "lsp_definition" }

func (t *LSPDefinitionTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	if LSPClient == nil {
		return "LSP not available. Please configure LSP servers in ~/.xelyon/config.yaml", nil, nil
	}

	path := args["path"]
	if path == "" {
		return "Error: path is required", nil, fmt.Errorf("path is required")
	}

	line, err := strconv.Atoi(args["line"])
	if err != nil || line < 1 {
		return "Error: line must be a positive number (1-indexed)", nil, fmt.Errorf("line must be a positive number")
	}

	character, err := strconv.Atoi(args["character"])
	if err != nil || character < 1 {
		return "Error: character must be a positive number (1-indexed)", nil, fmt.Errorf("character must be a positive number")
	}

	absPath, err := common.ValidatePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), LSPToolTimeout)
	defer cancel()

	locations, err := LSPClient.GoToDefinition(ctx, absPath, line, character)
	if err != nil {
		return fmt.Sprintf("LSP definition not available: %v\nTip: Use search_code to find definitions.", err), nil, nil
	}

	if len(locations) == 0 {
		return "Definition not found", nil, nil
	}

	// Format output (usually just one location)
	var sb strings.Builder
	sb.WriteString("Definition found:\n\n")
	for _, loc := range locations {
		filePath := lsplib.URIToFile(loc.URI)
		sb.WriteString(fmt.Sprintf("  %s:%d:%d\n", filePath, loc.Range.Start.Line+1, loc.Range.Start.Character+1))
	}

	green.Printf("🔍 LSP: Found definition\n")
	return sb.String(), nil, nil
}

// ===== lsp_hover Tool =====

// LSPHoverTool gets type information and documentation for a symbol
type LSPHoverTool struct{}

func (t *LSPHoverTool) Name() string { return "lsp_hover" }

func (t *LSPHoverTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	if LSPClient == nil {
		return "LSP not available. Please configure LSP servers in ~/.xelyon/config.yaml", nil, nil
	}

	path := args["path"]
	if path == "" {
		return "Error: path is required", nil, fmt.Errorf("path is required")
	}

	line, err := strconv.Atoi(args["line"])
	if err != nil || line < 1 {
		return "Error: line must be a positive number (1-indexed)", nil, fmt.Errorf("line must be a positive number")
	}

	character, err := strconv.Atoi(args["character"])
	if err != nil || character < 1 {
		return "Error: character must be a positive number (1-indexed)", nil, fmt.Errorf("character must be a positive number")
	}

	absPath, err := common.ValidatePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), LSPToolTimeout)
	defer cancel()

	hover, err := LSPClient.GetHover(ctx, absPath, line, character)
	if err != nil {
		return fmt.Sprintf("LSP hover not available: %v", err), nil, nil
	}

	if hover == nil || hover.Contents.Value == "" {
		return "No hover information available", nil, nil
	}

	green.Printf("🔍 LSP: Got hover info\n")
	return hover.Contents.Value, nil, nil
}

// ===== lsp_diagnostics Tool =====

// LSPDiagnosticsTool gets errors and warnings for a file
type LSPDiagnosticsTool struct{}

func (t *LSPDiagnosticsTool) Name() string { return "lsp_diagnostics" }

func (t *LSPDiagnosticsTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	if LSPClient == nil {
		return "LSP not available. Please configure LSP servers in ~/.xelyon/config.yaml", nil, nil
	}

	path := args["path"]
	if path == "" {
		return "Error: path is required", nil, fmt.Errorf("path is required")
	}

	absPath, err := common.ValidatePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), LSPToolTimeout)
	defer cancel()

	diagnostics, err := LSPClient.GetDiagnostics(ctx, absPath)
	if err != nil {
		return fmt.Sprintf("LSP diagnostics not available: %v", err), nil, nil
	}

	if len(diagnostics) == 0 {
		green.Println("✅ No errors or warnings found")
		return "No diagnostics found. Code looks good!", nil, nil
	}

	// Categorize diagnostics
	var errors, warnings, infos, hints []lsplib.Diagnostic
	for _, d := range diagnostics {
		switch d.Severity {
		case lsplib.DiagnosticSeverityError:
			errors = append(errors, d)
		case lsplib.DiagnosticSeverityWarning:
			warnings = append(warnings, d)
		case lsplib.DiagnosticSeverityInformation:
			infos = append(infos, d)
		case lsplib.DiagnosticSeverityHint:
			hints = append(hints, d)
		default:
			// Treat unknown severity as warning
			warnings = append(warnings, d)
		}
	}

	// Format output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d errors, %d warnings", len(errors), len(warnings)))
	if len(infos) > 0 || len(hints) > 0 {
		sb.WriteString(fmt.Sprintf(", %d info, %d hints", len(infos), len(hints)))
	}
	sb.WriteString(":\n\n")

	// Print errors first
	for _, d := range errors {
		sb.WriteString(fmt.Sprintf("❌ Error [%d:%d]: %s\n",
			d.Range.Start.Line+1, d.Range.Start.Character+1, d.Message))
	}

	// Then warnings
	for _, d := range warnings {
		sb.WriteString(fmt.Sprintf("⚠️ Warning [%d:%d]: %s\n",
			d.Range.Start.Line+1, d.Range.Start.Character+1, d.Message))
	}

	// Then info
	for _, d := range infos {
		sb.WriteString(fmt.Sprintf("ℹ️ Info [%d:%d]: %s\n",
			d.Range.Start.Line+1, d.Range.Start.Character+1, d.Message))
	}

	// Then hints
	for _, d := range hints {
		sb.WriteString(fmt.Sprintf("💡 Hint [%d:%d]: %s\n",
			d.Range.Start.Line+1, d.Range.Start.Character+1, d.Message))
	}

	// Print summary to console
	if len(errors) > 0 {
		red.Printf("📋 LSP: Found %d errors, %d warnings\n", len(errors), len(warnings))
	} else if len(warnings) > 0 {
		yellow.Printf("📋 LSP: Found %d warnings\n", len(warnings))
	} else {
		cyan.Printf("📋 LSP: Found %d info/hints\n", len(infos)+len(hints))
	}

	return sb.String(), nil, nil
}

// ===== lsp_rename Tool =====

// LSPRenameTool renames a symbol at the given position
type LSPRenameTool struct{}

func (t *LSPRenameTool) Name() string { return "lsp_rename" }

func (t *LSPRenameTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	if LSPClient == nil {
		return "LSP not available. Please configure LSP servers in ~/.xelyon/config.yaml", nil, nil
	}

	path := args["path"]
	if path == "" {
		return "Error: path is required", nil, fmt.Errorf("path is required")
	}

	line, err := strconv.Atoi(args["line"])
	if err != nil || line < 1 {
		return "Error: line must be a positive number (1-indexed)", nil, fmt.Errorf("line must be a positive number")
	}

	character, err := strconv.Atoi(args["character"])
	if err != nil || character < 1 {
		return "Error: character must be a positive number (1-indexed)", nil, fmt.Errorf("character must be a positive number")
	}

	newName := args["new_name"]
	if newName == "" {
		return "Error: new_name is required", nil, fmt.Errorf("new_name is required")
	}

	apply := args["apply"] == "true"

	absPath, err := common.ValidatePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), LSPToolTimeout)
	defer cancel()

	edit, err := LSPClient.Rename(ctx, absPath, line, character, newName)
	if err != nil {
		return fmt.Sprintf("LSP rename not available: %v\nTip: Use str_replace_editor to manually rename.", err), nil, nil
	}

	if edit == nil || len(edit.Changes) == 0 {
		return "No changes needed or rename not supported for this symbol", nil, nil
	}

	// Format output - list all changes
	var sb strings.Builder
	totalEdits := 0
	for _, edits := range edit.Changes {
		totalEdits += len(edits)
	}

	sb.WriteString(fmt.Sprintf("Rename to '%s' will affect %d location(s) in %d file(s):\n\n", newName, totalEdits, len(edit.Changes)))

	for uri, edits := range edit.Changes {
		filePath := lsplib.URIToFile(uri)
		sb.WriteString(fmt.Sprintf("📄 %s:\n", filePath))
		for _, e := range edits {
			sb.WriteString(fmt.Sprintf("   [%d:%d-%d:%d] → \"%s\"\n",
				e.Range.Start.Line+1, e.Range.Start.Character+1,
				e.Range.End.Line+1, e.Range.End.Character+1,
				e.NewText))
		}
		sb.WriteString("\n")
	}

	// Preview mode (default)
	if !apply {
		sb.WriteString("💡 Tip: Use apply=\"true\" to apply changes.")
		green.Printf("🔄 LSP: Rename preview - %d edits in %d files\n", totalEdits, len(edit.Changes))
		return sb.String(), nil, nil
	}

	// Apply mode
	fmt.Print(sb.String())

	// User confirmation
	dec := common.ConfirmWithAutoApproveDecision("lsp_rename", "Apply rename?")
	if dec.Action != common.ConfirmYes {
		if dec.Action == common.ConfirmComment {
			return fmt.Sprintf("Cancelled with comment: %s", dec.Comment), nil, nil
		}
		return "Rename cancelled", nil, nil
	}

	// Apply changes to each file
	filesModified := 0
	for uri, edits := range edit.Changes {
		filePath := lsplib.URIToFile(uri)

		// Create backup
		if _, backupErr := common.CreateBackup(filePath); backupErr != nil {
			common.Yellow.Printf("Warning: Failed to create backup for %s: %v\n", filePath, backupErr)
		}

		// Apply TextEdits
		if applyErr := applyTextEdits(filePath, edits); applyErr != nil {
			return fmt.Sprintf("Error applying edits to %s: %v", filePath, applyErr), nil, fmt.Errorf("failed to apply edits: %w", applyErr)
		}
		filesModified++
	}

	green.Printf("✅ Renamed to '%s' - %d edits in %d files\n", newName, totalEdits, filesModified)
	return fmt.Sprintf("Successfully renamed to '%s' in %d file(s)", newName, filesModified), nil, nil
}

// applyTextEdits applies TextEdits to a file
// Edits are applied in reverse order (bottom-up) to prevent position shifts
func applyTextEdits(filePath string, edits []lsplib.TextEdit) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Sort edits in reverse order (by line descending, then character descending)
	sortedEdits := make([]lsplib.TextEdit, len(edits))
	copy(sortedEdits, edits)
	sort.Slice(sortedEdits, func(i, j int) bool {
		if sortedEdits[i].Range.Start.Line != sortedEdits[j].Range.Start.Line {
			return sortedEdits[i].Range.Start.Line > sortedEdits[j].Range.Start.Line
		}
		return sortedEdits[i].Range.Start.Character > sortedEdits[j].Range.Start.Character
	})

	// Apply each edit
	for _, edit := range sortedEdits {
		startLine := edit.Range.Start.Line
		startChar := edit.Range.Start.Character
		endLine := edit.Range.End.Line
		endChar := edit.Range.End.Character

		// Bounds check
		if startLine < 0 || startLine >= len(lines) || endLine < 0 || endLine >= len(lines) {
			return fmt.Errorf("edit range out of bounds: line %d-%d (file has %d lines)", startLine+1, endLine+1, len(lines))
		}

		// Single line edit
		if startLine == endLine {
			line := lines[startLine]
			if startChar > len(line) {
				startChar = len(line)
			}
			if endChar > len(line) {
				endChar = len(line)
			}
			lines[startLine] = line[:startChar] + edit.NewText + line[endChar:]
		} else {
			// Multi-line edit
			startLineContent := lines[startLine]
			endLineContent := lines[endLine]

			if startChar > len(startLineContent) {
				startChar = len(startLineContent)
			}
			if endChar > len(endLineContent) {
				endChar = len(endLineContent)
			}

			// Combine: start of first line + new text + end of last line
			newContent := startLineContent[:startChar] + edit.NewText + endLineContent[endChar:]

			// Replace the affected lines with new content
			newLines := strings.Split(newContent, "\n")
			lines = append(lines[:startLine], append(newLines, lines[endLine+1:]...)...)
		}
	}

	// Write back
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
