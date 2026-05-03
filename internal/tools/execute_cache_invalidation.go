package tools

import (
	"path/filepath"
	"strings"
)

// invalidateToolCache はファイル変更系ツール実行後にキャッシュを無効化
func invalidateToolCache(execCtx ExecutionContext, tc *ToolCall, change *FileChange) {
	cache := execCtx.EffectiveToolCache()
	if cache == nil || tc == nil {
		return
	}

	if shouldClearWholeToolCache(tc) {
		cache.Clear()
		return
	}

	switch tc.Tool {
	// ファイル内容を変更するツール
	// write_file/str_replace は実変更結果（change.Details）を優先して無効化する。
	case "write_file", "str_replace":
		invalidatePathAndSearchCacheByChange(cache, execCtx, tc, change, "path")

	// format/lint は FileChange を返さないため従来どおり args で解決する。
	case "format", "lint":
		invalidatePathAndSearchCache(cache, execCtx, tc.Args["path"])

	// ファイル削除は実変更結果を優先し、対象ファイル・親ディレクトリ・検索キャッシュを無効化する。
	case "delete_file":
		for _, absPath := range resolveCacheTargetPathsForTool(execCtx, tc, change, "path") {
			cache.InvalidateFile(absPath)
			cache.InvalidateDir(filepath.Dir(absPath))
			cache.InvalidateSearchCacheForFile(absPath)
		}

	// コピーはコピー先のディレクトリキャッシュ＆検索キャッシュを無効化
	case "copy_file":
		if absPath, ok := resolveCacheTargetPath(execCtx, tc.Args["dest"]); ok {
			cache.InvalidateDir(filepath.Dir(absPath))
			cache.InvalidateSearchCacheForFile(absPath)
		}

	// ディレクトリ作成は検索結果に影響しないため検索キャッシュはクリアしない
	case "create_dir":
		if absPath, ok := resolveCacheTargetPath(execCtx, tc.Args["path"]); ok {
			cache.InvalidateDir(filepath.Dir(absPath))
		}

	}
}

func shouldClearWholeToolCache(tc *ToolCall) bool {
	if tc == nil {
		return false
	}
	switch tc.Tool {
	// 変更対象が限定できないツールは全キャッシュクリア
	case "apply_patch", "git_checkout", "run_skill_script":
		return true
	// bash: read-only コマンドならキャッシュを保持、それ以外は全クリア
	case "bash":
		return !isBashReadOnly(tc.Args["command"])
	default:
		return false
	}
}

func invalidatePathAndSearchCacheByChange(cache ToolCacheInterface, execCtx ExecutionContext, tc *ToolCall, change *FileChange, fallbackArgKey string) {
	if cache == nil {
		return
	}
	for _, absPath := range resolveCacheTargetPathsForTool(execCtx, tc, change, fallbackArgKey) {
		cache.InvalidateFile(absPath)
		cache.InvalidateSearchCacheForFile(absPath)
	}
}

func invalidatePathAndSearchCache(cache ToolCacheInterface, execCtx ExecutionContext, path string) {
	if cache == nil {
		return
	}
	if absPath, ok := resolveCacheTargetPath(execCtx, path); ok {
		cache.InvalidateFile(absPath)
		cache.InvalidateSearchCacheForFile(absPath)
	}
}

func resolveCacheTargetPathsForTool(execCtx ExecutionContext, tc *ToolCall, change *FileChange, fallbackArgKey string) []string {
	resolved := resolveCacheTargetPathsFromChange(change)
	if len(resolved) > 0 {
		return resolved
	}
	if tc == nil || fallbackArgKey == "" {
		return nil
	}
	if absPath, ok := resolveCacheTargetPath(execCtx, tc.Args[fallbackArgKey]); ok {
		return []string{absPath}
	}
	return nil
}

func resolveCacheTargetPathsFromChange(change *FileChange) []string {
	if change == nil {
		return nil
	}

	resolved := make([]string, 0, len(change.Details)+1)
	seen := make(map[string]struct{}, len(change.Details)+1)

	for _, detail := range change.Details {
		resolved = appendUniqueAbsolutePath(resolved, seen, detail.FilePath)
	}
	if len(resolved) > 0 {
		return resolved
	}

	// Details が空の legacy change との互換のため、絶対パス時のみ FilePath を利用する。
	resolved = appendUniqueAbsolutePath(resolved, seen, change.FilePath)
	return resolved
}

func appendUniqueAbsolutePath(paths []string, seen map[string]struct{}, candidate string) []string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" || !filepath.IsAbs(candidate) {
		return paths
	}
	cleaned := filepath.Clean(candidate)
	if _, exists := seen[cleaned]; exists {
		return paths
	}
	seen[cleaned] = struct{}{}
	return append(paths, cleaned)
}

func resolveCacheTargetPath(execCtx ExecutionContext, path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), true
	}
	if cwd := strings.TrimSpace(execCtx.InvocationCWD); cwd != "" {
		return filepath.Clean(filepath.Join(cwd, path)), true
	}
	if absPath, err := filepath.Abs(path); err == nil {
		return filepath.Clean(absPath), true
	}
	return "", false
}
