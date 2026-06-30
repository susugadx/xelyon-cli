//go:build !linux

package agent

import "os"

func headlessGitChangedFileMetadata(os.FileInfo) string {
	return ""
}
