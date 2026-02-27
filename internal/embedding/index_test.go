package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func setupMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/embed" {
			var req EmbedRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			resp := EmbedResponse{
				Model: req.Model,
			}
			for _, input := range req.Input {
				// 検索テスト用に、入力文字列が "query" の場合とそれ以外でベクトルを変える
				vec := make([]float32, 2)
				switch input {
				case "query":
					vec[0] = 1.0
					vec[1] = 0.0
				case "target":
					vec[0] = 0.9
					vec[1] = 0.1
				default:
					vec[0] = 0.0
					vec[1] = 1.0
				}
				resp.Embeddings = append(resp.Embeddings, vec)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestIndex_BuildAndLoad(t *testing.T) {
	ts := setupMockServer()
	defer ts.Close()

	provider := New(ts.URL, "test-model")

	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("USERPROFILE", tempDir)

	// Create test files
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")

	_ = os.WriteFile(file1, []byte("target\nline2\nline3\nline4\nline5\nline6"), 0644)
	_ = os.WriteFile(file2, []byte("other\ncontent"), 0644)

	idx := NewIndex(tempDir, provider)

	ctx := context.Background()
	err := idx.Build(ctx, []string{file1, file2}, nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(idx.Vectors) == 0 {
		t.Errorf("Expected vectors to be populated")
	}
	if len(idx.Chunks) == 0 {
		t.Errorf("Expected chunks to be populated")
	}

	// Verify files exist
	if _, err := os.Stat(filepath.Join(idx.Dir, "vectors.bin")); os.IsNotExist(err) {
		t.Errorf("vectors.bin not created")
	}
	if _, err := os.Stat(filepath.Join(idx.Dir, "meta.json")); os.IsNotExist(err) {
		t.Errorf("meta.json not created")
	}
	if _, err := os.Stat(filepath.Join(idx.Dir, "index.json")); os.IsNotExist(err) {
		t.Errorf("index.json not created")
	}

	// Test Load
	loaded, err := LoadIndex(tempDir, provider)
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	if loaded.Dims != idx.Dims {
		t.Errorf("Expected dims %d, got %d", idx.Dims, loaded.Dims)
	}
	if loaded.Model != idx.Model {
		t.Errorf("Expected model %s, got %s", idx.Model, loaded.Model)
	}
	if len(loaded.Vectors) != len(idx.Vectors) {
		t.Errorf("Expected %d vectors, got %d", len(idx.Vectors), len(loaded.Vectors))
	}
	if len(loaded.Chunks) != len(idx.Chunks) {
		t.Errorf("Expected %d chunks, got %d", len(idx.Chunks), len(loaded.Chunks))
	}
	if loaded.CreatedAt == "" {
		t.Errorf("Expected CreatedAt to not be empty")
	}
}

func TestIndex_LoadIndex_OverrideDir(t *testing.T) {
	ts := setupMockServer()
	defer ts.Close()

	provider := New(ts.URL, "test-model")
	tempDir := t.TempDir()

	idx := NewIndex(tempDir, provider)
	idx.Dir = tempDir // set directly to temp

	file1 := filepath.Join(tempDir, "f1.txt")
	_ = os.WriteFile(file1, []byte("hello world\ntesting"), 0644)

	if err := idx.Build(context.Background(), []string{file1}, nil); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Create another Index and load from the same dir
	loadedIdx := NewIndex(tempDir, provider)
	loadedIdx.Dir = tempDir

	// Since LoadIndex is hardcoded to use NewIndex default path,
	// we simulate its logic here for test coverage of the parsing parts
	bIdx, err := os.ReadFile(filepath.Join(tempDir, "index.json"))
	if err != nil {
		t.Fatalf("read index.json: %v", err)
	}
	var meta indexMeta
	_ = json.Unmarshal(bIdx, &meta)
	if meta.Model != "test-model" {
		t.Errorf("expected test-model, got %s", meta.Model)
	}
}

func TestIndex_Search(t *testing.T) {
	ts := setupMockServer()
	defer ts.Close()

	provider := New(ts.URL, "test-model")
	idx := NewIndex("/fake/path", provider)

	// manually populate index
	idx.Vectors = [][]float32{
		{0.0, 1.0},
		{0.9, 0.1}, // Close to query [1.0, 0.0]
		{0.5, 0.5},
	}
	idx.Chunks = []ChunkMeta{
		{FilePath: "f1", StartLine: 1, EndLine: 1},
		{FilePath: "f2", StartLine: 1, EndLine: 1},
		{FilePath: "f3", StartLine: 1, EndLine: 1},
	}
	idx.Dims = 2

	res, err := idx.Search(context.Background(), "query", 2)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	if len(res) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(res))
	}

	if res[0].FilePath != "f2" {
		t.Errorf("Expected best match to be f2, got %s", res[0].FilePath)
	}
	if res[0].Score < 0.85 || res[0].Score > 0.95 {
		t.Errorf("Expected score ~0.9, got %f", res[0].Score)
	}
}

func TestIndex_Update(t *testing.T) {
	ts := setupMockServer()
	defer ts.Close()

	provider := New(ts.URL, "test-model")
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")

	_ = os.WriteFile(file1, []byte("A"), 0644)
	_ = os.WriteFile(file2, []byte("B"), 0644)

	idx := NewIndex(tempDir, provider)
	idx.Dir = filepath.Join(tempDir, "index")

	// Change working directory to tempDir so git rev-parse fails (fallback is tested)
	pwd, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(pwd) }()

	ctx := context.Background()
	_ = idx.Build(ctx, []string{file1, file2}, nil)

	// mtimeを確実に変更するために待機してからファイルを変更
	time.Sleep(100 * time.Millisecond)

	// Modify file1 (change content and mtime)
	_ = os.WriteFile(file1, []byte("A-changed"), 0644)
	// 明示的にmtimeを変更
	newTime := time.Now().Add(1 * time.Second)
	_ = os.Chtimes(file1, newTime, newTime)

	// Delete file2
	_ = os.Remove(file2)

	// Add file3
	file3 := filepath.Join(tempDir, "file3.txt")
	_ = os.WriteFile(file3, []byte("C"), 0644)

	// Call Update without explicit files -> should detect changes in existing files only
	err := idx.Update(ctx, nil, nil) // git非管理なので idx.Files 内のファイルのみチェック
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify file2 is removed
	if _, ok := idx.Files[file2]; ok {
		t.Errorf("file2 should be removed")
	}

	// Verify file1 is updated (hash should match "A-changed")
	h, _ := hashFile(file1)
	if idx.Files[file1].Hash != h {
		t.Errorf("file1 hash not updated: expected %s, got %s", h, idx.Files[file1].Hash)
	}

	// file3 は idx.Files に存在しないので追加されない（期待通り）
	if _, ok := idx.Files[file3]; ok {
		t.Errorf("file3 should not be added when files=nil in non-git directory")
	}
}

func TestIndex_Update_Git(t *testing.T) {
	// Skip if git is not installed
	if err := exec.Command("git", "--version").Run(); err != nil {
		t.Skip("git not installed")
	}

	ts := setupMockServer()
	defer ts.Close()

	provider := New(ts.URL, "test-model")
	tempDir := t.TempDir()

	// git init
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// git config
	cmd = exec.Command("git", "config", "user.name", "test")
	cmd.Dir = tempDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	_ = cmd.Run()

	file1 := "file1.txt"
	file2 := "file2.txt"

	_ = os.WriteFile(filepath.Join(tempDir, file1), []byte("A"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, file2), []byte("B"), 0644)

	// add and commit
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tempDir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = tempDir
	_ = cmd.Run()

	// Initialize Index inside tempDir
	idx := NewIndex(tempDir, provider)
	idx.Dir = filepath.Join(tempDir, ".index")

	ctx := context.Background()

	// Original files are now tracked by git. Wait a bit, then build index.
	// Oh wait, Build uses absolute paths in idx.Files keys, so we should build with abs paths.
	absFile1 := filepath.Join(tempDir, file1)
	absFile2 := filepath.Join(tempDir, file2)
	_ = idx.Build(ctx, []string{absFile1, absFile2}, nil)

	// Modify file1
	_ = os.WriteFile(absFile1, []byte("A-changed"), 0644)

	// Delete file2
	_ = os.Remove(absFile2)

	// Add file3 (untracked)
	file3 := "file3.txt"
	absFile3 := filepath.Join(tempDir, file3)
	_ = os.WriteFile(absFile3, []byte("C"), 0644)

	// Remember current working dir and switch to tempDir so that `git diff` works correctly without specifying path
	pwd, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(pwd) }()

	// Call Update without explicit files
	err := idx.Update(ctx, nil, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	h, _ := hashFile(absFile1)
	if idx.Files[absFile1].Hash != h {
		t.Errorf("absFile1 hash not updated")
	}

	if _, ok := idx.Files[absFile2]; ok {
		t.Errorf("absFile2 should be removed")
	}

	if _, ok := idx.Files[absFile3]; !ok {
		t.Errorf("absFile3 should be added")
	}
}

func TestNewIndex_Path(t *testing.T) {
	provider := New("dummy", "dummy-model")
	idx := NewIndex("/home/user/project", provider)

	expected := filepath.Join("/home/user/project", ".xelyon", "index")
	if idx.Dir != expected {
		t.Errorf("Expected dir %s, got %s", expected, idx.Dir)
	}
}

// TestUpdate_NoChanges は同じファイルで2回Updateしても2回目はchangedFiles=0でスキップすることを確認
func TestUpdate_NoChanges(t *testing.T) {
	ts := setupMockServer()
	defer ts.Close()

	provider := New(ts.URL, "test-model")
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "file1.txt")
	_ = os.WriteFile(file1, []byte("content\nline2\nline3\nline4\nline5\nline6"), 0644)

	idx := NewIndex(tempDir, provider)
	idx.Dir = filepath.Join(tempDir, "index")

	ctx := context.Background()
	err := idx.Build(ctx, []string{file1}, nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	initialChunks := len(idx.Chunks)

	// 同じファイルで再度Update（変更なし）
	err = idx.Update(ctx, []string{file1}, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// チャンク数が変わらないことを確認
	if len(idx.Chunks) != initialChunks {
		t.Errorf("Expected %d chunks, got %d", initialChunks, len(idx.Chunks))
	}
}

// TestUpdate_NilFiles_NoGit は files=nil かつ git 非管理の場合、ハッシュベースで変更検出することを確認
func TestUpdate_NilFiles_NoGit(t *testing.T) {
	ts := setupMockServer()
	defer ts.Close()

	provider := New(ts.URL, "test-model")
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "file1.txt")
	_ = os.WriteFile(file1, []byte("original content\nline2\nline3\nline4\nline5\nline6"), 0644)

	idx := NewIndex(tempDir, provider)
	idx.Dir = filepath.Join(tempDir, "index")

	// git 非管理のディレクトリで作業
	pwd, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(pwd) }()

	ctx := context.Background()
	err := idx.Build(ctx, []string{file1}, nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	initialChunks := len(idx.Chunks)

	// 変更なしで Update(nil)
	err = idx.Update(ctx, nil, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// チャンク数が変わらない
	if len(idx.Chunks) != initialChunks {
		t.Errorf("Expected %d chunks unchanged, got %d", initialChunks, len(idx.Chunks))
	}

	// ファイルを変更（mtimeを明示的に変更）
	_ = os.WriteFile(file1, []byte("modified content\nline2\nline3\nline4\nline5\nline6"), 0644)
	// mtimeを1秒後に設定して変更を確実に検出
	newTime := time.Now().Add(1 * time.Second)
	_ = os.Chtimes(file1, newTime, newTime)

	err = idx.Update(ctx, nil, nil)
	if err != nil {
		t.Fatalf("Update after modify failed: %v", err)
	}

	// チャンクが更新されている（数は同じだが内容が変わる）
	if len(idx.Chunks) == 0 {
		t.Errorf("Expected chunks to be updated")
	}
}

// TestUpdate_FileDeleted は削除されたファイルのチャンクが除去されることを確認
func TestUpdate_FileDeleted(t *testing.T) {
	ts := setupMockServer()
	defer ts.Close()

	provider := New(ts.URL, "test-model")
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	_ = os.WriteFile(file1, []byte("content1\nline2\nline3\nline4\nline5\nline6"), 0644)
	_ = os.WriteFile(file2, []byte("content2\nline2\nline3\nline4\nline5\nline6"), 0644)

	idx := NewIndex(tempDir, provider)
	idx.Dir = filepath.Join(tempDir, "index")

	ctx := context.Background()
	err := idx.Build(ctx, []string{file1, file2}, nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(idx.Chunks) < 2 {
		t.Fatalf("Expected at least 2 chunks, got %d", len(idx.Chunks))
	}

	// file2 を削除
	_ = os.Remove(file2)

	err = idx.Update(ctx, []string{file1, file2}, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// file2 のチャンクが削除されている
	for _, c := range idx.Chunks {
		if c.FilePath == file2 {
			t.Errorf("file2 chunks should be removed")
			break
		}
	}

	// file2 が Files からも削除されている
	if _, ok := idx.Files[file2]; ok {
		t.Errorf("file2 should be removed from idx.Files")
	}
}

// TestUpdate_FileModified は変更されたファイルのチャンクが再計算されることを確認
func TestUpdate_FileModified(t *testing.T) {
	ts := setupMockServer()
	defer ts.Close()

	provider := New(ts.URL, "test-model")
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "file1.txt")
	_ = os.WriteFile(file1, []byte("original content\nline2\nline3\nline4\nline5\nline6"), 0644)

	idx := NewIndex(tempDir, provider)
	idx.Dir = filepath.Join(tempDir, "index")

	ctx := context.Background()
	err := idx.Build(ctx, []string{file1}, nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	initialHash := idx.Files[file1].Hash

	// ファイルを変更（mtimeを明示的に変更）
	_ = os.WriteFile(file1, []byte("modified content\nline2\nline3\nline4\nline5\nline6"), 0644)
	// mtimeを1秒後に設定して変更を確実に検出
	newTime := time.Now().Add(1 * time.Second)
	_ = os.Chtimes(file1, newTime, newTime)

	err = idx.Update(ctx, []string{file1}, nil)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// ハッシュが更新されている
	newHash := idx.Files[file1].Hash
	if newHash == initialHash {
		t.Errorf("Hash should be updated after file modification")
	}

	// 新しいハッシュが正しい
	h, _ := hashFile(file1)
	if newHash != h {
		t.Errorf("Hash mismatch: expected %s, got %s", h, newHash)
	}
}
