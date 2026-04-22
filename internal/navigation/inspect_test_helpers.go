package navigation

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestGoFiles は複数の Go ファイルを一時ディレクトリに作成し、
// そのディレクトリに cd した後、元に戻すための cleanup を登録する。
func setupTestGoFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: could not restore directory: %v", err)
		}
	})
	return dir
}

// setupTestGoFile はテスト用の Go ファイルを一時ディレクトリに作成し、
// そのディレクトリに cd した後、元に戻すための cleanup を返す。
func setupTestGoFile(t *testing.T, filename, content string) string {
	return setupTestGoFiles(t, map[string]string{filename: content})
}

const testGoSource = `package example

import "fmt"

// Build は何かをビルドする。
func Build(name string) error {
	fmt.Println(name)
	return nil
}

// Run は Build を呼ぶ。
func Run() {
	Build("test")
}

type Config struct {
	Name string
}

func (c *Config) Build() string {
	return c.Name
}

var DefaultBuilder = Build
`
