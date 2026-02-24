package tools

import (
	"sort"
	"sync"
)

// LineRange は行範囲を表す（1-indexed, inclusive）。
type LineRange struct {
	Start int
	End   int
}

// ReadTracker はセッション内で read_file されたファイルパスを追跡する。
// 書き込み系ツール実行前に「このファイルを読んだか？」をチェックするために使用。
type ReadTracker struct {
	mu     sync.RWMutex
	paths  map[string]bool        // ファイル全体の既読（read_file 用）
	ranges map[string][]LineRange // 行範囲の既読（search_code 用）
}

// GlobalReadTracker はグローバルな ReadTracker インスタンス。
// agent.NewAgent() と /clear コマンドで Reset される。
var GlobalReadTracker = NewReadTracker()

// NewReadTracker は新しい ReadTracker を作成する。
func NewReadTracker() *ReadTracker {
	return &ReadTracker{
		paths:  make(map[string]bool),
		ranges: make(map[string][]LineRange),
	}
}

// MarkRead はファイルを「読み済み」として記録する（absPath で保存）。
func (rt *ReadTracker) MarkRead(absPath string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.paths[absPath] = true
}

// IsRead はファイル全体が読み済みかを返す。
func (rt *ReadTracker) IsRead(absPath string) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.paths[absPath]
}

// MarkReadRange は指定行範囲を「読み済み」として記録する。
// 重複・隣接する範囲は自動的にマージされる。
func (rt *ReadTracker) MarkReadRange(absPath string, start, end int) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.ranges[absPath] = mergeRanges(append(rt.ranges[absPath], LineRange{Start: start, End: end}))
}

// IsReadLine は指定行が読み済みかを返す。
// ファイル全体が既読（MarkRead）なら常に true。
func (rt *ReadTracker) IsReadLine(absPath string, lineNum int) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	if rt.paths[absPath] {
		return true
	}
	for _, r := range rt.ranges[absPath] {
		if lineNum >= r.Start && lineNum <= r.End {
			return true
		}
	}
	return false
}

// IsReadRange は指定行範囲が完全に読み済みかを返す。
// ファイル全体が既読なら常に true。
// マージ済みの単一 range が [start, end] を包含するか確認する。
func (rt *ReadTracker) IsReadRange(absPath string, start, end int) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	if rt.paths[absPath] {
		return true
	}
	for _, r := range rt.ranges[absPath] {
		if r.Start <= start && r.End >= end {
			return true
		}
	}
	return false
}

// InvalidateFile は指定ファイルの既読フラグをリセットする。
// 書き込み系ツール実行後に呼ばれ、次回編集前の read_file を強制する。
func (rt *ReadTracker) InvalidateFile(absPath string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.paths, absPath)
	delete(rt.ranges, absPath)
}

// Reset はトラッカーをクリアする（新セッション開始時・/clear 時）。
func (rt *ReadTracker) Reset() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.paths = make(map[string]bool)
	rt.ranges = make(map[string][]LineRange)
}

// mergeRanges は LineRange スライスをソートし、重複・隣接する範囲をマージする。
func mergeRanges(ranges []LineRange) []LineRange {
	if len(ranges) <= 1 {
		return ranges
	}
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start < ranges[j].Start
	})
	merged := []LineRange{ranges[0]}
	for _, r := range ranges[1:] {
		last := &merged[len(merged)-1]
		if r.Start <= last.End+1 { // 隣接（+1）もマージ
			if r.End > last.End {
				last.End = r.End
			}
		} else {
			merged = append(merged, r)
		}
	}
	return merged
}
