package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func buildRootsCacheKey(roots []discoverRoot) string {
	if len(roots) == 0 {
		return "(no-roots)"
	}
	keys := make([]string, 0, len(roots))
	for _, root := range roots {
		keys = append(keys, cleanAbsPathOrFallback(root.Path))
	}
	return strings.Join(keys, "\x00")
}

func resolveDiscoverRootsFromOptions(opts DiscoverOptions) []discoverRoot {
	ctx := resolveDiscoverContext(opts)
	return resolveDiscoverRoots(ctx.cwd, ctx.home)
}

func buildRootsStateFingerprint(roots []discoverRoot) string {
	hasher := sha256.New()
	for _, root := range roots {
		path := cleanAbsPathOrFallback(root.Path)
		_, _ = hasher.Write([]byte("root:" + path))
		info, err := os.Stat(path)
		if err != nil {
			_, _ = hasher.Write([]byte("|err=" + err.Error() + "\n"))
			continue
		}
		_, _ = fmt.Fprintf(hasher, "|mtime=%d|size=%d\n", info.ModTime().UnixNano(), info.Size())

		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			_, _ = hasher.Write([]byte("|entries_err=" + readErr.Error() + "\n"))
			continue
		}

		childDirs := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				childDirs = append(childDirs, entry.Name())
			}
		}
		sort.Strings(childDirs)
		_, _ = hasher.Write([]byte("|child_dirs=" + strings.Join(childDirs, ",") + "\n"))
		for _, child := range childDirs {
			childPath := filepath.Join(path, child)
			childInfo, childErr := os.Stat(childPath)
			if childErr != nil {
				_, _ = hasher.Write([]byte("|child:" + child + "|err=" + childErr.Error() + "\n"))
				continue
			}
			_, _ = fmt.Fprintf(hasher, "|child:%s|mtime=%d|size=%d\n", child, childInfo.ModTime().UnixNano(), childInfo.Size())
		}
	}
	return hex.EncodeToString(hasher.Sum(nil))
}
