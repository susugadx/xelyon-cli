package skills

import "github.com/susugadx/xelyon-cli/internal/tools"

// RegisterTools は skills ツール群を登録する。
func RegisterTools(registry *tools.Registry) {
	registry.Register(&ActivateSkillTool{})
	registry.Register(&RunSkillScriptTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
