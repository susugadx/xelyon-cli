package probe

import "testing"

func newScratchOnlyProbeRunner(t *testing.T) (string, *ProbeRunner) {
	t.Helper()
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	return repo, NewProbeRunner(repo)
}
