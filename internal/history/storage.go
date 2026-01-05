package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultHistoryDir = ".xelyon/history"
)

// Storage は履歴の永続化を管理
type Storage struct {
	baseDir string
}

// NewStorage はストレージインスタンスを作成
func NewStorage() (*Storage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}

	historyDir := filepath.Join(home, defaultHistoryDir)
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create history dir: %w", err)
	}

	return &Storage{baseDir: historyDir}, nil
}

// Save はメッセージをJSONLファイルに追記
func (st *Storage) Save(session *Session) error {
	if len(session.Messages) == 0 {
		return nil
	}

	filePath := st.sessionPath(session.ID)

	// 追記モードでオープン
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	// 最後のメッセージをJSONLとして書き込み
	lastMsg := session.Messages[len(session.Messages)-1]
	data, err := json.Marshal(lastMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	// メタデータを更新
	return st.saveMetadata(session)
}

// Load はセッションファイルから読み込み
func (st *Storage) Load(sessionID string) (*Session, error) {
	// メタデータを読み込み
	meta, err := st.loadMetadata(sessionID)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:           meta.ID,
		Model:        meta.Model,
		StartTime:    meta.StartTime,
		LastModified: meta.LastModified,
		Messages:     []MessageEntry{},
	}

	// JSONLメッセージを読み込み
	filePath := st.sessionPath(sessionID)
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var msg MessageEntry
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			// 不正な行はスキップ
			continue
		}
		session.Messages = append(session.Messages, msg)
	}

	return session, nil
}

// ListSessions は全セッションを新しい順で返す
func (st *Storage) ListSessions() ([]SessionMetadata, error) {
	metaDir := filepath.Join(st.baseDir, "metadata")
	if _, err := os.Stat(metaDir); os.IsNotExist(err) {
		return []SessionMetadata{}, nil
	}

	entries, err := os.ReadDir(metaDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata dir: %w", err)
	}

	var sessions []SessionMetadata
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		sessionID := strings.TrimSuffix(entry.Name(), ".json")
		meta, err := st.loadMetadata(sessionID)
		if err != nil {
			// 破損したメタデータはスキップ
			continue
		}
		sessions = append(sessions, *meta)
	}

	// last_modifiedでソート（新しい順）
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastModified.After(sessions[j].LastModified)
	})

	return sessions, nil
}

// GetLastSession は最新のセッションIDを返す
func (st *Storage) GetLastSession() (string, error) {
	sessions, err := st.ListSessions()
	if err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		return "", fmt.Errorf("no sessions found")
	}
	return sessions[0].ID, nil
}

// sessionPath はセッションJSONLファイルのパスを返す
func (st *Storage) sessionPath(sessionID string) string {
	return filepath.Join(st.baseDir, sessionID+".jsonl")
}

// metadataPath はメタデータJSONファイルのパスを返す
func (st *Storage) metadataPath(sessionID string) string {
	return filepath.Join(st.baseDir, "metadata", sessionID+".json")
}

// saveMetadata はセッションメタデータを保存
func (st *Storage) saveMetadata(session *Session) error {
	metaDir := filepath.Join(st.baseDir, "metadata")
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata dir: %w", err)
	}

	// 最初のユーザーメッセージをプレビューに使用
	preview := ""
	for _, msg := range session.Messages {
		if msg.Role == "user" {
			preview = msg.Content
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			break
		}
	}

	meta := SessionMetadata{
		ID:           session.ID,
		Model:        session.Model,
		StartTime:    session.StartTime,
		LastModified: session.LastModified,
		MessageCount: len(session.Messages),
		Preview:      preview,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	path := st.metadataPath(session.ID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// loadMetadata はセッションメタデータを読み込み
func (st *Storage) loadMetadata(sessionID string) (*SessionMetadata, error) {
	path := st.metadataPath(sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var meta SessionMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &meta, nil
}
