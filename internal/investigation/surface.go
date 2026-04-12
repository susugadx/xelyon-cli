package investigation

// Surface は gather_context-first investigation surface の公開契約を表す。
//
// Current contract:
// - SurfaceDefault: gather_context が default。search_code/read_file/list_dir は hidden。
// - SurfaceEditExactControl: gather_context が default。read_file は apply_patch 向け exact-control override。
// - SurfaceLegacyOverrides: gather_context が default。search_code/read_file は low-level override として visible。
// - list_dir は current agent surfaces では hidden のまま維持し、directory investigation の front door は gather_context に寄せる。
type Surface string

const (
	SurfaceDefault          Surface = "default"
	SurfaceEditExactControl Surface = "edit_exact_control"
	SurfaceLegacyOverrides  Surface = "legacy_overrides"
)

// ToolRole は surface ごとの tool の位置づけを表す。
type ToolRole string

const (
	ToolRoleDefault          ToolRole = "default"
	ToolRoleLowLevelOverride ToolRole = "low_level_override"
	ToolRoleEditExactControl ToolRole = "edit_exact_control"
	ToolRoleHidden           ToolRole = "hidden"
)

const (
	ToolGatherContext = "gather_context"
	ToolSearchCode    = "search_code"
	ToolReadFile      = "read_file"
	ToolListDir       = "list_dir"
)

var surfaceToolRoles = map[Surface]map[string]ToolRole{
	SurfaceDefault: {
		ToolGatherContext: ToolRoleDefault,
		ToolSearchCode:    ToolRoleHidden,
		ToolReadFile:      ToolRoleHidden,
		ToolListDir:       ToolRoleHidden,
	},
	SurfaceEditExactControl: {
		ToolGatherContext: ToolRoleDefault,
		ToolSearchCode:    ToolRoleHidden,
		ToolReadFile:      ToolRoleEditExactControl,
		ToolListDir:       ToolRoleHidden,
	},
	SurfaceLegacyOverrides: {
		ToolGatherContext: ToolRoleDefault,
		ToolSearchCode:    ToolRoleLowLevelOverride,
		ToolReadFile:      ToolRoleLowLevelOverride,
		ToolListDir:       ToolRoleHidden,
	},
}

// ResolveSurface returns the investigation surface contract for the current visibility policy.
func ResolveSurface(allowLowLevelOverrides bool, allowReadFileExactControl bool) Surface {
	switch {
	case allowLowLevelOverrides:
		return SurfaceLegacyOverrides
	case allowReadFileExactControl:
		return SurfaceEditExactControl
	default:
		return SurfaceDefault
	}
}

// NormalizeSurface は zero value を default surface に揃える。
func NormalizeSurface(surface Surface) Surface {
	if _, ok := surfaceToolRoles[surface]; ok {
		return surface
	}
	return SurfaceDefault
}

func (s Surface) ToolRole(toolName string) ToolRole {
	roles := surfaceToolRoles[NormalizeSurface(s)]
	if role, ok := roles[toolName]; ok {
		return role
	}
	return ToolRoleHidden
}

func (s Surface) SearchCodeRole() ToolRole {
	return s.ToolRole(ToolSearchCode)
}

func (s Surface) ReadFileRole() ToolRole {
	return s.ToolRole(ToolReadFile)
}

func (s Surface) ListDirRole() ToolRole {
	return s.ToolRole(ToolListDir)
}

func (s Surface) AllowsLowLevelOverrides() bool {
	return s.SearchCodeRole() == ToolRoleLowLevelOverride
}

func (s Surface) HasVisibleReadFile() bool {
	return s.ReadFileRole() != ToolRoleHidden
}

func (s Surface) HasReadFileExactControl() bool {
	return s.ReadFileRole() == ToolRoleEditExactControl
}

// Summary returns a short human-readable summary of the current investigation surface.
func (s Surface) Summary() string {
	switch NormalizeSurface(s) {
	case SurfaceLegacyOverrides:
		return "gather_context default + search_code/read_file low-level overrides"
	case SurfaceEditExactControl:
		return "gather_context default + read_file exact-control override"
	default:
		return "gather_context default"
	}
}

// HelpSummary returns the shared short /help wording for core investigation tools.
// The investigation package owns these summaries so prompt/help/visibility contract wording
// for gather_context, search_code, read_file, and list_dir does not drift across surfaces.
func (s Surface) HelpSummary(toolName string) (string, bool) {
	switch toolName {
	case ToolGatherContext:
		return "Primary/default investigation tool", true
	case ToolSearchCode:
		return "Low-level code search override on legacy surfaces when exposed", true
	case ToolReadFile:
		if s.HasReadFileExactControl() {
			return "Exact file reader for edit/apply_patch exact-control override", true
		}
		return "Low-level exact file reader override when exposed", true
	case ToolListDir:
		return "Low-level directory listing; current gather_context-first surfaces keep it hidden", true
	default:
		return "", false
	}
}

func (r ToolRole) Visible() bool {
	return r != ToolRoleHidden
}
