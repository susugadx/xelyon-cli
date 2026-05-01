package history

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func (st *Storage) encodeHistoryLine(msg MessageEntry) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}
	if err := validateHistoryPlainLineSize(data); err != nil {
		return nil, err
	}

	if !st.encryption {
		if err := validateHistoryStoredLineSize(data); err != nil {
			return nil, err
		}
		return data, nil
	}

	encrypted, err := encryptSessionForStorage(data, st.passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt message: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(encrypted)
	line := []byte(encryptedHistoryLinePrefix + encoded)
	if err := validateHistoryStoredLineSize(line); err != nil {
		return nil, err
	}
	return line, nil
}

func (st *Storage) encodeHistoryLines(entries []MessageEntry) ([][]byte, error) {
	lines := make([][]byte, 0, len(entries))
	for _, msg := range entries {
		data, err := st.encodeHistoryLine(msg)
		if err != nil {
			return nil, err
		}
		lines = append(lines, data)
	}
	return lines, nil
}

// decodeHistoryLine は 1 行単位の best-effort 復元を担当する。
// 行レベルで回復不能でもここでは失敗にせず、session 全体として致命的かどうかは
// validateLoadedConversationIntegrity に委ねる。
func (st *Storage) decodeHistoryLine(data []byte) (MessageEntry, bool) {
	if st.encryption && len(data) > 0 {
		if hasEncryptedHistoryLinePrefix(data) {
			encrypted, ok := decodeEncryptedHistoryLine(data)
			if !ok {
				return MessageEntry{}, false
			}
			if decrypted, err := decryptSessionForStorage(encrypted, st.passphrase); err == nil {
				if msg, ok := unmarshalHistoryMessage(decrypted); ok {
					return msg, true
				}
				return MessageEntry{}, false
			}
			return MessageEntry{}, false
		}

		// enc: prefix 導入前の raw encrypted line も、1 行として残っていれば読めるようにする。
		if decrypted, err := decryptSessionForStorage(data, st.passphrase); err == nil {
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

func decodeEncryptedHistoryLine(data []byte) ([]byte, bool) {
	if !hasEncryptedHistoryLinePrefix(data) {
		return data, true
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data[encryptedHistoryLinePrefixLen:]))
	if err != nil {
		return nil, false
	}
	return decoded, true
}

func hasEncryptedHistoryLinePrefix(data []byte) bool {
	return bytes.HasPrefix(data, []byte(encryptedHistoryLinePrefix))
}

func unmarshalHistoryMessage(data []byte) (MessageEntry, bool) {
	var msg MessageEntry
	if err := json.Unmarshal(data, &msg); err != nil {
		return MessageEntry{}, false
	}
	return msg, true
}

func validateHistoryPlainLineSize(data []byte) error {
	if len(data) <= maxSessionHistoryPlainLineBytes {
		return nil
	}
	return fmt.Errorf("session history plaintext line exceeds limit (%d > %d bytes)", len(data), maxSessionHistoryPlainLineBytes)
}

func validateHistoryStoredLineSize(data []byte) error {
	if len(data) <= maxSessionHistoryStoredLineBytes {
		return nil
	}
	return fmt.Errorf("session history stored line exceeds limit (%d > %d bytes)", len(data), maxSessionHistoryStoredLineBytes)
}
