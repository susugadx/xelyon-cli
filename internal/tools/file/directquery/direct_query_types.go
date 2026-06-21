package directquery

// directQueryTargetKind describes the direct gather_context routing target.
type directQueryTargetKind string

const (
	directQueryTargetFile      directQueryTargetKind = "file"
	directQueryTargetDirectory directQueryTargetKind = "directory"
)

type directQueryResolutionKind string

const (
	directQueryResolutionFiles     directQueryResolutionKind = "files"
	directQueryResolutionDirectory directQueryResolutionKind = "directory"
)

// directQueryTarget is a resolved direct file/range/directory target for high-level orchestration.
type directQueryTarget struct {
	RawEntry      string
	FilePath      string
	ResolvedPath  string
	AllowedRoots  []string
	WorkspaceRoot string
	FileFilter    string
	BypassIgnores bool
	StartLine     int
	EndLine       int
	Kind          directQueryTargetKind
}

type directQueryResolution struct {
	Kind    directQueryResolutionKind
	Targets []directQueryTarget
}

// RouteKind is the file-package-owned route kind for gather_context direct flows.
type RouteKind string

const (
	RouteRead      RouteKind = "read"
	RouteDirectory RouteKind = "directory"
)

type OutcomeKind string

const (
	OutcomeNone     OutcomeKind = "none"
	OutcomeResolved OutcomeKind = "resolved"
	OutcomeError    OutcomeKind = "error"
)

// Policy controls gather_context-specific direct routing behavior.
type Policy struct {
	AllowImplicitBareFile bool
	ScopedPath            string
	FileFilter            string
}

// Route is the resolved direct route that gather_context can execute.
type Route struct {
	Kind    RouteKind
	targets []directQueryTarget
}

// Outcome classifies a gather_context query as direct, non-direct, or invalid direct.
type Outcome struct {
	Kind  OutcomeKind
	Route Route
	Error string
}

// RawEntries returns the normalized direct query entries for debugging and tests.
func (r Route) RawEntries() []string {
	entries := make([]string, 0, len(r.targets))
	for _, target := range r.targets {
		entries = append(entries, target.RawEntry)
	}
	return entries
}

// PrefersFullRead reports whether this direct route is a single whole-file read.
// gather_context can use this as a surface-specific hint without re-parsing query text.
func (r Route) PrefersFullRead() bool {
	if r.Kind != RouteRead || len(r.targets) != 1 {
		return false
	}
	target := r.targets[0]
	return target.Kind == directQueryTargetFile && target.StartLine == 0 && target.EndLine == 0
}
