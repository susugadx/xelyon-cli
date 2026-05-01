package history

import (
	"bufio"
	"errors"
	"fmt"
	"os"
)

// Load はセッションファイルから読み込み
func (st *Storage) Load(sessionID string) (*Session, error) {
	if err := validateSessionID(sessionID); err != nil {
		return nil, fmt.Errorf("invalid session ID %q: %w", sessionID, err)
	}

	meta, err := st.loadMetadata(sessionID)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:                        meta.ID,
		Model:                     meta.Model,
		ProviderName:              meta.ProviderName,
		ProviderConfigKey:         meta.ProviderConfigKey,
		StartTime:                 meta.StartTime,
		LastModified:              meta.LastModified,
		Messages:                  []MessageEntry{},
		ResponseID:                meta.ResponseID,
		ResponseModel:             meta.ResponseModel,
		ResponseProviderName:      meta.ResponseProviderName,
		ResponseProviderConfigKey: meta.ResponseProviderConfigKey,
	}
	restoreLoadedResponseContext(meta, session)

	filePath := st.sessionPath(sessionID)
	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			if canLoadWithoutHistoryFile(meta) {
				session.markPersisted()
				return session, nil
			}
			return nil, fmt.Errorf("failed to open session file: session history file missing for %s (metadata expects %d messages%s)", sessionID, meta.MessageCount, compactedStateExpectation(meta))
		}
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer f.Close()

	messages, err := st.loadSessionMessages(f)
	if err != nil {
		return nil, err
	}
	session.Messages = restoreLoadedSessionStateEntries(session, messages)
	if err := validateLoadedConversationIntegrity(meta, session); err != nil {
		return nil, err
	}
	session.markPersisted()

	return session, nil
}

func (st *Storage) loadSessionMessages(f *os.File) ([]MessageEntry, error) {
	var messages []MessageEntry
	scanner := bufio.NewScanner(f)
	// Scanner は token と区切り文字を同じ buffer に読むため、保存上限より 1 byte だけ余裕を持たせる。
	scanner.Buffer(make([]byte, 0, 64*1024), maxSessionHistoryStoredLineBytes+1)
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
		if errors.Is(err, bufio.ErrTooLong) {
			return nil, fmt.Errorf("failed to scan session file: history line exceeds stored line limit (%d bytes): %w", maxSessionHistoryStoredLineBytes, err)
		}
		return nil, fmt.Errorf("failed to scan session file: %w", err)
	}
	return messages, nil
}

func restoreLoadedSessionStateEntries(session *Session, messages []MessageEntry) []MessageEntry {
	if session == nil {
		return messages
	}

	conversation := make([]MessageEntry, 0, len(messages))
	for _, msg := range messages {
		if msg.EntryType == compactedStateEntryType {
			session.CompactedItems = cloneCompactedItems(msg.CompactedItems)
			session.IsCompactedMode = msg.IsCompactedMode && len(session.CompactedItems) > 0
			continue
		}
		conversation = append(conversation, msg)
	}
	return conversation
}

func canLoadWithoutHistoryFile(meta *SessionMetadata) bool {
	if meta == nil {
		return false
	}
	return meta.MessageCount == 0 && !metadataExpectsCompactedState(meta)
}

// validateLoadedConversationIntegrity は session 全体としてロードを許容できるかを判定する。
// 行単位の decode failure は decodeHistoryLine 側で吸収し、ここでは最終的に復元できた
// 会話メッセージ数と compacted state が metadata と一致するかを責務にする。
func validateLoadedConversationIntegrity(meta *SessionMetadata, session *Session) error {
	if meta == nil || session == nil {
		return nil
	}

	loadedCount := session.conversationMessageCount()
	if loadedCount != meta.MessageCount {
		return fmt.Errorf("session history is inconsistent for %s (metadata expects %d messages, loaded %d)", meta.ID, meta.MessageCount, loadedCount)
	}

	if !metadataExpectsCompactedState(meta) {
		return nil
	}
	loadedCompactedCount := session.compactedItemCount()
	if session.IsCompactedMode && loadedCompactedCount == meta.CompactedItemCount {
		return nil
	}
	return fmt.Errorf("session history is inconsistent for %s (metadata expects %d compacted items, loaded %d)", meta.ID, meta.CompactedItemCount, loadedCompactedCount)
}

func metadataExpectsCompactedState(meta *SessionMetadata) bool {
	return meta != nil && (meta.IsCompactedMode || meta.CompactedItemCount > 0)
}

func compactedStateExpectation(meta *SessionMetadata) string {
	if !metadataExpectsCompactedState(meta) {
		return ""
	}
	return fmt.Sprintf(", %d compacted items", meta.CompactedItemCount)
}
