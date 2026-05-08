package review

import "sort"

func diffWorktreeSnapshots(before, after worktreeSnapshot) []string {
	changed := make(map[string]struct{}, len(before.entries)+len(after.entries))

	for path, afterEntry := range after.entries {
		beforeEntry, ok := before.entries[path]
		if !ok || beforeEntry.statusCode != afterEntry.statusCode || beforeEntry.fingerprint != afterEntry.fingerprint {
			changed[path] = struct{}{}
		}
	}

	for path := range before.entries {
		if _, ok := after.entries[path]; !ok {
			changed[path] = struct{}{}
		}
	}

	paths := make([]string, 0, len(changed))
	for path := range changed {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
