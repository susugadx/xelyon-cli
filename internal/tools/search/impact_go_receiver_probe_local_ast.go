package search

import "github.com/susugadx/xelyon-cli/internal/tools/search/internal/goreceiverlocal"

func qualifiedReceiverRoleFromLocalDir(baseName, dir string) (methodProbeReceiverRole, bool) {
	role, complete := goreceiverlocal.RoleFromDir(baseName, dir)
	switch role {
	case goreceiverlocal.RoleConcrete:
		return methodProbeReceiverRoleConcrete, complete
	case goreceiverlocal.RoleInterface:
		return methodProbeReceiverRoleInterface, complete
	default:
		return methodProbeReceiverRoleUnknown, complete
	}
}

func qualifiedReceiverDirectMethodFromLocalDir(baseName, methodName, dir string) (bool, bool) {
	return goreceiverlocal.HasDirectMethod(baseName, methodName, dir)
}
