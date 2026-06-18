package applypatch

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type fileSnapshot struct {
	Existed  bool
	Contents []byte
	Perm     os.FileMode
}

type plannedFileOp struct {
	Kind     string
	Path     string
	Contents []byte
	Perm     os.FileMode
}

func captureSnapshot(snapshots map[string]fileSnapshot, path string) (fileSnapshot, error) {
	if snapshot, ok := snapshots[path]; ok {
		return snapshot, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			snapshot := fileSnapshot{}
			snapshots[path] = snapshot
			return snapshot, nil
		}
		return fileSnapshot{}, err
	}
	if info.IsDir() {
		return fileSnapshot{}, fmt.Errorf("path is a directory: %s", path)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return fileSnapshot{}, err
	}

	snapshot := fileSnapshot{
		Existed:  true,
		Contents: contents,
		Perm:     info.Mode().Perm(),
	}
	snapshots[path] = snapshot
	return snapshot, nil
}

func applyFileOps(ops []plannedFileOp, snapshots map[string]fileSnapshot) error {
	for _, op := range ops {
		var err error
		switch op.Kind {
		case "write":
			err = writeFileWithPerm(op.Path, op.Contents, op.Perm)
		case "delete":
			err = os.Remove(op.Path)
		default:
			err = fmt.Errorf("unsupported file operation: %s", op.Kind)
		}
		if err == nil {
			continue
		}
		if rollbackErr := rollbackSnapshots(snapshots); rollbackErr != nil {
			return fmt.Errorf("%w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}
	return nil
}

func writeFileWithPerm(path string, contents []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if perm == 0 {
		perm = 0o644
	}

	// 同じディレクトリにtempファイルを作る（同一FS上でrenameがアトミックになる）
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".xelyon-patch-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	success := false
	defer func() {
		if !success {
			tmp.Close()
			_ = os.Remove(tmpPath) //nolint:errcheck // クリーンアップ失敗は無視
		}
	}()

	if _, err := tmp.Write(contents); err != nil {
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	success = true
	return nil
}

func rollbackSnapshots(snapshots map[string]fileSnapshot) error {
	paths := make([]string, 0, len(snapshots))
	for path := range snapshots {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var errs []string
	for _, path := range paths {
		snapshot := snapshots[path]
		if !snapshot.Existed {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			}
			continue
		}
		if err := writeFileWithPerm(path, snapshot.Contents, snapshot.Perm); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
