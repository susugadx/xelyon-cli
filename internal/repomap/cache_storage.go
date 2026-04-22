package repomap

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

func readMapCacheData(rootPath string) ([]byte, error) {
	cachePath, err := cacheFilePath(rootPath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(cachePath)
}

func writeMapCacheData(rootPath string, data []byte) error {
	cachePath, err := cacheFilePath(rootPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(cachePath, data, 0600)
}

func cacheFilePath(rootPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(filepath.Clean(rootPath)))
	name := hex.EncodeToString(sum[:])[:12] + ".json"
	return filepath.Join(home, ".xelyon", "cache", "projectmap", name), nil
}
