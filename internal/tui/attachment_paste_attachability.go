package tui

import "os"

type droppedPathAttachability int

const (
	droppedPathAttachabilityUnknown droppedPathAttachability = iota
	droppedPathAttachabilityAttachable
	droppedPathAttachabilityNotAttachable
)

// droppedPathsAttachability は parse 済み path 候補の attach 可能性判定を担当する。
func droppedPathsAttachability(paths []string) droppedPathAttachability {
	if len(paths) == 0 {
		return droppedPathAttachabilityUnknown
	}
	for _, path := range paths {
		if !isAttachablePath(path) {
			return droppedPathAttachabilityNotAttachable
		}
	}
	return droppedPathAttachabilityAttachable
}

func isAttachablePath(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
