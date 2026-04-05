package file

import "github.com/susugadx/xelyon-cli/internal/tools"

// RegisterTools は file パッケージのツール群を Registry に登録する。
func RegisterTools(r *tools.Registry) {
	r.Register(&ReadFileTool{})
	r.Register(&WriteFileTool{})
	r.Register(&StrReplaceTool{})
	r.Register(&DeleteFileTool{})
	r.Register(&ListDirTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
