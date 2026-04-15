package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/crypto"
)

const maxSessionHistoryLineBytes = 16 * 1024 * 1024

// Load はセッションファイルから読み込み
func (st *Storage) Load(sessionID string) (*Session, error) {
	meta, err := st.loadMetadata(sessionID)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:                              meta.ID,
		Model:                           meta.Model,
		StartTime:                       meta.StartTime,
		LastModified:                    meta.LastModified,
		Messages:                        []MessageEntry{},
		PendingApprovedPlan:             meta.PendingApprovedPlan,
		PendingApprovedPlanHasChanges:   meta.PendingApprovedPlanHasChanges,
		PendingApprovedPlanChangedFiles: append([]string(nil), meta.PendingApprovedPlanChangedFiles...),
	}

	filePath := st.sessionPath(sessionID)
	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			if canLoadWithoutHistoryFile(meta) {
				session.markPersisted()
				return session, nil
			}
			return nil, fmt.Errorf("session history file missing for %s (metadata expects %d messages)", sessionID, meta.MessageCount)
		}
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer f.Close()

	messages, err := st.loadSessionMessages(f)
	if err != nil {
		return nil, err
	}
	session.Messages = messages
	if err := validateLoadedConversationIntegrity(meta, session); err != nil {
		return nil, err
	}
	session.markPersisted()

	return session, nil
}

func (st *Storage) loadSessionMessages(f *os.File) ([]MessageEntry, error) {
	var messages []MessageEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSessionHistoryLineBytes)
	for scanner.Scan() {
		data := append([]byte(nil), scanner.Bytes()...)
		if len(data) == 0 {
			continue
		}

		msg, ok := st.decodeHistoryLine(data)
		if !ok {
			continue
		}
		messages = append(messages, msg)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan session file: %w", err)
	}
	return messages, nil
}

// decodeHistoryLine は 1 行単位の best-effort 復元を担当する。
// 行レベルで回復不能でもここでは失敗にせず、session 全体として致命的かどうかは
// validateLoadedConversationIntegrity に委ねる。
func (st *Storage) decodeHistoryLine(data []byte) (MessageEntry, bool) {
	if st.encryption && len(data) > 0 {
		if decrypted, err := crypto.DecryptSession(data, st.passphrase); err == nil {
			if msg, ok := unmarshalHistoryMessage(decrypted); ok {
				return msg, true
			}
			return MessageEntry{}, false
		}

		// 暗号化有効化前の平文 JSONL も引き続き読めるようにする。
		if msg, ok := unmarshalHistoryMessage(data); ok {
			return msg, true
		}
		return MessageEntry{}, false
	}

	return unmarshalHistoryMessage(data)
}

func unmarshalHistoryMessage(data []byte) (MessageEntry, bool) {
	var msg MessageEntry
	if err := json.Unmarshal(data, &msg); err != nil {
		return MessageEntry{}, false
	}
	return msg, true
}

func canLoadWithoutHistoryFile(meta *SessionMetadata) bool {
	if meta == nil {
		return false
	}
	return meta.MessageCount == 0
}

// validateLoadedConversationIntegrity は session 全体としてロードを許容できるかを判定する。
// 行単位の decode failure は decodeHistoryLine 側で吸収し、ここでは最終的に復元できた
// 会話メッセージ数が metadata と一致するかだけを責務にする。
func validateLoadedConversationIntegrity(meta *SessionMetadata, session *Session) error {
	if meta == nil || session == nil {
		return nil
	}

	loadedCount := session.conversationMessageCount()
	if loadedCount == meta.MessageCount {
		return nil
	}
	return fmt.Errorf("session history is inconsistent for %s (metadata expects %d messages, loaded %d)", meta.ID, meta.MessageCount, loadedCount)
}
