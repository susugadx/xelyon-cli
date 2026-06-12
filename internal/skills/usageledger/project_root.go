package usageledger

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// ProjectRootOptions は usage ledger の repo key 解決入力。
type ProjectRootOptions struct {
	Config             *config.Config
	ProjectMapRootPath string
	InvocationCWD      string
}

// ResolveProjectRoot は usage ledger 用の project root を project-map root / invocation cwd から解決する。
func ResolveProjectRoot(opts ProjectRootOptions) (string, bool) {
	if root := cleanProjectRoot(opts.ProjectMapRootPath); strings.TrimSpace(root) != "" {
		return root, true
	}
	root, ok := config.ResolveProjectInstructionProjectRootForDir(opts.Config, opts.InvocationCWD)
	if !ok {
		return "", false
	}
	root = cleanProjectRoot(root)
	if strings.TrimSpace(root) == "" {
		return "", false
	}
	return root, true
}
