package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScriptPathErrorKind は ResolveScriptPath の失敗種別。
type ScriptPathErrorKind string

const (
	ScriptPathErrorRequired       ScriptPathErrorKind = "required"
	ScriptPathErrorAbsolute       ScriptPathErrorKind = "absolute_path_not_allowed"
	ScriptPathErrorInvalid        ScriptPathErrorKind = "invalid_path"
	ScriptPathErrorEscapesScripts ScriptPathErrorKind = "escapes_scripts_dir"
	ScriptPathErrorNotFound       ScriptPathErrorKind = "not_found"
	ScriptPathErrorDirectory      ScriptPathErrorKind = "is_directory"
	ScriptPathErrorResolveSymlink ScriptPathErrorKind = "resolve_symlink_failed"
	ScriptPathErrorSymlinkEscapes ScriptPathErrorKind = "symlink_escapes_scripts_dir"
)

// ScriptPathError は script path 解決失敗の typed error。
type ScriptPathError struct {
	Kind ScriptPathErrorKind
	Path string
	Err  error
}

func (e *ScriptPathError) Error() string {
	if e == nil {
		return ""
	}

	switch e.Kind {
	case ScriptPathErrorRequired:
		return "script path is required"
	case ScriptPathErrorAbsolute:
		return "absolute script path is not allowed"
	case ScriptPathErrorInvalid:
		return "script path is invalid"
	case ScriptPathErrorEscapesScripts:
		return "script path escapes scripts directory"
	case ScriptPathErrorNotFound:
		return wrapScriptPathError("script not found", e.Err)
	case ScriptPathErrorDirectory:
		return "script path points to a directory"
	case ScriptPathErrorResolveSymlink:
		return wrapScriptPathError("failed to resolve script symlink", e.Err)
	case ScriptPathErrorSymlinkEscapes:
		return "script symlink escapes skill scripts directory"
	default:
		if e.Err != nil {
			return e.Err.Error()
		}
		return "script path resolution failed"
	}
}

func (e *ScriptPathError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrapScriptPathError(prefix string, err error) string {
	if err == nil {
		return prefix
	}
	return fmt.Sprintf("%s: %v", prefix, err)
}

func newScriptPathError(kind ScriptPathErrorKind, path string, err error) error {
	return &ScriptPathError{
		Kind: kind,
		Path: path,
		Err:  err,
	}
}

// ResolveScriptPath は skill scripts 配下の実行対象を安全に解決する。
// path traversal と scripts root 外 symlink を拒否する。
func ResolveScriptPath(skill ParsedSkill, scriptPath string) (string, error) {
	relativePath, err := normalizeScriptRelativePath(scriptPath)
	if err != nil {
		return "", err
	}

	scriptRoot := resolveSkillScriptsRoot(skill)
	candidate, err := buildScriptCandidatePath(scriptRoot, relativePath)
	if err != nil {
		return "", err
	}

	if err := assertScriptCandidateFile(candidate); err != nil {
		return "", err
	}

	return resolveScriptRealPath(candidate, scriptRoot)
}

func normalizeScriptRelativePath(scriptPath string) (string, error) {
	scriptPath = strings.TrimSpace(scriptPath)
	if scriptPath == "" {
		return "", newScriptPathError(ScriptPathErrorRequired, scriptPath, nil)
	}
	if filepath.IsAbs(scriptPath) {
		return "", newScriptPathError(ScriptPathErrorAbsolute, scriptPath, nil)
	}

	cleanScript := filepath.Clean(scriptPath)
	if cleanScript == "." || cleanScript == string(filepath.Separator) {
		return "", newScriptPathError(ScriptPathErrorInvalid, scriptPath, nil)
	}
	return cleanScript, nil
}

func resolveSkillScriptsRoot(skill ParsedSkill) string {
	return filepath.Join(cleanAbsPathOrFallback(skill.Directory), "scripts")
}

func buildScriptCandidatePath(scriptRoot, relativePath string) (string, error) {
	root := cleanAbsPathOrFallback(scriptRoot)
	candidate := cleanAbsPathOrFallback(filepath.Join(root, relativePath))
	if !isSubPath(candidate, root) {
		return "", newScriptPathError(ScriptPathErrorEscapesScripts, relativePath, nil)
	}
	return candidate, nil
}

func assertScriptCandidateFile(candidate string) error {
	info, err := os.Lstat(candidate)
	if err != nil {
		return newScriptPathError(ScriptPathErrorNotFound, candidate, err)
	}
	if info.IsDir() {
		return newScriptPathError(ScriptPathErrorDirectory, candidate, nil)
	}
	return nil
}

func resolveScriptRealPath(candidate, scriptRoot string) (string, error) {
	realRoot, err := filepath.EvalSymlinks(scriptRoot)
	if err != nil {
		realRoot = cleanAbsPathOrFallback(scriptRoot)
	}
	realRoot = cleanAbsPathOrFallback(realRoot)

	realPath, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", newScriptPathError(ScriptPathErrorResolveSymlink, candidate, err)
	}
	realPath = cleanAbsPathOrFallback(realPath)
	if !isSubPath(realPath, realRoot) {
		return "", newScriptPathError(ScriptPathErrorSymlinkEscapes, candidate, nil)
	}
	return realPath, nil
}

func isSubPath(path, base string) bool {
	if path == base {
		return true
	}
	return strings.HasPrefix(path, base+string(filepath.Separator))
}
