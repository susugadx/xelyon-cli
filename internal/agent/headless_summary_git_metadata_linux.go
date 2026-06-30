//go:build linux

package agent

import (
	"fmt"
	"os"
	"syscall"
)

func headlessGitChangedFileMetadata(info os.FileInfo) string {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return ""
	}
	return fmt.Sprintf(
		"dev:%d ino:%d ctime:%d.%d uid:%d gid:%d",
		uint64(stat.Dev),
		uint64(stat.Ino),
		stat.Ctim.Sec,
		stat.Ctim.Nsec,
		stat.Uid,
		stat.Gid,
	)
}
