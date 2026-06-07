package rawoutputs

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/crypto"
)

func (s *Store) appendManifestRecord(record ManifestRecord) error {
	if err := validateSessionID(record.Ref.SessionID); err != nil {
		return err
	}
	path, err := s.ensureSafeSessionFileParent(record.Ref.SessionID, rawOutputManifestRelativePath)
	if err != nil {
		return reasonError(ReasonPathInvalid, "create manifest dir: %w", err)
	}
	line, err := s.encodeManifestLine(record)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return reasonError(ReasonPathInvalid, "open manifest: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return reasonError(ReasonPathInvalid, "append manifest: %w", err)
	}
	return nil
}

func (s *Store) loadLifecycle(sessionID string) (map[string]lifecycleState, error) {
	if err := validateSessionID(sessionID); err != nil {
		return nil, err
	}
	path, err := s.safeExistingSessionFilePath(sessionID, rawOutputManifestRelativePath)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return map[string]lifecycleState{}, nil
	}
	if err != nil {
		return nil, reasonError(ReasonManifestCorrupt, "open manifest: %w", err)
	}
	defer f.Close()

	states := map[string]lifecycleState{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 32*1024*1024)
	for scanner.Scan() {
		record, err := s.decodeManifestLine(scanner.Bytes())
		if err != nil {
			return nil, err
		}
		if record.SchemaVersion != schemaVersion {
			return nil, reasonError(ReasonManifestCorrupt, "unsupported schema version %d", record.SchemaVersion)
		}
		refID := record.Ref.RefID
		if refID == "" {
			return nil, reasonError(ReasonManifestCorrupt, "manifest record missing ref_id")
		}
		state := states[refID]
		switch record.RecordType {
		case recordTypeCreated:
			state.created = record
		case recordTypeQuarantined:
			state.quarantined = true
		case recordTypeTombstoned:
			state.tombstoned = true
		case recordTypeGCCollected:
			state.collected = true
		default:
			return nil, reasonError(ReasonManifestCorrupt, "unknown record type %q", record.RecordType)
		}
		states[refID] = state
	}
	if err := scanner.Err(); err != nil {
		return nil, reasonError(ReasonManifestCorrupt, "scan manifest: %w", err)
	}
	return states, nil
}

func (s *Store) encodeManifestLine(record ManifestRecord) ([]byte, error) {
	record.Source = sanitizeSourceMetadata(record.Source)
	record.Classification = sanitizeClassificationMetadata(record.Classification)
	data, err := json.Marshal(record)
	if err != nil {
		return nil, reasonError(ReasonManifestCorrupt, "marshal manifest: %w", err)
	}
	if !s.opts.EncryptionEnabled {
		return data, nil
	}
	encrypted, err := crypto.EncryptSession(data, s.opts.Passphrase)
	if err != nil {
		return nil, reasonError(ReasonEncryptionRequired, "encrypt manifest: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(encrypted)
	return []byte(encryptedLinePrefix + encoded), nil
}

func (s *Store) decodeManifestLine(line []byte) (ManifestRecord, error) {
	data := append([]byte(nil), line...)
	if s.opts.EncryptionEnabled {
		if !strings.HasPrefix(string(data), encryptedLinePrefix) {
			return ManifestRecord{}, reasonError(ReasonDecryptFailed, "manifest line is not encrypted")
		}
		decoded, err := base64.StdEncoding.DecodeString(string(data[len(encryptedLinePrefix):]))
		if err != nil {
			return ManifestRecord{}, reasonError(ReasonDecryptFailed, "decode manifest line: %w", err)
		}
		decrypted, err := crypto.DecryptSession(decoded, s.opts.Passphrase)
		if err != nil {
			return ManifestRecord{}, reasonError(ReasonDecryptFailed, "decrypt manifest line: %w", err)
		}
		data = decrypted
	}
	var record ManifestRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return ManifestRecord{}, reasonError(ReasonManifestCorrupt, "unmarshal manifest: %w", err)
	}
	return record, nil
}

func (s *Store) writeIndex(sessionID string, records []ManifestRecord) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}
	path, err := s.ensureSafeSessionFileParent(sessionID, rawOutputIndexRelativePath)
	if err != nil {
		return reasonError(ReasonPathInvalid, "create index dir: %w", err)
	}
	type indexRecord struct {
		SchemaVersion int              `json:"schema_version"`
		Records       []ManifestRecord `json:"records"`
	}
	data, err := json.Marshal(indexRecord{SchemaVersion: schemaVersion, Records: records})
	if err != nil {
		return reasonError(ReasonIndexCorrupt, "marshal index: %w", err)
	}
	if s.opts.EncryptionEnabled {
		encrypted, err := crypto.EncryptSession(data, s.opts.Passphrase)
		if err != nil {
			return reasonError(ReasonEncryptionRequired, "encrypt index: %w", err)
		}
		data = []byte(encryptedLinePrefix + base64.StdEncoding.EncodeToString(encrypted))
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "raw_outputs.index-*.tmp")
	if err != nil {
		return reasonError(ReasonPathInvalid, "create index temp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return reasonError(ReasonPathInvalid, "write index temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return reasonError(ReasonPathInvalid, "chmod index temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return reasonError(ReasonPathInvalid, "close index temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return reasonError(ReasonPathInvalid, "commit index: %w", err)
	}
	return nil
}
