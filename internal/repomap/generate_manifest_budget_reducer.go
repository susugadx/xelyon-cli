package repomap

const (
	maxManifestTopLevelDirs  = 8
	maxManifestTopLevelFiles = 8
	maxManifestPriorityFiles = 10
)

type manifestSectionLimits struct {
	dirLimit      int
	fileLimit     int
	priorityLimit int
	changeLimit   int
}

type projectMapManifestBudgetReducer struct {
	pm            *ProjectMap
	topDirs       []string
	topFiles      []string
	priorityFiles []string
	limits        manifestSectionLimits
}

func newProjectMapManifestBudgetReducer(pm *ProjectMap, topDirs, topFiles, priorityFiles []string) *projectMapManifestBudgetReducer {
	return &projectMapManifestBudgetReducer{
		pm:            pm,
		topDirs:       topDirs,
		topFiles:      topFiles,
		priorityFiles: priorityFiles,
		limits: manifestSectionLimits{
			dirLimit:      minInt(len(topDirs), maxManifestTopLevelDirs),
			fileLimit:     minInt(len(topFiles), maxManifestTopLevelFiles),
			priorityLimit: minInt(len(priorityFiles), maxManifestPriorityFiles),
			changeLimit:   len(pm.GitStatus),
		},
	}
}

func (r *projectMapManifestBudgetReducer) reduce() string {
	for {
		result := renderManifest(
			r.topDirs,
			r.limits.dirLimit,
			r.topFiles,
			r.limits.fileLimit,
			r.priorityFiles,
			r.limits.priorityLimit,
			r.pm.GitStatus,
			r.limits.changeLimit,
		)
		if result == "" || r.pm.fitsBudget(result) {
			return result
		}
		if !r.shrink() {
			return r.pm.generateManifestFallback(len(r.pm.Files), len(r.pm.GitStatus))
		}
	}
}

func (r *projectMapManifestBudgetReducer) shrink() bool {
	switch {
	case r.limits.priorityLimit > 0:
		r.limits.priorityLimit--
	case r.limits.changeLimit > 0:
		r.limits.changeLimit--
	case r.limits.fileLimit > 0:
		r.limits.fileLimit--
	case r.limits.dirLimit > 0:
		r.limits.dirLimit--
	default:
		return false
	}
	return true
}
