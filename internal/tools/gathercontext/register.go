package gathercontext

import "github.com/susugadx/xelyon-cli/internal/tools"

// RegisterTools は gather_context ツールを registry に登録する。
func RegisterTools(registry *tools.Registry) {
	registry.Register(&Tool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
