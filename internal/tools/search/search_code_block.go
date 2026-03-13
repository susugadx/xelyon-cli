package search

import (
	"fmt"
	"os"
	"path/filepath"

	internalast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// matchBlock はマッチ行とその前後のコンテキスト行をまとめたブロック
type matchBlock struct {
	matches []Match
	typ     MatchType
}

// buildMatchBlocks は deduped な Match リストをブロックに分割する
// コンテキスト行はマッチ行に到達するまで pending に蓄積し、次のマッチブロックの先頭に付与
func buildMatchBlocks(matches []Match) []matchBlock {
	var blocks []matchBlock
	var pending []Match

	for _, m := range matches {
		if m.IsMatch {
			var blockMatches []Match
			blockMatches = append(blockMatches, pending...)
			blockMatches = append(blockMatches, m)
			blocks = append(blocks, matchBlock{typ: m.Type, matches: blockMatches})
			pending = nil
		} else {
			pending = append(pending, m)
		}
	}
	if len(pending) > 0 && len(blocks) > 0 {
		last := &blocks[len(blocks)-1]
		last.matches = append(last.matches, pending...)
	}
	return blocks
}

// findBlockForLine は指定行番号を含む最内ブロックを返す
func findBlockForLine(ranges []common.BlockRange, lineNum int) *BlockInfo {
	var best *common.BlockRange
	for i := range ranges {
		r := &ranges[i]
		if lineNum >= r.StartLine && lineNum <= r.EndLine {
			if best == nil || r.StartLine > best.StartLine {
				best = r
			}
		}
	}
	if best == nil {
		return nil
	}
	return &BlockInfo{Name: best.Name, StartLine: best.StartLine}
}

// getFileContentWithCache はファイル内容を取得する（キャッシュ連携）
func getFileContentWithCache(cache tools.ToolCacheInterface, filePath string) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return ""
	}
	if cache != nil {
		if cached, ok := cache.GetFile(absPath); ok {
			return cached
		}
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}
	content := string(data)
	if cache != nil {
		cache.SetFile(absPath, content)
	}
	return content
}

func detectBlocksWithCache(cache tools.ToolCacheInterface, results []SearchResult) {
	for i := range results {
		r := &results[i]
		if internalast.IsSupportedFile(r.FilePath) {
			continue
		}
		content := getFileContentWithCache(cache, r.FilePath)
		if content == "" {
			continue
		}
		isBrace := common.IsBraceLanguage(r.FilePath)
		blocks := common.BuildBlockMap(content, isBrace)
		for j := range r.Matches {
			m := &r.Matches[j]
			if !m.IsMatch || m.Block != nil {
				continue
			}
			m.Block = findBlockForLine(blocks, m.LineNum)
			if m.Block != nil && m.Block.StartLine == m.LineNum {
				m.Block = nil
			}
		}
	}
}

// collapseBlockMatches は同一ブロック内に3つ以上のマッチがある場合、中間マッチを圧縮する。
// 先頭マッチの前コンテキスト＋先頭マッチ＋圧縮マーカー＋末尾マッチ＋末尾マッチの後コンテキストだけ残す。
// 圧縮マーカーは LineNum=-1 の合成 Match で表現する。
func collapseBlockMatches(results []SearchResult) {
	for i := range results {
		r := &results[i]
		type blockKey struct {
			Name      string
			StartLine int
		}
		type matchIdx struct {
			idx     int
			lineNum int
		}
		groups := make(map[blockKey][]matchIdx)
		var order []blockKey

		for j, m := range r.Matches {
			if !m.IsMatch || m.Block == nil {
				continue
			}
			key := blockKey{Name: m.Block.Name, StartLine: m.Block.StartLine}
			if _, exists := groups[key]; !exists {
				order = append(order, key)
			}
			groups[key] = append(groups[key], matchIdx{idx: j, lineNum: m.LineNum})
		}

		removeSet := make(map[int]bool)
		type insertion struct {
			afterIdx int
			marker   Match
		}
		var insertions []insertion

		for _, key := range order {
			idxs := groups[key]
			if len(idxs) < 3 {
				continue
			}

			first := idxs[0]
			last := idxs[len(idxs)-1]
			collapsedCount := len(idxs) - 2

			for _, mi := range idxs[1 : len(idxs)-1] {
				removeSet[mi.idx] = true
			}

			for j := first.idx + 1; j < last.idx; j++ {
				if !r.Matches[j].IsMatch {
					removeSet[j] = true
				}
			}

			insertions = append(insertions, insertion{
				afterIdx: first.idx,
				marker: Match{
					LineNum: -1,
					Line:    fmt.Sprintf("[+%d more matches in %s]", collapsedCount, key.Name),
					IsMatch: false,
				},
			})
		}

		if len(removeSet) == 0 {
			continue
		}

		insertMap := make(map[int]Match)
		for _, ins := range insertions {
			insertMap[ins.afterIdx] = ins.marker
		}

		newMatches := make([]Match, 0, len(r.Matches))
		for j, m := range r.Matches {
			if !removeSet[j] {
				newMatches = append(newMatches, m)
			}
			if marker, ok := insertMap[j]; ok {
				newMatches = append(newMatches, marker)
			}
		}
		r.Matches = newMatches
	}
}
