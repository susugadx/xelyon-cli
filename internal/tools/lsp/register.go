package lsp

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// RegisterTools registers all LSP tools to the registry
func RegisterTools(registry *tools.Registry) {
	registry.Register(&LSPFindTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
