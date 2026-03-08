package agent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ToolCache はツール結果のキャッシュ
// read_file, list_dir の結果をキャッシュしてトークン消費を削減
type ToolCache struct {
	files        map[string]cacheEntry
	dirs         map[string]cacheEntry
	searches     map[string]cacheEntry
	resultHashes map[string]resultHashEntry // content hash → 既出情報（履歴重複排除用）
	negatives    map[string]negativeCacheEntry
	mu           sync.RWMutex
}

// cacheEntry はキャッシュエントリ
type cacheEntry struct {
	Content       string
	ModTime       time.Time
	AccessedAt    time.Time
	AffectedFiles []string // 検索キャッシュのみ使用
}

// resultHashEntry は tool result のハッシュ記録
type resultHashEntry struct {
	ToolName   string
	Turn       int
	AccessedAt time.Time
}

// negativeCacheEntry はエラー/空結果のネガティブキャッシュエントリ
type negativeCacheEntry struct {
	ToolName string
	Result   string // 元のエラーメッセージ（1行目）
	CachedAt time.Time
}

// NegativeCacheTTL はネガティブキャッシュの有効期間
const NegativeCacheTTL = 30 * time.Second

// MaxFileCacheSize はキャッシュする最大ファイルサイズ (1MB)
const MaxFileCacheSize = 1024 * 1024

const (
	MaxFileCacheEntries   = 100
	MaxDirCacheEntries    = 50
	MaxSearchCacheEntries = 50
	MaxResultHashEntries  = 200
)

// NewToolCache は新しい ToolCache を作成
func NewToolCache() *ToolCache {
	return &ToolCache{
		files:        make(map[string]cacheEntry),
		dirs:         make(map[string]cacheEntry),
		searches:     make(map[string]cacheEntry),
		resultHashes: make(map[string]resultHashEntry),
		negatives:    make(map[string]negativeCacheEntry),
	}
}

// GetFile はファイル内容のキャッシュを取得
func (c *ToolCache) GetFile(path string) (string, bool) {
	c.mu.RLock()
	entry, exists := c.files[path]
	c.mu.RUnlock()

	if !exists {
		return "", false
	}

	// ファイルの mtime をチェック
	info, err := os.Stat(path)
	if err != nil {
		// ファイルが削除された等
		c.InvalidateFile(path)
		return "", false
	}

	// ファイルが変更されていたらキャッシュ無効
	if info.ModTime().After(entry.ModTime) {
		c.InvalidateFile(path)
		return "", false
	}

	c.mu.Lock()
	entry.AccessedAt = time.Now()
	c.files[path] = entry
	c.mu.Unlock()
	return entry.Content, true
}

// SetFile はファイル内容をキャッシュに保存
func (c *ToolCache) SetFile(path, content string) {
	// 1MB以上はキャッシュしない
	if len(content) > MaxFileCacheSize {
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.files[path] = cacheEntry{
		Content:    content,
		ModTime:    info.ModTime(),
		AccessedAt: time.Now(),
	}
	pruneOldestEntries(c.files, MaxFileCacheEntries)
}

// GetDir はディレクトリ一覧のキャッシュを取得
func (c *ToolCache) GetDir(path string) (string, bool) {
	c.mu.RLock()
	entry, exists := c.dirs[path]
	c.mu.RUnlock()

	if !exists {
		return "", false
	}

	// ディレクトリの mtime をチェック
	info, err := os.Stat(path)
	if err != nil {
		c.InvalidateDir(path)
		return "", false
	}

	// ディレクトリが変更されていたらキャッシュ無効
	if info.ModTime().After(entry.ModTime) {
		c.InvalidateDir(path)
		return "", false
	}

	c.mu.Lock()
	entry.AccessedAt = time.Now()
	c.dirs[path] = entry
	c.mu.Unlock()
	return entry.Content, true
}

// SetDir はディレクトリ一覧をキャッシュに保存
func (c *ToolCache) SetDir(path, result string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.dirs[path] = cacheEntry{
		Content:    result,
		ModTime:    info.ModTime(),
		AccessedAt: time.Now(),
	}
	pruneOldestEntries(c.dirs, MaxDirCacheEntries)
}

// InvalidateFile は指定ファイルのキャッシュを無効化
func (c *ToolCache) InvalidateFile(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.files, path)
}

// InvalidateDir は指定ディレクトリのキャッシュを無効化
func (c *ToolCache) InvalidateDir(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.dirs, path)
}

// Clear は全キャッシュをクリア
func (c *ToolCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files = make(map[string]cacheEntry)
	c.dirs = make(map[string]cacheEntry)
	c.searches = make(map[string]cacheEntry)
	c.resultHashes = make(map[string]resultHashEntry)
	c.negatives = make(map[string]negativeCacheEntry)
}

// DeduplicateResult は同一内容の tool result が既に履歴に存在するか確認し、
// 重複の場合は参照文字列を返す。新規の場合はハッシュを記録して空文字を返す。
// 対象ツール: read_file, search_code, list_dir のみ。
func (c *ToolCache) DeduplicateResult(toolName, content string, turn int) string {
	if !isDeduplicableToolResult(toolName) {
		return ""
	}

	hash := contentHash(content)

	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, exists := c.resultHashes[hash]; exists {
		entry.AccessedAt = time.Now()
		c.resultHashes[hash] = entry
		return fmt.Sprintf("[Identical to previous %s result from turn %d — content omitted to save tokens]", entry.ToolName, entry.Turn)
	}

	c.resultHashes[hash] = resultHashEntry{ToolName: toolName, Turn: turn, AccessedAt: time.Now()}
	pruneOldestHashEntries(c.resultHashes, MaxResultHashEntries)
	return ""
}

// isDeduplicableToolResult は重複排除対象のツールかどうかを返す
func isDeduplicableToolResult(toolName string) bool {
	switch toolName {
	case "read_file", "search_code", "list_dir":
		return true
	}
	return false
}

// contentHash は content の SHA-256 先頭16文字を返す
func contentHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h[:8])
}

// searchCacheKey は検索キャッシュのキーを生成
func searchCacheKey(pattern, path string) string {
	return pattern + "::" + path
}

// GetSearch は検索結果のキャッシュを取得
func (c *ToolCache) GetSearch(pattern, path string) (string, bool) {
	key := searchCacheKey(pattern, path)

	c.mu.RLock()
	entry, exists := c.searches[key]
	c.mu.RUnlock()

	if !exists {
		return "", false
	}

	c.mu.Lock()
	entry.AccessedAt = time.Now()
	c.searches[key] = entry
	c.mu.Unlock()
	return entry.Content, true
}

// SetSearch は検索結果をキャッシュに保存
func (c *ToolCache) SetSearch(pattern, path, result string, affectedFiles []string) {
	key := searchCacheKey(pattern, path)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.searches[key] = cacheEntry{
		Content:       result,
		AccessedAt:    time.Now(),
		AffectedFiles: append([]string(nil), affectedFiles...),
		// 検索キャッシュはmtimeチェック不要（全クリア方式）
	}
	pruneOldestEntries(c.searches, MaxSearchCacheEntries)
}

// ClearSearchCache は検索キャッシュをクリア
func (c *ToolCache) ClearSearchCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.searches = make(map[string]cacheEntry)
}

// InvalidateSearchCacheForFile は指定ファイルパスを含む検索キャッシュエントリを無効化
func (c *ToolCache) InvalidateSearchCacheForFile(absPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, entry := range c.searches {
		for _, fp := range entry.AffectedFiles {
			if fp == absPath {
				delete(c.searches, key)
				break
			}
		}
	}
}

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

	yellow.Printf("⚠ Negative cache hit: %s previously returned: %s\n", toolName, entry.Result)
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

// Stats はキャッシュの統計情報を返す
func (c *ToolCache) Stats() (files, dirs, searches int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.files), len(c.dirs), len(c.searches)
}

// インターフェース実装の確認
var _ tools.ToolCacheInterface = (*ToolCache)(nil)

func pruneOldestHashEntries(m map[string]resultHashEntry, maxEntries int) {
	if len(m) <= maxEntries {
		return
	}

	pruneCount := len(m) - maxEntries + maxEntries/10
	if pruneCount < 1 {
		pruneCount = 1
	}

	type kv struct {
		key  string
		time time.Time
	}
	items := make([]kv, 0, len(m))
	for k, v := range m {
		items = append(items, kv{key: k, time: v.AccessedAt})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].time.Before(items[j].time)
	})
	for i := 0; i < pruneCount && i < len(items); i++ {
		delete(m, items[i].key)
	}
}

const (
	toolCacheDir     = ".xelyon/cache"
	toolCacheFile    = ".xelyon/cache/tool_cache.json"
	maxCacheFileSize = 10 * 1024 * 1024 // 10MB
)

// persistedCacheEntry はディスク永続化用のエントリ
type persistedCacheEntry struct {
	Content string    `json:"content"`
	ModTime time.Time `json:"mod_time"`
}

// persistedCache はディスク永続化用のキャッシュ全体
type persistedCache struct {
	Files map[string]persistedCacheEntry `json:"files"`
	Dirs  map[string]persistedCacheEntry `json:"dirs"`
}

// Save はファイル・ディレクトリキャッシュをディスクに永続化する。
// searches, resultHashes, negatives はセッション跨ぎで無効になりやすいため永続化しない。
func (c *ToolCache) Save() error {
	c.mu.RLock()
	pc := persistedCache{
		Files: make(map[string]persistedCacheEntry, len(c.files)),
		Dirs:  make(map[string]persistedCacheEntry, len(c.dirs)),
	}
	for k, v := range c.files {
		pc.Files[k] = persistedCacheEntry{Content: v.Content, ModTime: v.ModTime}
	}
	for k, v := range c.dirs {
		pc.Dirs[k] = persistedCacheEntry{Content: v.Content, ModTime: v.ModTime}
	}
	c.mu.RUnlock()

	// エントリがなければ保存不要（既存ファイルも削除）
	if len(pc.Files) == 0 && len(pc.Dirs) == 0 {
		_ = os.Remove(toolCacheFile)
		return nil
	}

	data, err := json.Marshal(pc)
	if err != nil {
		return fmt.Errorf("marshal tool cache: %w", err)
	}

	// サイズ上限チェック
	if len(data) > maxCacheFileSize {
		return nil
	}

	if err := os.MkdirAll(toolCacheDir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	if err := os.WriteFile(toolCacheFile, data, 0600); err != nil {
		return fmt.Errorf("write tool cache: %w", err)
	}
	return nil
}

// Load はディスクからキャッシュを復元し、mtime 検証して変更済みエントリを破棄する。
func (c *ToolCache) Load() error {
	data, err := os.ReadFile(toolCacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // ファイルなし → 正常
		}
		return fmt.Errorf("read tool cache: %w", err)
	}

	// サイズ上限チェック
	if len(data) > maxCacheFileSize {
		_ = os.Remove(toolCacheFile)
		return nil
	}

	var pc persistedCache
	if err := json.Unmarshal(data, &pc); err != nil {
		// 壊れたファイルは削除して続行
		_ = os.Remove(toolCacheFile)
		return nil
	}

	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	for path, entry := range pc.Files {
		info, err := os.Stat(path)
		if err != nil {
			continue // ファイルが存在しない → スキップ
		}
		if info.ModTime().After(entry.ModTime) {
			continue // 変更されている → スキップ
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		c.files[absPath] = cacheEntry{
			Content:    entry.Content,
			ModTime:    entry.ModTime,
			AccessedAt: now,
		}
	}

	for path, entry := range pc.Dirs {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.ModTime().After(entry.ModTime) {
			continue
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		c.dirs[absPath] = cacheEntry{
			Content:    entry.Content,
			ModTime:    entry.ModTime,
			AccessedAt: now,
		}
	}

	return nil
}

func pruneOldestEntries(m map[string]cacheEntry, maxEntries int) {
	if len(m) <= maxEntries {
		return
	}

	// 超過数 + 10% をまとめて削除（頻繁な prune を回避）
	pruneCount := len(m) - maxEntries + maxEntries/10
	if pruneCount < 1 {
		pruneCount = 1
	}

	type kv struct {
		key  string
		time time.Time
	}
	items := make([]kv, 0, len(m))
	for k, v := range m {
		items = append(items, kv{key: k, time: v.AccessedAt})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].time.Before(items[j].time)
	})
	for i := 0; i < pruneCount && i < len(items); i++ {
		delete(m, items[i].key)
	}
}
