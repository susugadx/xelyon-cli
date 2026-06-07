package rawoutputs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"hash"
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

type artifactContentScanner struct {
	hash     hash.Hash
	detector sensitiveStreamDetector
	maxBytes int64
	total    int64
	runes    int
}

func newArtifactContentScanner(maxBytes int64) *artifactContentScanner {
	return &artifactContentScanner{
		hash:     sha256.New(),
		maxBytes: maxBytes,
	}
}

func (s *artifactContentScanner) Scan(chunk []byte) error {
	if len(chunk) == 0 {
		return nil
	}
	s.total += int64(len(chunk))
	if s.total > s.maxBytes {
		return reasonError(ReasonArtifactTooLarge, "artifact %d > max %d", s.total, s.maxBytes)
	}
	s.runes += utf8.RuneCount(chunk)
	if s.detector.Write(chunk) {
		return reasonError(ReasonSensitiveArtifactForbidden, "sensitive output is not stored in normal raw output artifact store")
	}
	if _, err := s.hash.Write(chunk); err != nil {
		return err
	}
	return nil
}

func (s *artifactContentScanner) Stats() tempWriteStats {
	return tempWriteStats{
		sha256Hex: hex.EncodeToString(s.hash.Sum(nil)),
		byteSize:  s.total,
		runeSize:  s.runes,
	}
}

type preparedObjectWriter struct {
	file      *os.File
	tempPath  string
	encrypted io.WriteCloser
}

func (s *Store) writeTempPreparedObject(ctx context.Context, sessionID string, body io.Reader) (string, tempWriteStats, error) {
	scanner := newArtifactContentScanner(s.opts.MaxArtifactBytes)
	buf := make([]byte, s.opts.ChunkBytes)
	n, firstReadErr := body.Read(buf)
	firstChunk := append([]byte(nil), buf[:n]...)
	if n > 0 {
		if err := scanner.Scan(firstChunk); err != nil {
			return "", tempWriteStats{}, err
		}
	}
	if firstReadErr != nil && firstReadErr != io.EOF {
		return "", tempWriteStats{}, firstReadErr
	}

	writer, err := s.newPreparedObjectWriter(sessionID)
	if err != nil {
		return "", tempWriteStats{}, err
	}
	keepTemp := false
	defer func() {
		if !keepTemp {
			writer.abort()
		}
	}()
	if n > 0 {
		if err := writer.write(firstChunk); err != nil {
			return "", tempWriteStats{}, reasonError(ReasonPathInvalid, "write temp object: %w", err)
		}
	}
	if firstReadErr == io.EOF {
		if err := writer.closePayload(); err != nil {
			return "", tempWriteStats{}, err
		}
		keepTemp = true
		return s.finalizeTempObject(writer.file, writer.tempPath, scanner.Stats())
	}
	for {
		if err := ctx.Err(); err != nil {
			return "", tempWriteStats{}, err
		}
		n, readErr := body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if err := scanner.Scan(chunk); err != nil {
				return "", tempWriteStats{}, err
			}
			if err := writer.write(chunk); err != nil {
				return "", tempWriteStats{}, reasonError(ReasonPathInvalid, "write temp object: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", tempWriteStats{}, readErr
		}
	}
	if err := writer.closePayload(); err != nil {
		return "", tempWriteStats{}, err
	}
	keepTemp = true
	return s.finalizeTempObject(writer.file, writer.tempPath, scanner.Stats())
}

func (s *Store) newPreparedObjectWriter(sessionID string) (*preparedObjectWriter, error) {
	tmpDir, err := s.ensureSafeSessionDir(sessionID, rawOutputTmpRelativeDir)
	if err != nil {
		return nil, reasonError(ReasonPathInvalid, "create temp dir: %w", err)
	}
	tmp, err := os.CreateTemp(tmpDir, "rawout-*.tmp")
	if err != nil {
		return nil, reasonError(ReasonPathInvalid, "create temp object: %w", err)
	}
	writer := &preparedObjectWriter{
		file:     tmp,
		tempPath: tmp.Name(),
	}
	if s.opts.EncryptionEnabled {
		encryptedWriter, err := crypto.NewSessionStreamEncryptWriter(tmp, s.opts.Passphrase)
		if err != nil {
			writer.abort()
			return nil, reasonError(ReasonEncryptionRequired, "create encrypted stream: %w", err)
		}
		writer.encrypted = encryptedWriter
	}
	return writer, nil
}

func (w *preparedObjectWriter) write(chunk []byte) error {
	dst := io.Writer(w.file)
	if w.encrypted != nil {
		dst = w.encrypted
	}
	_, err := dst.Write(chunk)
	return err
}

func (w *preparedObjectWriter) closePayload() error {
	if w.encrypted == nil {
		return nil
	}
	if err := w.encrypted.Close(); err != nil {
		return reasonError(ReasonEncryptionRequired, "close encrypted stream: %w", err)
	}
	w.encrypted = nil
	return nil
}

func (w *preparedObjectWriter) abort() {
	if w == nil {
		return
	}
	if w.encrypted != nil {
		_ = w.encrypted.Close()
		w.encrypted = nil
	}
	if w.file != nil {
		_ = w.file.Close()
	}
	if w.tempPath != "" {
		_ = os.Remove(w.tempPath)
	}
}

func (s *Store) finalizeTempObject(tmp *os.File, tempPath string, stats tempWriteStats) (string, tempWriteStats, error) {
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tempPath)
		return "", tempWriteStats{}, reasonError(ReasonPathInvalid, "chmod temp object: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tempPath)
		return "", tempWriteStats{}, reasonError(ReasonPathInvalid, "sync temp object: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", tempWriteStats{}, reasonError(ReasonPathInvalid, "close temp object: %w", err)
	}
	return tempPath, stats, nil
}

func (s *Store) commitPreparedObject(sessionID, tempPath string, artifact RawOutputArtifact) error {
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
		if crypto.IsSessionStreamEncrypted(stored) {
			var decrypted bytes.Buffer
			if err := crypto.DecryptSessionStream(context.Background(), &decrypted, bytes.NewReader(stored), s.opts.Passphrase); err != nil {
				return nil, reasonError(ReasonDecryptFailed, "decrypt object: %w", err)
			}
			body = decrypted.Bytes()
		} else {
			decrypted, err := crypto.DecryptSession(stored, s.opts.Passphrase)
			if err != nil {
				return nil, reasonError(ReasonDecryptFailed, "decrypt object: %w", err)
			}
			body = decrypted
		}
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
