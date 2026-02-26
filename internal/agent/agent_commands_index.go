package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/embedding"
)

// handleIndexCommand は /index コマンドの処理を行います
func (a *Agent) handleIndexCommand() error {
	cfg := config.GetGlobalConfig()
	if !cfg.Embedding.Enabled {
		return fmt.Errorf("embedding is disabled. Enable it in config.yaml to use /index")
	}

	provider := embedding.New(cfg.Embedding.BaseURL, cfg.Embedding.Model)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	fmt.Printf("Gathering files for indexing...\n")
	files := gatherIndexFiles(cwd, cfg.Embedding.Extensions)
	if len(files) == 0 {
		return fmt.Errorf("no files found to index")
	}

	fmt.Printf("Indexing... %d files\n", len(files))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	progress := func(current, total int) {
		fmt.Printf("\rIndexing... %d/%d files", current, total)
	}

	idx, err := embedding.LoadIndex(cwd, provider)
	if err != nil {
		// インデックスが存在しない場合は新規ビルド
		idx = embedding.NewIndex(cwd, provider)
		err = idx.Build(ctx, files, progress)
	} else {
		// 既存のインデックスがある場合は差分更新
		err = idx.Update(ctx, files, progress)
	}
	fmt.Println() // 改行

	if err != nil {
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "dial tcp") {
			return fmt.Errorf("failed to connect to Ollama. Is it running?\nOriginal error: %w", err)
		}
		return fmt.Errorf("indexing failed: %w", err)
	}

	// 成功したら Agent のインデックス参照を更新
	a.indexMu.Lock()
	a.index = idx
	a.indexMu.Unlock()

	// 結果表示
	fmt.Println(strings.Repeat("━", 80))
	fmt.Println("📊 Embedding Index")
	fmt.Println(strings.Repeat("━", 80))
	fmt.Println("✅ Index updated successfully")
	fmt.Printf("Files:    %d\n", len(idx.Files))
	fmt.Printf("Chunks:   %d\n", len(idx.Chunks))
	fmt.Printf("Dims:     %d\n", idx.Dims)
	fmt.Printf("Model:    %s\n", idx.Model)
	fmt.Printf("Dir:      %s\n", ".xelyon/index/")
	fmt.Printf("Updated:  %s\n", time.Now().Format(time.RFC3339))
	fmt.Println(strings.Repeat("━", 80))

	return nil
}

// updateIndexBackground はバックグラウンドでインデックスの差分更新を行います
func (a *Agent) updateIndexBackground(cwd string, provider *embedding.Provider) error {
	cfg := config.GetGlobalConfig()
	files := gatherIndexFiles(cwd, cfg.Embedding.Extensions)
	if len(files) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	idx, err := embedding.LoadIndex(cwd, provider)
	if err != nil {
		// インデックスが存在しない場合は何もしない（初回は /index 手動のみ）
		return nil
	}

	if err := idx.Update(ctx, files, nil); err != nil {
		// Ollama未起動時など → 静かに失敗
		return err
	}
	a.indexMu.Lock()
	a.index = idx
	a.indexMu.Unlock()
	return nil
}

// gatherIndexFiles はインデックス対象のファイルを収集します
func gatherIndexFiles(root string, extensions []string) []string {
	var files []string

	// 1. git ls-files を試行
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = root
	out, err := cmd.Output()
	if err == nil {
		// gitリポジトリ内: 結果が0件でもWalkDirにフォールバックしない
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			path := filepath.Join(root, line)
			if isValidIndexFile(path, extensions) {
				files = append(files, path)
			}
		}
		return files
	}

	// 2. git が使えない場合は filepath.WalkDir でフォールバック
	extMap := make(map[string]bool)
	for _, ext := range extensions {
		extMap[ext] = true
	}

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == ".xelyon" {
				return filepath.SkipDir
			}
			return nil
		}
		if isValidIndexFile(path, extensions) {
			files = append(files, path)
		}
		return nil
	})

	return files
}

// isValidIndexFile は対象ファイルかどうかを判定します
func isValidIndexFile(path string, extensions []string) bool {
	name := filepath.Base(path)

	// 除外ファイル
	switch name {
	case "go.mod", "go.sum", "package-lock.json", "yarn.lock", "pnpm-lock.yaml":
		return false
	}
	if strings.HasSuffix(name, ".min.js") || strings.HasSuffix(name, ".min.css") || strings.Contains(name, ".generated.") {
		return false
	}

	// 特殊ファイル名の一致
	if name == "Makefile" || name == "Dockerfile" {
		for _, ext := range extensions {
			if ext == name {
				return true
			}
		}
	}

	// 拡張子の一致
	ext := filepath.Ext(path)
	for _, validExt := range extensions {
		if ext == validExt {
			return true
		}
	}

	return false
}
