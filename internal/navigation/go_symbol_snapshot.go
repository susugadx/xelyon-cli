package navigation

import "github.com/susugadx/xelyon-cli/internal/repomap"

// GoSymbolRuntime carries nil-safe runtime state for snapshot-backed Go symbol resolution.
type GoSymbolRuntime struct {
	ProjectMap         *repomap.ProjectMap
	ProjectMapRootPath string
	ProjectMapStateKey string
	InvocationCWD      string
}

type goSymbolSnapshot struct {
	RootPath string
	StateKey string
	ByName   map[string][]goSymbolSnapshotEntry
}

type goSymbolSnapshotEntry struct {
	Name         string
	Kind         string
	File         string
	Line         int
	EndLine      int
	Signature    string
	Exported     bool
	Receiver     string
	ReceiverNorm string
	PackageDir   string
	StableKey    string
	Collision    bool
}
