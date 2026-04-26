package file

// DirectQueryTargetKind describes the direct gather_context routing target.
type DirectQueryTargetKind string

const (
	DirectQueryTargetFile      DirectQueryTargetKind = "file"
	DirectQueryTargetDirectory DirectQueryTargetKind = "directory"
)

type DirectQueryResolutionKind string

const (
	DirectQueryResolutionFiles     DirectQueryResolutionKind = "files"
	DirectQueryResolutionDirectory DirectQueryResolutionKind = "directory"
)

// DirectQueryTarget is a resolved direct file/range/directory target for high-level orchestration.
type DirectQueryTarget struct {
	RawEntry      string
	FilePath      string
	ResolvedPath  string
	AllowedRoots  []string
	WorkspaceRoot string
	FileFilter    string
	BypassIgnores bool
	StartLine     int
	EndLine       int
	Kind          DirectQueryTargetKind
}

type DirectQueryResolution struct {
	Kind    DirectQueryResolutionKind
	Targets []DirectQueryTarget
}

// GatherContextDirectRouteKind is the file-package-owned route kind for gather_context direct flows.
type GatherContextDirectRouteKind string

const (
	GatherContextDirectRouteRead      GatherContextDirectRouteKind = "read"
	GatherContextDirectRouteDirectory GatherContextDirectRouteKind = "directory"
)

type GatherContextDirectRouteOutcomeKind string

const (
	GatherContextDirectRouteOutcomeNone     GatherContextDirectRouteOutcomeKind = "none"
	GatherContextDirectRouteOutcomeResolved GatherContextDirectRouteOutcomeKind = "resolved"
	GatherContextDirectRouteOutcomeError    GatherContextDirectRouteOutcomeKind = "error"
)

// GatherContextDirectRoutePolicy controls gather_context-specific direct routing behavior.
type GatherContextDirectRoutePolicy struct {
	AllowImplicitBareFile bool
	ScopedPath            string
	FileFilter            string
}

// GatherContextDirectRoute is the resolved direct route that gather_context can execute.
type GatherContextDirectRoute struct {
	Kind    GatherContextDirectRouteKind
	targets []DirectQueryTarget
}

// GatherContextDirectRouteOutcome classifies a gather_context query as direct, non-direct, or invalid direct.
type GatherContextDirectRouteOutcome struct {
	Kind  GatherContextDirectRouteOutcomeKind
	Route GatherContextDirectRoute
	Error string
}

// RawEntries returns the normalized direct query entries for debugging and tests.
func (r GatherContextDirectRoute) RawEntries() []string {
	entries := make([]string, 0, len(r.targets))
	for _, target := range r.targets {
		entries = append(entries, target.RawEntry)
	}
	return entries
}

// PrefersFullRead reports whether this direct route is a single whole-file read.
// gather_context can use this as a surface-specific hint without re-parsing query text.
func (r GatherContextDirectRoute) PrefersFullRead() bool {
	if r.Kind != GatherContextDirectRouteRead || len(r.targets) != 1 {
		return false
	}
	target := r.targets[0]
	return target.Kind == DirectQueryTargetFile && target.StartLine == 0 && target.EndLine == 0
}
