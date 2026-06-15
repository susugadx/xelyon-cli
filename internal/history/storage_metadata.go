package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const metadataPreviewMaxRunes = 80

// saveMetadata はセッションメタデータを保存
func (st *Storage) saveMetadata(session *Session) error {
	if session == nil {
		return nil
	}

	metaDir := filepath.Join(st.baseDir, "metadata")
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata dir: %w", err)
	}

	meta := buildSessionMetadata(session)
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	path := st.metadataPath(session.ID)
	if err := replaceMetadataFile(path, data); err != nil {
		return err
	}

	return nil
}

func buildSessionMetadata(session *Session) SessionMetadata {
	return SessionMetadata{
		ID:                        session.ID,
		Model:                     session.Model,
		ProviderName:              session.ProviderName,
		ProviderConfigKey:         session.ProviderConfigKey,
		WorkingDir:                session.WorkingDir,
		StartTime:                 session.StartTime,
		LastModified:              session.LastModified,
		MessageCount:              session.conversationMessageCount(),
		CompactedItemCount:        session.compactedItemCount(),
		IsCompactedMode:           session.compactedItemCount() > 0,
		Preview:                   extractSessionPreview(session.Messages, metadataPreviewMaxRunes),
		ResponseID:                session.ResponseID,
		ResponseContextVersion:    responseContextMetadataVersionForSession(session),
		ResponseModel:             session.ResponseModel,
		ResponseProviderName:      session.ResponseProviderName,
		ResponseProviderConfigKey: session.ResponseProviderConfigKey,
	}
}

func extractSessionPreview(messages []MessageEntry, maxRunes int) string {
	for _, msg := range messages {
		if msg.Role == "user" {
			return TruncateWithEllipsis(msg.Content, maxRunes)
		}
	}
	return ""
}

func replaceMetadataFile(filePath string, data []byte) error {
	return replaceFileAtomically(filePath, func(tmp *os.File) error {
		if _, err := tmp.Write(data); err != nil {
			return fmt.Errorf("failed to write metadata: %w", err)
		}
		return nil
	})
}
