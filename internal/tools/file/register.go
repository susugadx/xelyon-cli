package file

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/file/listtool"
	"github.com/susugadx/xelyon-cli/internal/tools/file/mutation"
	"github.com/susugadx/xelyon-cli/internal/tools/file/readtool"
)

// RegisterTools は file tool owner package 群を Registry に登録する。
func RegisterTools(r *tools.Registry) {
	readtool.RegisterTools(r)
	mutation.RegisterTools(r)
	listtool.RegisterTools(r)
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
