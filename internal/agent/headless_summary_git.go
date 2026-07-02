package agent

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

const headlessGitChangedFileHashLimitBytes int64 = 8 * 1024 * 1024

type headlessGitChangedFilesBaseline struct {
	root  string
	files map[string]headlessGitChangedFileState
	paths []string
	ok    bool
}

type headlessGitChangedFileSnapshot struct {
	path  string
	state headlessGitChangedFileState
}

type headlessGitChangedFileState struct {
	status     string
	exists     bool
	isDir      bool
	mode       os.FileMode
	size       int64
	mtime      int64
	metadata   string
	linkTarget string
	hash       string
}

type headlessGitStatusRecord struct {
	status string
	path   string
}

func newHeadlessGitChangedFilesBaseline(cwd string, readOnly bool) headlessGitChangedFilesBaseline {
	if readOnly {
		return headlessGitChangedFilesBaseline{}
	}
	root, snapshots, ok := headlessGitChangedFileSnapshots(cwd)
	if !ok {
		return headlessGitChangedFilesBaseline{}
	}
	files := make(map[string]headlessGitChangedFileState, len(snapshots))
	paths := make([]string, 0, len(snapshots))
	for _, snapshot := range snapshots {
		files[snapshot.path] = snapshot.state
		paths = append(paths, snapshot.path)
	}
	return headlessGitChangedFilesBaseline{
		root:  root,
		files: files,
		paths: paths,
		ok:    true,
	}
}

func headlessGitChangedFilesSinceBaseline(baseline headlessGitChangedFilesBaseline) ([]string, bool) {
	if !baseline.ok {
		return nil, false
	}
	_, snapshots, ok := headlessGitChangedFileSnapshots(baseline.root)
	if !ok {
		return nil, false
	}
	paths := make([]string, 0, len(snapshots))
	seen := make(map[string]struct{}, len(snapshots)+len(baseline.paths))
	current := make(map[string]headlessGitChangedFileState, len(snapshots))
	appendChanged := func(path string) {
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	for _, snapshot := range snapshots {
		current[snapshot.path] = snapshot.state
		before, existed := baseline.files[snapshot.path]
		if existed && before == snapshot.state {
			continue
		}
		appendChanged(snapshot.path)
	}
	for _, path := range baseline.paths {
		if _, ok := current[path]; ok {
			continue
		}
		appendChanged(path)
	}
	return paths, true
}

func headlessGitChangedFileSnapshots(cwd string) (string, []headlessGitChangedFileSnapshot, bool) {
	root, ok := headlessGitRoot(cwd)
	if !ok {
		return "", nil, false
	}
	output, ok := headlessGitStatusPorcelainZ(root)
	if !ok {
		return "", nil, false
	}
	records := parseGitStatusPorcelainZRecords(output)
	snapshots := make([]headlessGitChangedFileSnapshot, 0, len(records))
	for _, record := range records {
		path, ok := taskstate.NormalizeRepoRelativePath(record.path)
		if !ok {
			continue
		}
		snapshots = append(snapshots, headlessGitChangedFileSnapshot{
			path:  path,
			state: headlessGitChangedFileStateForPath(root, path, record.status),
		})
	}
	return root, snapshots, true
}

func headlessGitRoot(cwd string) (string, bool) {
	args := []string{"rev-parse", "--show-toplevel"}
	if strings.TrimSpace(cwd) != "" {
		args = append([]string{"-C", cwd}, args...)
	}
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", false
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", false
	}
	return root, true
}

func headlessGitStatusPorcelainZ(root string) ([]byte, bool) {
	args := []string{"status", "--porcelain", "-z", "--untracked-files=all"}
	if strings.TrimSpace(root) != "" {
		args = append([]string{"-C", root}, args...)
	}
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, false
	}
	return output, true
}

func headlessGitChangedFileStateForPath(root, path, status string) headlessGitChangedFileState {
	state := headlessGitChangedFileState{status: status}
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	info, err := os.Lstat(fullPath)
	if err != nil {
		return state
	}
	state.exists = true
	state.isDir = info.IsDir()
	state.mode = info.Mode()
	state.size = info.Size()
	state.mtime = info.ModTime().UnixNano()
	state.metadata = headlessGitChangedFileMetadata(info)
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(fullPath)
		if err != nil {
			state.linkTarget = fmt.Sprintf("readlink_error:%T", err)
		} else {
			state.linkTarget = target
		}
	}
	if !info.Mode().IsRegular() {
		return state
	}

	if info.Size() > headlessGitChangedFileHashLimitBytes {
		state.hash = fmt.Sprintf("too_large:%d", info.Size())
		return state
	}

	file, err := os.Open(fullPath)
	if err != nil {
		state.hash = fmt.Sprintf("read_error:%T", err)
		return state
	}
	defer file.Close()

	hasher := sha256.New()
	n, err := io.Copy(hasher, io.LimitReader(file, headlessGitChangedFileHashLimitBytes+1))
	if err != nil {
		state.hash = fmt.Sprintf("read_error:%T", err)
		return state
	}
	if n > headlessGitChangedFileHashLimitBytes {
		state.hash = fmt.Sprintf("too_large:%d", info.Size())
		return state
	}
	state.hash = fmt.Sprintf("%x", hasher.Sum(nil))
	return state
}

func parseGitStatusPorcelainZRecords(output []byte) []headlessGitStatusRecord {
	if len(output) == 0 {
		return nil
	}
	records := strings.Split(string(output), "\x00")
	out := make([]headlessGitStatusRecord, 0, len(records))
	for i := 0; i < len(records); i++ {
		record := records[i]
		if len(record) < 4 {
			continue
		}
		status := record[:2]
		path := record[3:]
		if path != "" {
			out = append(out, headlessGitStatusRecord{
				status: status,
				path:   path,
			})
		}
		if status[0] == 'R' || status[0] == 'C' {
			i++
		}
	}
	return out
}
