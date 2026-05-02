package review

type worktreeSnapshot struct {
	entries map[string]worktreeSnapshotEntry
}

type worktreeSnapshotEntry struct {
	statusCode  string
	fingerprint string
}
