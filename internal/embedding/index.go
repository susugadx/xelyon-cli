package embedding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	IndexVersion = "1.0"
)

type Index struct {
	Dir       string      // インデックス保存ディレクトリ
	Provider  *Provider   // Embedding API
	Vectors   [][]float32 // 全チャンクのベクトル
	Chunks    []ChunkMeta // メタデータ（meta.json対応）
	Files     map[string]FileMeta
	Dims      int
	Model     string
	CreatedAt string
}

type ChunkMeta struct {
	FilePath  string `json:"file"`
	StartLine int    `json:"start"`
	EndLine   int    `json:"end"`
	BlockName string `json:"block"`
	Hash      string `json:"hash"` // チャンク内容のSHA256
}

type FileMeta struct {
	Mtime string `json:"mtime"`
	Hash  string `json:"hash"`
}

type indexMeta struct {
	Version     string `json:"version"`
	Model       string `json:"model"`
	Dims        int    `json:"dims"`
	TotalChunks int    `json:"total_chunks"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type SearchResult struct {
	ChunkMeta
	Score   float32
	Content string // チャンクのテキスト内容
}

// NewIndex は新しいIndexインスタンスを作成します
func NewIndex(projectRoot string, provider *Provider) *Index {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		absRoot = projectRoot
	}
	hash := sha256.Sum256([]byte(absRoot))
	hashStr := hex.EncodeToString(hash[:])[:16]

	homeDir, _ := os.UserHomeDir()
	indexDir := filepath.Join(homeDir, ".xelyon", "embedding_index", hashStr)

	return &Index{
		Dir:      indexDir,
		Provider: provider,
		Files:    make(map[string]FileMeta),
	}
}

// hashContent は文字列のSHA256ハッシュを計算します
func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// hashFile はファイルのSHA256ハッシュを計算します
func hashFile(filePath string) (string, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return hashContent(string(b)), nil
}

// Build は指定されたファイルのインデックスを構築します
func (idx *Index) Build(ctx context.Context, files []string) error {
	idx.Vectors = nil
	idx.Chunks = nil
	idx.Files = make(map[string]FileMeta)

	return idx.processFiles(ctx, files, false)
}

func (idx *Index) processFiles(ctx context.Context, files []string, isUpdate bool) error {
	var allTexts []string
	var allChunks []ChunkMeta

	for _, file := range files {
		// skip missing or empty files
		info, err := os.Stat(file)
		if err != nil || info.Size() == 0 {
			continue
		}

		b, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		content := string(b)
		fileHash := hashContent(content)

		chunks := ChunkFile(file, content)
		if len(chunks) == 0 {
			continue
		}

		idx.Files[file] = FileMeta{
			Mtime: info.ModTime().Format(time.RFC3339),
			Hash:  fileHash,
		}

		for _, c := range chunks {
			chunkContent := extractLines(content, c.StartLine, c.EndLine)
			chunkHash := hashContent(chunkContent)

			allTexts = append(allTexts, chunkContent)
			allChunks = append(allChunks, ChunkMeta{
				FilePath:  c.FilePath,
				StartLine: c.StartLine,
				EndLine:   c.EndLine,
				BlockName: c.BlockName,
				Hash:      chunkHash,
			})
		}
	}

	if len(allTexts) == 0 {
		if !isUpdate {
			return idx.Save()
		}
		return nil
	}

	// バッチでEmbedding
	const batchSize = 100
	var allVectors [][]float32

	for i := 0; i < len(allTexts); i += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		end := i + batchSize
		if end > len(allTexts) {
			end = len(allTexts)
		}

		vecs, err := idx.Provider.Embed(ctx, allTexts[i:end])
		if err != nil {
			return fmt.Errorf("embedding error: %w", err)
		}
		allVectors = append(allVectors, vecs...)
	}

	if len(allVectors) > 0 {
		idx.Dims = len(allVectors[0])
		idx.Model = idx.Provider.model
	}

	idx.Vectors = append(idx.Vectors, allVectors...)
	idx.Chunks = append(idx.Chunks, allChunks...)

	return idx.Save()
}

func extractLines(content string, start, end int) string {
	lines := strings.Split(content, "\n")
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return ""
	}
	return strings.Join(lines[start-1:end], "\n")
}

// Save はインデックスをディスクに保存します
func (idx *Index) Save() error {
	if len(idx.Vectors) != len(idx.Chunks) {
		return fmt.Errorf("index corrupted: vectors(%d) != chunks(%d)", len(idx.Vectors), len(idx.Chunks))
	}

	if err := os.MkdirAll(idx.Dir, 0755); err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	if idx.CreatedAt == "" {
		idx.CreatedAt = now
	}

	// index.json
	meta := indexMeta{
		Version:     IndexVersion,
		Model:       idx.Model,
		Dims:        idx.Dims,
		TotalChunks: len(idx.Chunks),
		CreatedAt:   idx.CreatedAt,
		UpdatedAt:   now,
	}
	bMeta, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(idx.Dir, "index.json"), bMeta); err != nil {
		return err
	}

	// meta.json (Chunks + Files)
	type dataMeta struct {
		Chunks []ChunkMeta         `json:"chunks"`
		Files  map[string]FileMeta `json:"files"`
	}
	dm := dataMeta{
		Chunks: idx.Chunks,
		Files:  idx.Files,
	}
	bData, err := json.MarshalIndent(dm, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(idx.Dir, "meta.json"), bData); err != nil {
		return err
	}

	// vectors.bin
	var buf bytes.Buffer
	for _, vec := range idx.Vectors {
		if err := binary.Write(&buf, binary.LittleEndian, vec); err != nil {
			return err
		}
	}
	return writeAtomic(filepath.Join(idx.Dir, "vectors.bin"), buf.Bytes())
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp) // cleanup
		return err
	}
	return nil
}

// LoadIndex はインデックスをディスクから読み込みます
func LoadIndex(projectRoot string, provider *Provider) (*Index, error) {
	idx := NewIndex(projectRoot, provider)

	// index.json
	bIdx, err := os.ReadFile(filepath.Join(idx.Dir, "index.json"))
	if err != nil {
		return nil, err
	}
	var meta indexMeta
	if err := json.Unmarshal(bIdx, &meta); err != nil {
		return nil, err
	}
	idx.Dims = meta.Dims
	idx.Model = meta.Model
	idx.CreatedAt = meta.CreatedAt

	// meta.json
	bMeta, err := os.ReadFile(filepath.Join(idx.Dir, "meta.json"))
	if err != nil {
		return nil, err
	}
	type dataMeta struct {
		Chunks []ChunkMeta         `json:"chunks"`
		Files  map[string]FileMeta `json:"files"`
	}
	var dm dataMeta
	if err := json.Unmarshal(bMeta, &dm); err != nil {
		return nil, err
	}
	idx.Chunks = dm.Chunks
	idx.Files = dm.Files

	// vectors.bin
	bVecs, err := os.ReadFile(filepath.Join(idx.Dir, "vectors.bin"))
	if err != nil {
		return nil, err
	}

	if idx.Dims > 0 {
		numVectors := len(bVecs) / (idx.Dims * 4)
		idx.Vectors = make([][]float32, numVectors)
		r := bytes.NewReader(bVecs)
		for i := 0; i < numVectors; i++ {
			vec := make([]float32, idx.Dims)
			if err := binary.Read(r, binary.LittleEndian, &vec); err != nil {
				return nil, err
			}
			idx.Vectors[i] = vec
		}
	}

	return idx, nil
}

// cosineSimilarity はコサイン類似度を計算します（ベクトルはL2正規化済みと仮定）
func cosineSimilarity(a, b []float32) float32 {
	var dot float32
	for i := range a {
		dot += a[i] * b[i]
	}
	return dot
}

// Search は類似度検索を実行します
func (idx *Index) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if len(idx.Vectors) == 0 {
		return nil, nil
	}

	qVec, err := idx.Provider.EmbedSingle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query embed error: %w", err)
	}

	type score struct {
		index int
		val   float32
	}
	var scores []score
	for i, v := range idx.Vectors {
		scores = append(scores, score{i, cosineSimilarity(qVec, v)})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].val > scores[j].val
	})

	fileCache := make(map[string]string) // filePath -> content
	var results []SearchResult
	for i := 0; i < topK && i < len(scores); i++ {
		s := scores[i]
		meta := idx.Chunks[s.index]

		content := ""
		if cached, ok := fileCache[meta.FilePath]; ok {
			content = extractLines(cached, meta.StartLine, meta.EndLine)
		} else {
			b, err := os.ReadFile(meta.FilePath)
			if err == nil {
				fileContent := string(b)
				fileCache[meta.FilePath] = fileContent
				content = extractLines(fileContent, meta.StartLine, meta.EndLine)
			}
		}

		results = append(results, SearchResult{
			ChunkMeta: meta,
			Score:     s.val,
			Content:   content,
		})
	}

	return results, nil
}

// Update は差分更新を行います
func (idx *Index) Update(ctx context.Context, files []string) error {
	// 変更・追加されたファイル、削除されたファイルを検出
	var changedFiles []string
	deletedFiles := make(map[string]bool)

	if len(files) == 0 {
		// git 管理下かチェック
		cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
		if err := cmd.Run(); err == nil {
			// git管理下
			topCmd := exec.Command("git", "rev-parse", "--show-toplevel")
			topOut, _ := topCmd.Output()
			repoRoot := strings.TrimSpace(string(topOut))

			changedSet := make(map[string]bool)

			// 1. untracked/modified を取得
			statusCmd := exec.Command("git", "status", "--porcelain")
			if out, err := statusCmd.Output(); err == nil {
				lines := strings.Split(string(out), "\n")
				for _, line := range lines {
					if len(line) < 4 {
						continue
					}
					status := line[:2]
					file := strings.TrimSpace(line[3:])
					file = filepath.Join(repoRoot, file)

					// D は削除, それ以外(M, A, ??など)は変更/追加扱い
					if strings.Contains(status, "D") {
						deletedFiles[file] = true
					} else {
						changedFiles = append(changedFiles, file)
						changedSet[file] = true
					}
				}
			}
			// 2. 差分（trackedだが未コミットなど）をさらに取得
			diffCmd := exec.Command("git", "diff", "--name-only", "HEAD")
			if out, err := diffCmd.Output(); err == nil {
				lines := strings.Split(string(out), "\n")
				for _, file := range lines {
					file = strings.TrimSpace(file)
					if file == "" {
						continue
					}
					file = filepath.Join(repoRoot, file)

					if _, err := os.Stat(file); err != nil {
						deletedFiles[file] = true
					} else {
						// 重複チェック
						if !changedSet[file] {
							changedFiles = append(changedFiles, file)
							changedSet[file] = true
						}
					}
				}
			}
		} else {
			// git 非管理（フォールバック）
			for f, m := range idx.Files {
				info, err := os.Stat(f)
				if err != nil {
					deletedFiles[f] = true
					continue
				}
				h, err := hashFile(f)
				if err != nil || h != m.Hash || info.ModTime().Format(time.RFC3339) != m.Mtime {
					changedFiles = append(changedFiles, f)
				}
			}
		}
	} else {
		for _, f := range files {
			_, err := os.Stat(f)
			if err != nil {
				deletedFiles[f] = true
			} else {
				changedFiles = append(changedFiles, f)
			}
		}
	}

	if len(changedFiles) == 0 && len(deletedFiles) == 0 {
		return nil // 変更なし
	}

	// 削除対象のチャンクを除外
	var newChunks []ChunkMeta
	var newVectors [][]float32

	changedMap := make(map[string]bool)
	for _, f := range changedFiles {
		changedMap[f] = true
	}

	for i, c := range idx.Chunks {
		if deletedFiles[c.FilePath] || changedMap[c.FilePath] {
			continue // 削除 or 更新対象なので古いチャンクは捨てる
		}
		newChunks = append(newChunks, c)
		newVectors = append(newVectors, idx.Vectors[i])
	}

	idx.Chunks = newChunks
	idx.Vectors = newVectors

	for f := range deletedFiles {
		delete(idx.Files, f)
	}

	// 変更・追加ファイルのチャンクを再計算して追加
	if len(changedFiles) > 0 {
		return idx.processFiles(ctx, changedFiles, true)
	}

	return idx.Save()
}
