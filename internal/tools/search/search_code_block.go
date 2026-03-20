package search

import (
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
