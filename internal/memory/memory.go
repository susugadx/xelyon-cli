package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

const (
	globalMemoryDir  = ".xelyon"
	projectMemoryDir = ".xelyon"
	memoryFile       = "memory.json"
)

// Memory はユーザーの記憶エントリ
type Memory struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Project   string    `json:"project,omitempty"` // プロジェクト名（プロジェクト別メモリの場合）
}

// MemoryStore は記憶のストレージ
type MemoryStore struct {
	GlobalPath  string
	ProjectPath string
	memories    []Memory
}

// NewMemoryStore は新しいMemoryStoreを作成
func NewMemoryStore() (*MemoryStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	globalPath := filepath.Join(home, globalMemoryDir, memoryFile)

	// プロジェクト別メモリパスを探す
	projectPath := ""
	cwd, err := os.Getwd()
	if err == nil {
		// カレントディレクトリから上位に向かって.xelyon/を探す
		dir := cwd
		for {
			candidatePath := filepath.Join(dir, projectMemoryDir, memoryFile)
			parentDir := filepath.Dir(candidatePath)
			if info, err := os.Stat(parentDir); err == nil && info.IsDir() {
				projectPath = candidatePath
				break
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	store := &MemoryStore{
		GlobalPath:  globalPath,
		ProjectPath: projectPath,
		memories:    []Memory{},
	}

	// メモリを読み込み
	if err := store.Load(); err != nil {
		// ファイルが存在しない場合はエラーにしない
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return store, nil
}

// Load はメモリを読み込み（プロジェクト別 → グローバルの順）
func (ms *MemoryStore) Load() error {
	ms.memories = []Memory{}

	// プロジェクト別メモリを優先
	if ms.ProjectPath != "" {
		if err := ms.loadFromFile(ms.ProjectPath, true); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	// グローバルメモリを読み込み
	if err := ms.loadFromFile(ms.GlobalPath, false); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// loadFromFile はファイルからメモリを読み込み
func (ms *MemoryStore) loadFromFile(path string, isProject bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var memories []Memory
	if err := json.Unmarshal(data, &memories); err != nil {
		return fmt.Errorf("failed to parse memory file %s: %w", path, err)
	}

	// プロジェクト別メモリの場合、Projectフィールドを設定
	if isProject {
		projectName := filepath.Base(filepath.Dir(filepath.Dir(path)))
		for i := range memories {
			memories[i].Project = projectName
		}
	}

	ms.memories = append(ms.memories, memories...)
	return nil
}

// Save はメモリを保存
func (ms *MemoryStore) Save() error {
	// プロジェクト別とグローバルを分離
	var globalMemories []Memory
	var projectMemories []Memory

	for _, m := range ms.memories {
		if m.Project != "" {
			projectMemories = append(projectMemories, m)
		} else {
			globalMemories = append(globalMemories, m)
		}
	}

	// グローバルメモリを保存
	if err := ms.saveToFile(ms.GlobalPath, globalMemories); err != nil {
		return err
	}

	// プロジェクト別メモリを保存
	if ms.ProjectPath != "" && len(projectMemories) > 0 {
		if err := ms.saveToFile(ms.ProjectPath, projectMemories); err != nil {
			return err
		}
	}

	return nil
}

// saveToFile はファイルにメモリを保存
func (ms *MemoryStore) saveToFile(path string, memories []Memory) error {
	// ディレクトリ作成
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// JSON化
	data, err := json.MarshalIndent(memories, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal memories: %w", err)
	}

	// ファイル書き込み
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write memory file: %w", err)
	}

	return nil
}

// Add は新しい記憶を追加
func (ms *MemoryStore) Add(content string, isProject bool) (*Memory, error) {
	memory := Memory{
		ID:        uuid.New().String()[:8],
		Content:   content,
		CreatedAt: time.Now(),
	}

	// プロジェクト別メモリの場合
	if isProject && ms.ProjectPath != "" {
		projectName := filepath.Base(filepath.Dir(filepath.Dir(ms.ProjectPath)))
		memory.Project = projectName
	}

	ms.memories = append(ms.memories, memory)

	if err := ms.Save(); err != nil {
		return nil, err
	}

	return &memory, nil
}

// List は全メモリを返す
func (ms *MemoryStore) List() []Memory {
	return ms.memories
}

// Delete は指定IDのメモリを削除
func (ms *MemoryStore) Delete(id string) error {
	found := false
	newMemories := []Memory{}

	for _, m := range ms.memories {
		if m.ID == id {
			found = true
			continue
		}
		newMemories = append(newMemories, m)
	}

	if !found {
		return fmt.Errorf("memory not found: %s", id)
	}

	ms.memories = newMemories
	return ms.Save()
}

// Clear は全メモリを削除
func (ms *MemoryStore) Clear(projectOnly bool) error {
	if projectOnly {
		// プロジェクト別メモリのみ削除
		newMemories := []Memory{}
		for _, m := range ms.memories {
			if m.Project == "" {
				newMemories = append(newMemories, m)
			}
		}
		ms.memories = newMemories
	} else {
		// 全メモリ削除
		ms.memories = []Memory{}
	}

	return ms.Save()
}

// GetMemoriesAsText はシステムプロンプト用のテキストを生成
func (ms *MemoryStore) GetMemoriesAsText() string {
	if len(ms.memories) == 0 {
		return ""
	}

	text := "\n【ユーザーの記憶】\n"
	for _, m := range ms.memories {
		prefix := ""
		if m.Project != "" {
			prefix = fmt.Sprintf("[Project: %s] ", m.Project)
		}
		text += fmt.Sprintf("- %s%s\n", prefix, m.Content)
	}
	text += "\n"

	return text
}
