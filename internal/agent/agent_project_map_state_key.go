package agent

import (
	"fmt"
	"os"
)

func resolveProjectMapStateKeyFromGit(rootPath string) string {
	head := gitProjectMapHEAD(rootPath)
	status := gitProjectMapStatusDigest(rootPath)
	if head == "" && status == "" {
		return ""
	}
	return head + ":" + status
}

func resolveProjectMapStateKeyFromWatch(agent *Agent, rootPath string) string {
	if digest := nonGitProjectMapWatchDigest(rootPath, projectMapWatchDirs(agent), projectMapIgnorePatterns(agent)); digest != "" {
		return "dirs:" + digest
	}
	return ""
}

func resolveProjectMapStateKeyFromRootStat(rootPath string) string {
	info, err := os.Stat(rootPath)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("fs:%d", info.ModTime().UTC().UnixNano())
}
