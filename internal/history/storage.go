package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/crypto"
)

const (
	defaultHistoryDir = ".xelyon/history"
)

var (
	encryptSessionForStorage = crypto.EncryptSession
	decryptSessionForStorage = crypto.DecryptSession
	userHomeDirForStorage    = os.UserHomeDir
	getPassphraseForStorage  = crypto.GetOrCreatePassphrase
)

// Storage は履歴の永続化を管理
type Storage struct {
	baseDir    string
	encryption bool   // 暗号化有効フラグ
	passphrase string // 暗号化キー
}

// NewStorage はストレージインスタンスを作成
func NewStorage() (*Storage, error) {
	home, err := userHomeDirForStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}

	historyDir := filepath.Join(home, defaultHistoryDir)
	if err := os.MkdirAll(historyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create history dir: %w", err)
	}

	// 暗号化設定（環境変数: XELYON_ENCRYPT_HISTORY=1 で有効化）
	encryptionEnabled := os.Getenv("XELYON_ENCRYPT_HISTORY") == "1"
	var passphrase string
	if encryptionEnabled {
		passphrase, err = getPassphraseForStorage()
		if err != nil {
			return nil, fmt.Errorf("failed to get encryption key: %w", err)
		}
	}

	return &Storage{
		baseDir:    historyDir,
		encryption: encryptionEnabled,
		passphrase: passphrase,
	}, nil
}

// Save は未保存のメッセージをJSONLファイルに追記
func (st *Storage) Save(session *Session) error {
	if session == nil {
		return nil
	}
	unsaved := session.unsavedMessages()
	if len(unsaved) == 0 {
		return st.saveMetadata(session)
	}

	filePath := st.sessionPath(session.ID)

	// 追記モードでオープン
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	for _, msg := range unsaved {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		// 暗号化が有効な場合は暗号化
		if st.encryption {
			encrypted, err := encryptSessionForStorage(data, st.passphrase)
			if err != nil {
				return fmt.Errorf("failed to encrypt message: %w", err)
			}
			data = encrypted
		}

		if _, err := f.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}
	}
	session.markPersisted()

	// メタデータを更新
	return st.saveMetadata(session)
}

// Rewrite はセッション全体をJSONLファイルに再書き込みする。
func (st *Storage) Rewrite(session *Session) error {
	filePath := st.sessionPath(session.ID)

	if len(session.Messages) == 0 {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove history file: %w", err)
		}
		return st.saveMetadata(session)
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to rewrite history file: %w", err)
	}
	defer f.Close()

	for _, msg := range session.Messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		if st.encryption {
			encrypted, err := encryptSessionForStorage(data, st.passphrase)
			if err != nil {
				return fmt.Errorf("failed to encrypt message: %w", err)
			}
			data = encrypted
		}

		if _, err := f.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("failed to rewrite message: %w", err)
		}
	}
	session.markPersisted()

	return st.saveMetadata(session)
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
			if utf8.RuneCountInString(preview) > 80 {
				preview = truncateRunes(preview, 80) + "..."
			}
			break
		}
	}

	meta := SessionMetadata{
		ID:                        session.ID,
		Model:                     session.Model,
		ProviderName:              session.ProviderName,
		ProviderConfigKey:         session.ProviderConfigKey,
		StartTime:                 session.StartTime,
		LastModified:              session.LastModified,
		MessageCount:              session.conversationMessageCount(),
		Preview:                   preview,
		ResponseID:                session.ResponseID,
		ResponseContextVersion:    responseContextMetadataVersionForSession(session),
		ResponseModel:             session.ResponseModel,
		ResponseProviderName:      session.ResponseProviderName,
		ResponseProviderConfigKey: session.ResponseProviderConfigKey,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	path := st.metadataPath(session.ID)
	if err := os.WriteFile(path, data, 0600); err != nil {
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
