package artifact

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReviewRunArtifactWriter は /review runner の debug artifact 保存境界を表す。
// nil の場合、runner は artifact を完全に無効化する。
type ReviewRunArtifactWriter interface {
	WriteReviewRunArtifact(name string, content []byte) error
}

// DirectoryReviewRunArtifactWriter は 1 回の /review 実行 artifact を
// 指定 directory へ保存する writer。
type DirectoryReviewRunArtifactWriter struct {
	dir      string
	repoRoot string
}

type bufferedReviewRunArtifact struct {
	name    string
	content []byte
}

// BufferedReviewRunArtifactWriter は artifact を保存先 I/O なしで一時保持する writer。
type BufferedReviewRunArtifactWriter struct {
	artifacts []bufferedReviewRunArtifact
}

// NewBufferedReviewRunArtifactWriter は flush 可能な in-memory artifact writer を返す。
func NewBufferedReviewRunArtifactWriter() *BufferedReviewRunArtifactWriter {
	return &BufferedReviewRunArtifactWriter{}
}

const (
	reviewRunArtifactRootDirName = ".xelyon"
	reviewRunArtifactRunsDirName = "review-runs"
)

// NewReviewRunDirectoryArtifactWriter は保存先 directory を作成して writer を返す。
func NewReviewRunDirectoryArtifactWriter(dir string) (*DirectoryReviewRunArtifactWriter, error) {
	resolvedDir, err := normalizeReviewRunArtifactDirectory(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(resolvedDir, 0o700); err != nil {
		return nil, err
	}
	writer := &DirectoryReviewRunArtifactWriter{dir: resolvedDir}
	if err := writer.validateDirectory(); err != nil {
		return nil, err
	}
	if err := os.Chmod(resolvedDir, 0o700); err != nil {
		return nil, err
	}
	return writer, nil
}

// NewReviewRunRepoArtifactWriter は repo-local な /review artifact 保存先を作成する。
// repo 管理下の artifact directory component が symlink の場合は拒否する。
func NewReviewRunRepoArtifactWriter(repoRoot, runID string) (*DirectoryReviewRunArtifactWriter, error) {
	root, err := normalizeReviewRunArtifactRepoRoot(repoRoot)
	if err != nil {
		return nil, err
	}
	if err := validateReviewRunArtifactPathComponent("artifact run directory", runID); err != nil {
		return nil, err
	}

	dir := filepath.Join(root, reviewRunArtifactRootDirName, reviewRunArtifactRunsDirName, runID)
	if err := createReviewRunArtifactRepoDirectory(root, dir); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, err
	}
	if err := validateReviewRunArtifactRepoDirectory(root, dir); err != nil {
		return nil, err
	}
	return &DirectoryReviewRunArtifactWriter{dir: dir, repoRoot: root}, nil
}

// Dir は writer の保存先 directory を返す。
func (w *DirectoryReviewRunArtifactWriter) Dir() string {
	if w == nil {
		return ""
	}
	return w.dir
}

// WriteReviewRunArtifact は artifact を 0600 で保存する。
// 同名 artifact が既に存在する場合は _2, _3 ... suffix を付けて両方残す。
func (w *DirectoryReviewRunArtifactWriter) WriteReviewRunArtifact(name string, content []byte) error {
	if w == nil {
		return fmt.Errorf("artifact writer is nil")
	}
	if err := validateReviewRunArtifactName(name); err != nil {
		return err
	}
	if err := w.validateDirectory(); err != nil {
		return err
	}

	for index := 1; ; index++ {
		path := filepath.Join(w.dir, reviewRunArtifactFilename(name, index))
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return err
		}
		if _, err := file.Write(content); err != nil {
			_ = file.Close()
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
		return os.Chmod(path, 0o600)
	}
}
