package embedding

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// Chunk はEmbedding用のコードチャンク
type Chunk struct {
	FilePath  string // ファイルパス
	StartLine int    // 開始行番号
	EndLine   int    // 終了行番号
	Content   string // チャンクのテキスト内容
	BlockName string // ブロック名（"func retryWithBackoff" 等、トップレベルなら空）
}

// ChunkFile は1つのファイルをチャンクに分割する
func ChunkFile(filePath string, content string) []Chunk {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// 空ファイルまたは極小ファイル（5行以下）は1チャンクとして返す
	if totalLines == 0 || (totalLines == 1 && lines[0] == "") {
		return []Chunk{}
	}
	if totalLines <= 5 {
		return []Chunk{{
			FilePath:  filePath,
			StartLine: 1,
			EndLine:   totalLines,
			Content:   content,
			BlockName: "",
		}}
	}

	isBrace := common.IsBraceLanguage(filePath)
	allBlocks := common.BuildBlockMap(content, isBrace)

	// 入れ子になったブロックを除外（トップレベルのブロックのみを対象とする）
	var blocks []common.BlockRange
	var lastEnd int
	for _, b := range allBlocks {
		if b.StartLine <= lastEnd {
			continue // すでに処理対象のブロックに包含されている
		}
		blocks = append(blocks, b)
		lastEnd = b.EndLine
	}

	var chunks []Chunk
	processedLines := make([]bool, totalLines+1) // 1-indexed

	// 1. ブロックごとにチャンクを作成
	for _, b := range blocks {
		start := b.StartLine
		end := b.EndLine
		if end > totalLines {
			end = totalLines
		}

		blockLines := end - start + 1
		if blockLines > 100 {
			// 巨大ブロックの分割 (50行ごと、重複10行)
			for s := start; s <= end; s += 40 {
				e := s + 49
				if e > end {
					e = end
				}
				chunkContent := strings.Join(lines[s-1:e], "\n")
				chunks = append(chunks, Chunk{
					FilePath:  filePath,
					StartLine: s,
					EndLine:   e,
					Content:   chunkContent,
					BlockName: b.Name,
				})
				if e == end {
					break
				}
			}
		} else {
			// 通常サイズのブロック
			chunkContent := strings.Join(lines[start-1:end], "\n")
			chunks = append(chunks, Chunk{
				FilePath:  filePath,
				StartLine: start,
				EndLine:   end,
				Content:   chunkContent,
				BlockName: b.Name,
			})
		}

		for i := start; i <= end; i++ {
			processedLines[i] = true
		}
	}

	// 2. ブロックに属さないトップレベルコードを収集
	var currentStart int
	var currentLines []string

	for i := 1; i <= totalLines; i++ {
		if !processedLines[i] {
			if currentStart == 0 {
				currentStart = i
			}
			currentLines = append(currentLines, lines[i-1])
		} else {
			if currentStart != 0 {
				chunkContent := strings.Join(currentLines, "\n")
				if strings.TrimSpace(chunkContent) != "" {
					chunks = append(chunks, Chunk{
						FilePath:  filePath,
						StartLine: currentStart,
						EndLine:   i - 1,
						Content:   chunkContent,
						BlockName: "",
					})
				}
				currentStart = 0
				currentLines = nil
			}
		}
	}
	// 終端処理
	if currentStart != 0 {
		chunkContent := strings.Join(currentLines, "\n")
		if strings.TrimSpace(chunkContent) != "" {
			chunks = append(chunks, Chunk{
				FilePath:  filePath,
				StartLine: currentStart,
				EndLine:   totalLines,
				Content:   chunkContent,
				BlockName: "",
			})
		}
	}

	return chunks
}
