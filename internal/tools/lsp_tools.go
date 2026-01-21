package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

// LSPClient is set by agent during initialization
var LSPClient *lsp.Client

// LSPToolTimeout is the timeout for LSP operations
const LSPToolTimeout = 30 * time.Second

// ===== lsp_references Tool =====

// LSPReferencesTool finds all references to a symbol
type LSPReferencesTool struct{}

func (t *LSPReferencesTool) Name() string { return "lsp_references" }

func (t *LSPReferencesTool) Run(args map[string]string) (string, *FileChange, error) {
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
	absPath, err := ValidatePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}

	// Include declaration by default
	includeDecl := true
	if args["include_declaration"] == "false" {
		includeDecl = false
	}

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
		filePath := lsp.URIToFile(loc.URI)
		sb.WriteString(fmt.Sprintf("  %s:%d:%d\n", filePath, loc.Range.Start.Line+1, loc.Range.Start.Character+1))
	}

	green.Printf("🔍 LSP: Found %d references\n", len(locations))
	return sb.String(), nil, nil
}

// ===== lsp_definition Tool =====

// LSPDefinitionTool finds the definition of a symbol
type LSPDefinitionTool struct{}

func (t *LSPDefinitionTool) Name() string { return "lsp_definition" }

func (t *LSPDefinitionTool) Run(args map[string]string) (string, *FileChange, error) {
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

	absPath, err := ValidatePath(path)
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
		filePath := lsp.URIToFile(loc.URI)
		sb.WriteString(fmt.Sprintf("  %s:%d:%d\n", filePath, loc.Range.Start.Line+1, loc.Range.Start.Character+1))
	}

	green.Printf("🔍 LSP: Found definition\n")
	return sb.String(), nil, nil
}

// ===== lsp_hover Tool =====

// LSPHoverTool gets type information and documentation for a symbol
type LSPHoverTool struct{}

func (t *LSPHoverTool) Name() string { return "lsp_hover" }

func (t *LSPHoverTool) Run(args map[string]string) (string, *FileChange, error) {
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

	absPath, err := ValidatePath(path)
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

// ===== Registration =====

// RegisterLSPTools registers all LSP tools to the registry
func RegisterLSPTools(r *Registry) {
	r.Register(&LSPReferencesTool{})
	r.Register(&LSPDefinitionTool{})
	r.Register(&LSPHoverTool{})
}
