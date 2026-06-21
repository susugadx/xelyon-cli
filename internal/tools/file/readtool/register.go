package readtool

import "github.com/susugadx/xelyon-cli/internal/tools"

// RegisterTools は read_file tool を Registry に登録する。
func RegisterTools(r *tools.Registry) {
	r.Register(&ReadFileTool{})
}
