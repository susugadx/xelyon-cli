package agent

import (
	"os"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ToolCache はツール結果のキャッシュ
// read_file, list_dir の結果をキャッシュしてトークン消費を削減
type ToolCache struct {
	files map[string]cacheEntry
	dirs  map[string]cacheEntry
	mu    sync.RWMutex
}

// cacheEntry はキャッシュエントリ
type cacheEntry struct {
	Content string
	ModTime time.Time
}

// MaxFileCacheSize はキャッシュする最大ファイルサイズ (1MB)
const MaxFileCacheSize = 1024 * 1024

// NewToolCache は新しい ToolCache を作成
func NewToolCache() *ToolCache {
	return &ToolCache{
		files: make(map[string]cacheEntry),
		dirs:  make(map[string]cacheEntry),
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

	green.Printf("📦 Cache hit: %s\n", path)
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
		Content: content,
		ModTime: info.ModTime(),
	}
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

	green.Printf("📦 Cache hit: %s\n", path)
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
		Content: result,
		ModTime: info.ModTime(),
	}
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
}

// Stats はキャッシュの統計情報を返す
func (c *ToolCache) Stats() (files, dirs int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.files), len(c.dirs)
}

// インターフェース実装の確認
var _ tools.ToolCacheInterface = (*ToolCache)(nil)
