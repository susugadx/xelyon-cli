package agent

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// isNegativeResult はエラーや空結果かどうかを判定する
func isNegativeResult(content string) bool {
	trimmed := strings.TrimSpace(content)
	return trimmed == "No matches found" ||
		strings.HasPrefix(trimmed, "Error:") ||
		strings.HasPrefix(trimmed, "Error reading file:")
}

// negativeCacheKey はツール名と引数からキャッシュキーを生成する
func negativeCacheKey(toolName string, args map[string]interface{}) string {
	// ツール名 + 引数の文字列化（ソート済み）
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(toolName)
	for _, k := range keys {
		b.WriteString("::")
		b.WriteString(k)
		b.WriteString("=")
		fmt.Fprintf(&b, "%v", args[k])
	}
	return b.String()
}

// CheckNegativeCache はネガティブキャッシュを確認する。
// ヒット時は (前回の結果, true) を返し、ログ表示する。ブロックはしない。
func (c *ToolCache) CheckNegativeCache(toolName string, args map[string]interface{}) (string, bool) {
	key := negativeCacheKey(toolName, args)

	c.mu.RLock()
	entry, exists := c.negatives[key]
	c.mu.RUnlock()

	if !exists {
		return "", false
	}

	// TTL チェック
	if time.Since(entry.CachedAt) > NegativeCacheTTL {
		c.mu.Lock()
		delete(c.negatives, key)
		c.mu.Unlock()
		return "", false
	}

	return entry.Result, true
}

// SetNegativeCache はエラー/空結果をネガティブキャッシュに記録する。
// isNegativeResult が true の結果のみキャッシュする。
func (c *ToolCache) SetNegativeCache(toolName string, args map[string]interface{}, content string) {
	if !isNegativeResult(content) {
		return
	}

	// 1行目だけ保存
	result := strings.TrimSpace(content)
	if idx := strings.IndexByte(result, '\n'); idx >= 0 {
		result = result[:idx]
	}

	key := negativeCacheKey(toolName, args)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.negatives[key] = negativeCacheEntry{
		ToolName: toolName,
		Result:   result,
		CachedAt: time.Now(),
	}
}
