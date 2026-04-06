package locator

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Location はコードベース上の位置を表す。
type Location struct {
	FilePath     string // Display path shown to the model. Usually project-root-relative, but some producers keep their original relative form.
	ResolvedPath string // Optional absolute path used for follow-up reads/edits when display path is ambiguous.
	Line         int    // 開始行（1-based）
	EndLine      int    // 終了行（1-based、0の場合はLine単体）
	Name         string // シンボル名（例: "func computeReplacements"）。テキスト検索マッチの場合は空。
}

// Registry はLocator IDと位置情報のマッピングを管理する。
// エージェントインスタンスごとに1つ作成し、セッション中は追記のみ（削除・上書きしない）。
type Registry struct {
	mu      sync.RWMutex
	entries []Location
	dedup   map[string]int // "filepath:line:endline:name" → entries index（重複登録防止）
}

// NewRegistry は新しい空のRegistryを作成する。
func NewRegistry() *Registry {
	return &Registry{
		dedup: make(map[string]int),
	}
}

// Register は位置情報を登録し、Locator ID文字列（例: "[L1]"）を返す。
// 同じ位置が既に登録済みなら、既存のIDを返す（冪等）。
func (r *Registry) Register(loc Location) string {
	key := dedupKey(loc)

	r.mu.RLock()
	if idx, ok := r.dedup[key]; ok {
		r.mu.RUnlock()
		return formatID(idx)
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// double-check after write lock
	if idx, ok := r.dedup[key]; ok {
		return formatID(idx)
	}

	idx := len(r.entries)
	r.entries = append(r.entries, loc)
	r.dedup[key] = idx
	return formatID(idx)
}

// Resolve はLocator ID（"[L5]" または "L5"）からLocationを逆引きする。
func (r *Registry) Resolve(id string) (Location, bool) {
	idx, ok := parseID(id)
	if !ok {
		return Location{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if idx < 0 || idx >= len(r.entries) {
		return Location{}, false
	}
	return r.entries[idx], true
}

// ResolveMulti は複数のLocator IDを一括で逆引きする。
// 形式: "[L1,L5,L12]" または "[L1, L5, L12]"
// 解決できないIDはスキップし、解決できたLocationのみ返す。
func (r *Registry) ResolveMulti(raw string) []Location {
	raw = strings.Trim(raw, "[] ")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")

	// 一括ロックでスナップショット一貫性を保証
	r.mu.RLock()
	defer r.mu.RUnlock()

	var locs []Location
	for _, p := range parts {
		p = strings.TrimSpace(p)
		idx, ok := parseID(p)
		if !ok || idx < 0 || idx >= len(r.entries) {
			continue
		}
		locs = append(locs, r.entries[idx])
	}
	return locs
}

func dedupKey(loc Location) string {
	// NULセパレータでフィールド値内のコロン等との衝突を防止
	return loc.FilePath + "\x00" + normalizedResolvedPathKey(loc) + "\x00" + strconv.Itoa(loc.Line) + "\x00" + strconv.Itoa(loc.EndLine) + "\x00" + loc.Name
}

func normalizedResolvedPathKey(loc Location) string {
	resolvedPath := strings.TrimSpace(loc.ResolvedPath)
	if resolvedPath == "" {
		return ""
	}
	resolvedPath = filepath.Clean(resolvedPath)

	filePath := strings.TrimSpace(loc.FilePath)
	if filepath.IsAbs(filePath) {
		filePath = filepath.Clean(filePath)
		if filePath == resolvedPath {
			return ""
		}
	}
	return resolvedPath
}

func formatID(idx int) string {
	return fmt.Sprintf("[L%d]", idx+1) // 1-based: entries[0] → [L1]
}

// parseID は "L5", "[L5]" を解析して 0-based index を返す。
func parseID(s string) (int, bool) {
	s = strings.Trim(s, "[] ")
	if !strings.HasPrefix(s, "L") && !strings.HasPrefix(s, "l") {
		return 0, false
	}
	n, err := strconv.Atoi(s[1:])
	if err != nil || n < 1 {
		return 0, false
	}
	return n - 1, true // 1-based → 0-based
}
