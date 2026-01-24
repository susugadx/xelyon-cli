package lsp

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// RegisterTools registers all LSP tools to the registry
func RegisterTools(registry *tools.Registry) {
	registry.Register(&LSPReferencesTool{})
	registry.Register(&LSPDefinitionTool{})
	registry.Register(&LSPHoverTool{})
	registry.Register(&LSPDiagnosticsTool{})
	registry.Register(&LSPRenameTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
