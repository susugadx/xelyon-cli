package rawoutputs

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	var body bytes.Buffer
	_, err := s.scanAndVerifyObject(context.Background(), record, chunkScannerFunc(func(chunk []byte) error {
		_, err := body.Write(chunk)
		return err
	}))
	if err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}

func (s *Store) scanAndVerifyObject(ctx context.Context, record ManifestRecord, scanner ChunkScanner) (ScanResult, error) {
	path, err := s.safeExistingSessionFilePath(record.Ref.SessionID, record.Artifact.RelativePath)
	if err != nil {
		return ScanResult{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ScanResult{}, reasonError(ReasonArtifactMissing, "object missing")
		}
		return ScanResult{}, reasonError(ReasonPathInvalid, "stat object: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return ScanResult{}, reasonError(ReasonPathInvalid, "object path is symlink")
	}
	f, err := os.Open(path)
	if err != nil {
		return ScanResult{}, reasonError(ReasonPathInvalid, "open object: %w", err)
	}
	defer f.Close()

	writer := newVerifyingChunkWriter(ctx, scanner)
	if record.Artifact.Encrypted {
		reader := bufio.NewReader(f)
		peeked, peekErr := reader.Peek(len(crypto.SessionStreamEncryptionMagic))
		if peekErr == nil && crypto.IsSessionStreamEncrypted(peeked) {
			if err := crypto.DecryptSessionStream(ctx, writer, reader, s.opts.Passphrase); err != nil {
				if scannerErr, ok := rawOutputChunkScannerErrorCause(err); ok {
					return ScanResult{}, rawOutputChunkScannerError{err: scannerErr}
				}
				if writer.consumerErr != nil && errors.Is(err, writer.consumerErr) {
					return ScanResult{}, rawOutputChunkScannerError{err: writer.consumerErr}
				}
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return ScanResult{}, err
				}
				return ScanResult{}, reasonError(ReasonDecryptFailed, "decrypt object: %w", err)
			}
		} else {
			stored, err := io.ReadAll(reader)
			if err != nil {
				return ScanResult{}, reasonError(ReasonPathInvalid, "read object: %w", err)
			}
			decrypted, err := crypto.DecryptSession(stored, s.opts.Passphrase)
			if err != nil {
				return ScanResult{}, reasonError(ReasonDecryptFailed, "decrypt object: %w", err)
			}
			if _, err := writer.Write(decrypted); err != nil {
				return ScanResult{}, err
			}
		}
	} else if err := s.scanPlainObject(ctx, f, writer); err != nil {
		return ScanResult{}, err
	}

	got := "sha256:" + hex.EncodeToString(writer.hash.Sum(nil))
	if got != record.Artifact.ContentHash || got != record.Ref.ContentHash {
		return ScanResult{}, reasonError(ReasonArtifactHashMismatch, "hash mismatch got %s want %s", got, record.Artifact.ContentHash)
	}
	return ScanResult{
		Ref:         record.Ref,
		ContentHash: got,
		SizeBytes:   writer.total,
	}, nil
}

func (s *Store) scanPlainObject(ctx context.Context, f *os.File, writer *verifyingChunkWriter) error {
	buf := make([]byte, s.opts.ChunkBytes)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := f.Read(buf)
		if n > 0 {
			if _, err := writer.Write(buf[:n]); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return reasonError(ReasonPathInvalid, "read object: %w", readErr)
		}
	}
}

type verifyingChunkWriter struct {
	ctx         context.Context
	scanner     ChunkScanner
	hash        hash.Hash
	total       int64
	consumerErr error
}

func newVerifyingChunkWriter(ctx context.Context, scanner ChunkScanner) *verifyingChunkWriter {
	if ctx == nil {
		ctx = context.Background()
	}
	return &verifyingChunkWriter{
		ctx:     ctx,
		scanner: scanner,
		hash:    sha256.New(),
	}
}

func (w *verifyingChunkWriter) Write(chunk []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	if len(chunk) == 0 {
		return 0, nil
	}
	if _, err := w.hash.Write(chunk); err != nil {
		return 0, err
	}
	w.total += int64(len(chunk))
	if err := w.scanner.Scan(chunk); err != nil {
		w.consumerErr = err
		return 0, rawOutputChunkScannerError{err: err}
	}
	return len(chunk), nil
}

type rawOutputChunkScannerError struct {
	err error
}

func (e rawOutputChunkScannerError) Error() string {
	if e.err == nil {
		return "raw output chunk scanner error"
	}
	return e.err.Error()
}

func (e rawOutputChunkScannerError) Unwrap() error {
	return e.err
}

func rawOutputChunkScannerErrorCause(err error) (error, bool) {
	var scannerErr rawOutputChunkScannerError
	if errors.As(err, &scannerErr) && scannerErr.err != nil {
		return scannerErr.err, true
	}
	return nil, false
}

type chunkScannerFunc func([]byte) error

func (f chunkScannerFunc) Scan(chunk []byte) error {
	return f(chunk)
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
