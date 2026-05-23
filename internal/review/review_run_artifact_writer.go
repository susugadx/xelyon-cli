package review

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

// WriteReviewRunArtifact は artifact をメモリに保持する。
func (w *BufferedReviewRunArtifactWriter) WriteReviewRunArtifact(name string, content []byte) error {
	if w == nil {
		return fmt.Errorf("artifact writer is nil")
	}
	if err := validateReviewRunArtifactName(name); err != nil {
		return err
	}
	copied := append([]byte(nil), content...)
	w.artifacts = append(w.artifacts, bufferedReviewRunArtifact{
		name:    name,
		content: copied,
	})
	return nil
}

// Len は保持中の artifact 数を返す。
func (w *BufferedReviewRunArtifactWriter) Len() int {
	if w == nil {
		return 0
	}
	return len(w.artifacts)
}

// FlushTo は保持中の artifact を指定 writer へ登録順で書き出す。
func (w *BufferedReviewRunArtifactWriter) FlushTo(dst ReviewRunArtifactWriter) error {
	if w == nil {
		return fmt.Errorf("artifact writer is nil")
	}
	if dst == nil {
		return fmt.Errorf("artifact destination writer is nil")
	}
	for _, artifact := range w.artifacts {
		if err := dst.WriteReviewRunArtifact(artifact.name, artifact.content); err != nil {
			return err
		}
	}
	return nil
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

func (w *DirectoryReviewRunArtifactWriter) validateDirectory() error {
	if w.repoRoot != "" {
		return validateReviewRunArtifactRepoDirectory(w.repoRoot, w.dir)
	}
	info, err := os.Lstat(w.dir)
	if err != nil {
		return err
	}
	return validateReviewRunArtifactDirectoryInfo(w.dir, info)
}

func normalizeReviewRunArtifactDirectory(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", fmt.Errorf("artifact directory is empty")
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve artifact directory %q: %w", dir, err)
	}
	return filepath.Clean(absDir), nil
}

func normalizeReviewRunArtifactRepoRoot(repoRoot string) (string, error) {
	root, err := normalizeReviewRunArtifactDirectory(repoRoot)
	if err != nil {
		return "", fmt.Errorf("artifact repo root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("artifact repo root %q: %w", root, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("artifact repo root must be a directory: %q", root)
	}
	return root, nil
}

func createReviewRunArtifactRepoDirectory(repoRoot, dir string) error {
	if err := validateReviewRunArtifactPathInsideRepo(repoRoot, dir); err != nil {
		return err
	}
	components, err := reviewRunArtifactRelativeComponents(repoRoot, dir)
	if err != nil {
		return err
	}
	current := repoRoot
	for _, component := range components {
		current = filepath.Join(current, component)
		if err := ensureReviewRunArtifactDirectoryComponent(current); err != nil {
			return err
		}
	}
	return nil
}

func reviewRunArtifactRelativeComponents(repoRoot, dir string) ([]string, error) {
	rel, err := filepath.Rel(repoRoot, dir)
	if err != nil {
		return nil, fmt.Errorf("resolve artifact directory relative path: %w", err)
	}
	if isReviewRunArtifactOutsideRelativePath(rel) {
		return nil, fmt.Errorf("artifact directory must stay under repo root: %q", dir)
	}
	components := splitReviewRunArtifactPath(rel)
	if len(components) == 0 {
		return nil, fmt.Errorf("artifact directory must not be repo root")
	}
	for _, component := range components {
		if err := validateReviewRunArtifactPathComponent("artifact directory component", component); err != nil {
			return nil, err
		}
	}
	return components, nil
}

func splitReviewRunArtifactPath(path string) []string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	components := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		components = append(components, part)
	}
	return components
}

func ensureReviewRunArtifactDirectoryComponent(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if err := os.Mkdir(path, 0o700); err != nil && !os.IsExist(err) {
			return err
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return err
	}
	return validateReviewRunArtifactDirectoryInfo(path, info)
}

func validateReviewRunArtifactDirectoryInfo(path string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("artifact directory component must not be a symlink: %q", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("artifact directory component must be a directory: %q", path)
	}
	return nil
}

func validateReviewRunArtifactRepoDirectory(repoRoot, dir string) error {
	if err := validateReviewRunArtifactPathInsideRepo(repoRoot, dir); err != nil {
		return err
	}
	components, err := reviewRunArtifactRelativeComponents(repoRoot, dir)
	if err != nil {
		return err
	}
	current := repoRoot
	for _, component := range components {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if err := validateReviewRunArtifactDirectoryInfo(current, info); err != nil {
			return err
		}
	}
	return validateReviewRunArtifactEvaluatedPathInsideRepo(repoRoot, dir)
}

func validateReviewRunArtifactPathInsideRepo(repoRoot, candidate string) error {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve artifact repo root: %w", err)
	}
	path, err := filepath.Abs(candidate)
	if err != nil {
		return fmt.Errorf("resolve artifact directory: %w", err)
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("resolve artifact directory relative path: %w", err)
	}
	if isReviewRunArtifactOutsideRelativePath(rel) {
		return fmt.Errorf("artifact directory must stay under repo root: %q", candidate)
	}
	return nil
}

func validateReviewRunArtifactEvaluatedPathInsideRepo(repoRoot, candidate string) error {
	root, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve artifact repo root symlinks: %w", err)
	}
	path, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return fmt.Errorf("resolve artifact directory symlinks: %w", err)
	}
	return validateReviewRunArtifactPathInsideRepo(root, path)
}

func isReviewRunArtifactOutsideRelativePath(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel)
}

func validateReviewRunArtifactPathComponent(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is empty", field)
	}
	if value == "." || value == ".." {
		return fmt.Errorf("%s must not be %q", field, value)
	}
	if filepath.Base(value) != value || filepath.Clean(value) != value {
		return fmt.Errorf("%s must be a single path component: %q", field, value)
	}
	if strings.ContainsAny(value, `/\`) {
		return fmt.Errorf("%s must not contain path separators: %q", field, value)
	}
	return nil
}

func validateReviewRunArtifactName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("artifact name is empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("artifact name must not be %q", name)
	}
	if filepath.Base(name) != name || filepath.Clean(name) != name {
		return fmt.Errorf("artifact name must be a file name: %q", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("artifact name must not contain path separators: %q", name)
	}
	return nil
}

func reviewRunArtifactFilename(name string, index int) string {
	if index <= 1 {
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s_%d%s", base, index, ext)
}
