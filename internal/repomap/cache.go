//go:build !norepomap
// +build !norepomap

package repomap

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CachedFile はキャッシュされたファイル情報
type CachedFile struct {
	Path    string    `json:"path"`
	ModTime time.Time `json:"mod_time"`
	Symbols []Symbol  `json:"symbols"`
}

// RepoMapCache はRepoMapのキャッシュ
type RepoMapCache struct {
	RootPath  string                 `json:"root_path"`
	UpdatedAt time.Time              `json:"updated_at"`
	Files     map[string]*CachedFile `json:"files"`
}

// getCachePath はキャッシュファイルのパスを返す
func getCachePath(rootPath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(homeDir, ".xelyon", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", md5.Sum([]byte(rootPath)))[:12]
	return filepath.Join(cacheDir, "repomap_"+hash+".json"), nil
}

// LoadCache はキャッシュを読み込む
func LoadCache(rootPath string) (*RepoMapCache, error) {
	cachePath, err := getCachePath(rootPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}
	var cache RepoMapCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

// SaveCache はキャッシュを保存する
func SaveCache(cache *RepoMapCache) error {
	cachePath, err := getCachePath(cache.RootPath)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath, data, 0644)
}

// getGitChangedFiles は git status で変更ファイルを取得
// git リポジトリでない場合は nil を返す（エラーではない）
func getGitChangedFiles(rootPath string) map[string]bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = rootPath
	output, err := cmd.Output()
	if err != nil {
		return nil // git リポジトリじゃない
	}

	changed := make(map[string]bool)
	for _, line := range strings.Split(string(output), "\n") {
		if len(line) > 3 {
			file := strings.TrimSpace(line[3:])
			// リネームの場合は -> 以降を取得
			if idx := strings.Index(file, " -> "); idx != -1 {
				file = file[idx+4:]
			}
			changed[filepath.Join(rootPath, file)] = true
		}
	}
	return changed
}
