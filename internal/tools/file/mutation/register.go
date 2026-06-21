package mutation

import "github.com/susugadx/xelyon-cli/internal/tools"

// RegisterTools は file mutation tool 群を Registry に登録する。
func RegisterTools(r *tools.Registry) {
	r.Register(&WriteFileTool{})
	r.Register(&StrReplaceTool{})
	r.Register(&DeleteFileTool{})
}
