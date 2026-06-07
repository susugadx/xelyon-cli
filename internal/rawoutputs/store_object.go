package rawoutputs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/crypto"
)

type tempWriteStats struct {
	sha256Hex string
	byteSize  int64
	runeSize  int
}

func (s *Store) writeTempPlainObject(ctx context.Context, sessionID string, body io.Reader) (string, tempWriteStats, error) {
	tmpDir, err := s.ensureSafeSessionDir(sessionID, rawOutputTmpRelativeDir)
	if err != nil {
		return "", tempWriteStats{}, reasonError(ReasonPathInvalid, "create temp dir: %w", err)
	}
	tmp, err := os.CreateTemp(tmpDir, "rawout-*.tmp")
	if err != nil {
		return "", tempWriteStats{}, reasonError(ReasonPathInvalid, "create temp object: %w", err)
	}
	tempPath := tmp.Name()
	defer tmp.Close()

	hash := sha256.New()
	buf := make([]byte, s.opts.ChunkBytes)
	var total int64
	var runes int
	for {
		if err := ctx.Err(); err != nil {
			_ = os.Remove(tempPath)
			return "", tempWriteStats{}, err
		}
		n, readErr := body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			total += int64(n)
			if total > s.opts.MaxArtifactBytes {
				_ = os.Remove(tempPath)
				return "", tempWriteStats{}, reasonError(ReasonArtifactTooLarge, "artifact %d > max %d", total, s.opts.MaxArtifactBytes)
			}
			runes += utf8.RuneCount(chunk)
			if _, err := hash.Write(chunk); err != nil {
				_ = os.Remove(tempPath)
				return "", tempWriteStats{}, err
			}
			if _, err := tmp.Write(chunk); err != nil {
				_ = os.Remove(tempPath)
				return "", tempWriteStats{}, reasonError(ReasonPathInvalid, "write temp object: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = os.Remove(tempPath)
			return "", tempWriteStats{}, readErr
		}
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = os.Remove(tempPath)
		return "", tempWriteStats{}, reasonError(ReasonPathInvalid, "chmod temp object: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = os.Remove(tempPath)
		return "", tempWriteStats{}, reasonError(ReasonPathInvalid, "sync temp object: %w", err)
	}
	return tempPath, tempWriteStats{sha256Hex: hex.EncodeToString(hash.Sum(nil)), byteSize: total, runeSize: runes}, nil
}

func (s *Store) readPlainObjectToMemory(ctx context.Context, body io.Reader) ([]byte, tempWriteStats, error) {
	hash := sha256.New()
	var buf bytes.Buffer
	chunkBuf := make([]byte, s.opts.ChunkBytes)
	var total int64
	var runes int
	for {
		if err := ctx.Err(); err != nil {
			return nil, tempWriteStats{}, err
		}
		n, readErr := body.Read(chunkBuf)
		if n > 0 {
			chunk := chunkBuf[:n]
			total += int64(n)
			if total > s.opts.MaxArtifactBytes {
				return nil, tempWriteStats{}, reasonError(ReasonArtifactTooLarge, "artifact %d > max %d", total, s.opts.MaxArtifactBytes)
			}
			runes += utf8.RuneCount(chunk)
			if _, err := hash.Write(chunk); err != nil {
				return nil, tempWriteStats{}, err
			}
			if _, err := buf.Write(chunk); err != nil {
				return nil, tempWriteStats{}, err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, tempWriteStats{}, readErr
		}
	}
	return buf.Bytes(), tempWriteStats{sha256Hex: hex.EncodeToString(hash.Sum(nil)), byteSize: total, runeSize: runes}, nil
}

func (s *Store) commitObject(sessionID, tempPath string, artifact RawOutputArtifact) error {
	if artifact.Encrypted {
		return reasonError(ReasonEncryptionRequired, "encrypted artifact must use encrypted commit path")
	}
	finalPath, err := s.ensureSafeSessionFileParent(sessionID, artifact.RelativePath)
	if err != nil {
		return reasonError(ReasonPathInvalid, "create object dir: %w", err)
	}
	if _, err := os.Lstat(finalPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return reasonError(ReasonPathInvalid, "stat object: %w", err)
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return reasonError(ReasonPathInvalid, "commit object: %w", err)
	}
	return os.Chmod(finalPath, 0o600)
}

func (s *Store) commitEncryptedObject(sessionID string, plain []byte, artifact RawOutputArtifact) error {
	finalPath, err := s.ensureSafeSessionFileParent(sessionID, artifact.RelativePath)
	if err != nil {
		return reasonError(ReasonPathInvalid, "create object dir: %w", err)
	}
	if _, err := os.Lstat(finalPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return reasonError(ReasonPathInvalid, "stat object: %w", err)
	}
	encrypted, err := crypto.EncryptSession(plain, s.opts.Passphrase)
	if err != nil {
		return reasonError(ReasonEncryptionRequired, "encrypt object: %w", err)
	}
	tmpEncrypted, err := os.CreateTemp(filepath.Dir(finalPath), "rawout-enc-*.tmp")
	if err != nil {
		return reasonError(ReasonPathInvalid, "create encrypted temp object: %w", err)
	}
	encryptedPath := tmpEncrypted.Name()
	if _, err := tmpEncrypted.Write(encrypted); err != nil {
		_ = tmpEncrypted.Close()
		_ = os.Remove(encryptedPath)
		return reasonError(ReasonPathInvalid, "write encrypted object: %w", err)
	}
	if err := tmpEncrypted.Chmod(0o600); err != nil {
		_ = tmpEncrypted.Close()
		_ = os.Remove(encryptedPath)
		return reasonError(ReasonPathInvalid, "chmod encrypted object: %w", err)
	}
	if err := tmpEncrypted.Sync(); err != nil {
		_ = tmpEncrypted.Close()
		_ = os.Remove(encryptedPath)
		return reasonError(ReasonPathInvalid, "sync encrypted object: %w", err)
	}
	if err := tmpEncrypted.Close(); err != nil {
		_ = os.Remove(encryptedPath)
		return reasonError(ReasonPathInvalid, "close encrypted object: %w", err)
	}
	if err := os.Rename(encryptedPath, finalPath); err != nil {
		_ = os.Remove(encryptedPath)
		return reasonError(ReasonPathInvalid, "commit encrypted object: %w", err)
	}
	return os.Chmod(finalPath, 0o600)
}

func (s *Store) readAndVerifyObject(record ManifestRecord) ([]byte, error) {
	path, err := s.safeExistingSessionFilePath(record.Ref.SessionID, record.Artifact.RelativePath)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, reasonError(ReasonArtifactMissing, "object missing")
		}
		return nil, reasonError(ReasonPathInvalid, "stat object: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, reasonError(ReasonPathInvalid, "object path is symlink")
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		return nil, reasonError(ReasonPathInvalid, "read object: %w", err)
	}
	body := stored
	if record.Artifact.Encrypted {
		decrypted, err := crypto.DecryptSession(stored, s.opts.Passphrase)
		if err != nil {
			return nil, reasonError(ReasonDecryptFailed, "decrypt object: %w", err)
		}
		body = decrypted
	}
	hash := sha256.Sum256(body)
	got := "sha256:" + hex.EncodeToString(hash[:])
	if got != record.Artifact.ContentHash || got != record.Ref.ContentHash {
		return nil, reasonError(ReasonArtifactHashMismatch, "hash mismatch got %s want %s", got, record.Artifact.ContentHash)
	}
	return body, nil
}

func buildRefID(req CreateRequest, contentHash string) string {
	hash := sha256.Sum256([]byte(strings.Join([]string{
		req.SessionID,
		string(req.Surface),
		req.Source.EventID,
		req.Source.ToolCallID,
		req.Source.ToolName,
		req.Source.CommandHash,
		contentHash,
	}, "\x00")))
	return refIDPrefix + hex.EncodeToString(hash[:])[:12]
}

func objectRelativePath(hash string) string {
	return filepath.ToSlash(filepath.Join("objects", "sha256", hash[:2], hash[2:4], hash+".raw"))
}

func approxTokensFromBytes(bytes int64) int {
	if bytes <= 0 {
		return 0
	}
	tokens := int(bytes / 4)
	if tokens == 0 {
		return 1
	}
	return tokens
}
