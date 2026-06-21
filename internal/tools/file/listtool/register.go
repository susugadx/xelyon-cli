package listtool

import "github.com/susugadx/xelyon-cli/internal/tools"

// RegisterTools は list_dir tool を Registry に登録する。
func RegisterTools(r *tools.Registry) {
	r.Register(&ListDirTool{})
}
