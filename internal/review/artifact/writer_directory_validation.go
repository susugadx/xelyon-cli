package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
