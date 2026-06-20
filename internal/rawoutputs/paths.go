package rawoutputs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	rawOutputManifestRelativePath = "manifests/raw_outputs.jsonl"
	rawOutputIndexRelativePath    = "indexes/raw_outputs.index.json"
	rawOutputTmpRelativeDir       = "objects/tmp"
)

func secureCleanRoot(root string) (string, error) {
	clean := filepath.Clean(root)
	if clean == "." || clean == string(filepath.Separator) || !filepath.IsAbs(clean) {
		return "", reasonError(ReasonPathInvalid, "invalid root %q", root)
	}
	return clean, nil
}

func rejectSymlinkedRootParents(root string) error {
	clean := filepath.Clean(root)
	volume := filepath.VolumeName(clean)
	rest := strings.TrimPrefix(clean, volume)
	separator := string(filepath.Separator)
	current := volume + separator
	if volume == "" {
		current = separator
	}
	if err := checkExistingDirIsNotSymlink(current); err != nil {
		return err
	}
	for _, component := range pathComponents(strings.TrimPrefix(rest, separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return reasonError(ReasonPathInvalid, "stat directory %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return reasonError(ReasonPathInvalid, "directory path contains symlink: %s", current)
		}
		if !info.IsDir() {
			return reasonError(ReasonPathInvalid, "path component is not a directory: %s", current)
		}
	}
	return nil
}

func validateSessionID(sessionID string) error {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return reasonError(ReasonRefInvalid, "session ID is empty")
	}
	if len(trimmed) > 128 {
		return reasonError(ReasonRefInvalid, "session ID is too long")
	}
	if trimmed != sessionID {
		return reasonError(ReasonRefInvalid, "session ID must not include leading or trailing whitespace")
	}
	if sessionID == "." || sessionID == ".." || strings.ContainsAny(sessionID, `/\`) || !safeIDPattern.MatchString(sessionID) {
		return reasonError(ReasonRefInvalid, "session ID has unsafe characters")
	}
	return nil
}

func validateSurface(surface Surface) error {
	switch surface {
	case SurfaceCommandOutput,
		SurfaceMCPToolResult,
		SurfaceXelyonWebSearchToolResult,
		SurfaceProviderNativeBuiltinReplay,
		SurfaceReviewProbeResult:
		return nil
	default:
		return reasonError(ReasonRefInvalid, "invalid surface %q", surface)
	}
}

func validateRefID(refID string) error {
	if !strings.HasPrefix(refID, refIDPrefix) || !safeIDPattern.MatchString(refID) {
		return reasonError(ReasonRefInvalid, "invalid ref ID")
	}
	return nil
}

func validateRef(ref RawOutputRef) error {
	if err := validateSessionID(ref.SessionID); err != nil {
		return err
	}
	if err := validateRefID(ref.RefID); err != nil {
		return err
	}
	if _, err := parseSHA256ID(ref.ArtifactID); err != nil {
		return err
	}
	if _, err := parseSHA256ID(ref.ContentHash); err != nil {
		return err
	}
	if ref.ArtifactID != ref.ContentHash {
		return reasonError(ReasonRefInvalid, "artifact ID and content hash differ")
	}
	if err := validateSurface(Surface(ref.Surface)); err != nil {
		return err
	}
	return nil
}

func parseSHA256ID(value string) (string, error) {
	hash := strings.TrimPrefix(value, "sha256:")
	if len(hash) != 64 {
		return "", reasonError(ReasonRefInvalid, "invalid sha256 length")
	}
	for _, r := range hash {
		if r >= '0' && r <= '9' || r >= 'a' && r <= 'f' {
			continue
		}
		return "", reasonError(ReasonRefInvalid, "invalid sha256 characters")
	}
	return hash, nil
}

func (s *Store) sessionRoot(sessionID string) string {
	return filepath.Join(string(s.root), "sessions", sessionID)
}

func (s *Store) manifestPath(sessionID string) string {
	return filepath.Join(s.sessionRoot(sessionID), filepath.FromSlash(rawOutputManifestRelativePath))
}

func (s *Store) indexPath(sessionID string) string {
	return filepath.Join(s.sessionRoot(sessionID), filepath.FromSlash(rawOutputIndexRelativePath))
}

func (s *Store) safeSessionPath(sessionID, relative string) (string, error) {
	if err := validateSessionID(sessionID); err != nil {
		return "", err
	}
	if filepath.IsAbs(relative) {
		return "", reasonError(ReasonPathInvalid, "relative path is absolute")
	}
	if strings.Contains(relative, `\`) {
		return "", reasonError(ReasonPathInvalid, "relative path contains backslash")
	}
	root := s.sessionRoot(sessionID)
	path := filepath.Clean(filepath.Join(root, filepath.FromSlash(relative)))
	if path != root && !strings.HasPrefix(path, root+string(filepath.Separator)) {
		return "", reasonError(ReasonPathInvalid, "path escapes session root")
	}
	return path, nil
}

func (s *Store) ensureSessionDirs(sessionID string) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}
	for _, dir := range []string{
		"manifests",
		"indexes",
		"objects",
		"objects/sha256",
	} {
		if _, err := s.ensureSafeSessionDir(sessionID, dir); err != nil {
			return reasonError(ReasonPathInvalid, "create %s: %w", dir, err)
		}
	}
	return nil
}

func (s *Store) ensureSafeSessionDir(sessionID, relativeDir string) (string, error) {
	path, err := s.safeSessionPath(sessionID, relativeDir)
	if err != nil {
		return "", err
	}
	if err := s.ensureSafeStoreDir(path); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) ensureSafeSessionFileParent(sessionID, relativeFile string) (string, error) {
	path, err := s.safeSessionPath(sessionID, relativeFile)
	if err != nil {
		return "", err
	}
	if err := s.ensureSafeStoreDir(filepath.Dir(path)); err != nil {
		return "", err
	}
	if err := rejectSymlinkFinalPath(path); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) safeExistingSessionFilePath(sessionID, relativeFile string) (string, error) {
	path, err := s.safeSessionPath(sessionID, relativeFile)
	if err != nil {
		return "", err
	}
	if err := s.rejectSymlinkedStoreParents(filepath.Dir(path)); err != nil {
		return "", err
	}
	if err := rejectSymlinkFinalPath(path); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) ensureSafeStoreDir(dir string) error {
	root, rel, err := s.storeRelativePath(dir)
	if err != nil {
		return err
	}
	if err := ensureExistingDirIsNotSymlink(root); err != nil {
		return err
	}
	current := root
	for _, component := range pathComponents(rel) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if err := os.Mkdir(current, 0o700); err != nil && !os.IsExist(err) {
				return reasonError(ReasonPathInvalid, "create directory %s: %w", current, err)
			}
			info, err = os.Lstat(current)
		}
		if err != nil {
			return reasonError(ReasonPathInvalid, "stat directory %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return reasonError(ReasonPathInvalid, "directory path contains symlink: %s", current)
		}
		if !info.IsDir() {
			return reasonError(ReasonPathInvalid, "path component is not a directory: %s", current)
		}
	}
	return nil
}

func (s *Store) rejectSymlinkedStoreParents(path string) error {
	root, rel, err := s.storeRelativePath(path)
	if err != nil {
		return err
	}
	if err := checkExistingDirIsNotSymlink(root); err != nil {
		return err
	}
	current := root
	for _, component := range pathComponents(rel) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return reasonError(ReasonPathInvalid, "stat directory %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return reasonError(ReasonPathInvalid, "directory path contains symlink: %s", current)
		}
		if !info.IsDir() {
			return reasonError(ReasonPathInvalid, "path component is not a directory: %s", current)
		}
	}
	return nil
}

func (s *Store) storeRelativePath(path string) (string, string, error) {
	root := filepath.Clean(string(s.root))
	cleanPath := filepath.Clean(path)
	if cleanPath != root && !strings.HasPrefix(cleanPath, root+string(filepath.Separator)) {
		return "", "", reasonError(ReasonPathInvalid, "path escapes store root")
	}
	rel, err := filepath.Rel(root, cleanPath)
	if err != nil {
		return "", "", reasonError(ReasonPathInvalid, "rel path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", reasonError(ReasonPathInvalid, "path escapes store root")
	}
	return root, rel, nil
}

func ensureExistingDirIsNotSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return reasonError(ReasonPathInvalid, "stat directory %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return reasonError(ReasonPathInvalid, "directory path contains symlink: %s", path)
	}
	if !info.IsDir() {
		return reasonError(ReasonPathInvalid, "path component is not a directory: %s", path)
	}
	return nil
}

func checkExistingDirIsNotSymlink(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return reasonError(ReasonPathInvalid, "stat directory %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return reasonError(ReasonPathInvalid, "directory path contains symlink: %s", path)
	}
	if !info.IsDir() {
		return reasonError(ReasonPathInvalid, "path component is not a directory: %s", path)
	}
	return nil
}

func rejectSymlinkFinalPath(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return reasonError(ReasonPathInvalid, "stat path %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return reasonError(ReasonPathInvalid, "path is symlink: %s", path)
	}
	return nil
}

func pathComponents(rel string) []string {
	if rel == "" || rel == "." {
		return nil
	}
	return strings.Split(rel, string(filepath.Separator))
}

func (s *Store) sessionDirs() ([]string, error) {
	root := filepath.Join(string(s.root), "sessions")
	if err := s.rejectSymlinkedStoreParents(root); err != nil {
		return nil, err
	}
	if err := rejectSymlinkFinalPath(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, reasonError(ReasonPathInvalid, "read sessions dir: %w", err)
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if err := validateSessionID(name); err != nil {
			return nil, fmt.Errorf("invalid session dir %q: %w", name, err)
		}
		out = append(out, name)
	}
	return out, nil
}
