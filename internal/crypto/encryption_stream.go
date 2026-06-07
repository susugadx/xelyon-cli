package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	sessionStreamEncryptionMagicV1 = "XELYON_SESSION_STREAM_AESGCM_V1\n"

	// SessionStreamEncryptionMagic は current streaming session encryption format の識別子。
	SessionStreamEncryptionMagic = "XELYON_SESSION_STREAM_AESGCM_V2\n"

	maxSessionStreamCiphertextChunkBytes = 128 * 1024 * 1024

	sessionStreamRecordTypeData  byte = 1
	sessionStreamRecordTypeFinal byte = 2
)

// NewSessionStreamEncryptWriter は plaintext chunk を versioned AES-GCM stream として書き込む writer を返す。
func NewSessionStreamEncryptWriter(dst io.Writer, passphrase string) (io.WriteCloser, error) {
	if dst == nil {
		return nil, fmt.Errorf("destination is nil")
	}
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	gcm, err := newSessionStreamGCM(passphrase, salt)
	if err != nil {
		return nil, err
	}
	if _, err := dst.Write([]byte(SessionStreamEncryptionMagic)); err != nil {
		return nil, fmt.Errorf("write stream magic: %w", err)
	}
	if _, err := dst.Write(salt); err != nil {
		return nil, fmt.Errorf("write stream salt: %w", err)
	}
	return &sessionStreamEncryptWriter{dst: dst, gcm: gcm}, nil
}

type sessionStreamEncryptWriter struct {
	dst    io.Writer
	gcm    cipher.AEAD
	index  uint64
	closed bool
}

func (w *sessionStreamEncryptWriter) Write(p []byte) (int, error) {
	if w.closed {
		return 0, fmt.Errorf("stream encrypt writer is closed")
	}
	if len(p) == 0 {
		return 0, nil
	}
	nonce := make([]byte, w.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return 0, fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext := w.gcm.Seal(nil, nonce, p, sessionStreamRecordAAD(sessionStreamRecordTypeData, w.index))
	if len(ciphertext) > maxSessionStreamCiphertextChunkBytes {
		return 0, fmt.Errorf("ciphertext chunk too large: %d", len(ciphertext))
	}
	if err := writeSessionStreamRecordV2(w.dst, sessionStreamRecordTypeData, nonce, ciphertext); err != nil {
		return 0, err
	}
	w.index++
	return len(p), nil
}

func (w *sessionStreamEncryptWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	nonce := make([]byte, w.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate final nonce: %w", err)
	}
	ciphertext := w.gcm.Seal(nil, nonce, nil, sessionStreamRecordAAD(sessionStreamRecordTypeFinal, w.index))
	return writeSessionStreamRecordV2(w.dst, sessionStreamRecordTypeFinal, nonce, ciphertext)
}

// DecryptSessionStream は versioned AES-GCM stream を復号して dst に書き込む。
func DecryptSessionStream(ctx context.Context, dst io.Writer, src io.Reader, passphrase string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if dst == nil {
		return fmt.Errorf("destination is nil")
	}
	if src == nil {
		return fmt.Errorf("source is nil")
	}
	magic := make([]byte, len(SessionStreamEncryptionMagic))
	if _, err := io.ReadFull(src, magic); err != nil {
		return fmt.Errorf("read stream magic: %w", err)
	}
	magicText := string(magic)
	if magicText != SessionStreamEncryptionMagic && magicText != sessionStreamEncryptionMagicV1 {
		return fmt.Errorf("unsupported stream encryption format")
	}
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(src, salt); err != nil {
		return fmt.Errorf("read stream salt: %w", err)
	}
	gcm, err := newSessionStreamGCM(passphrase, salt)
	if err != nil {
		return err
	}
	if magicText == sessionStreamEncryptionMagicV1 {
		return decryptSessionStreamV1Records(ctx, dst, src, gcm)
	}
	return decryptSessionStreamV2Records(ctx, dst, src, gcm)
}

func decryptSessionStreamV1Records(ctx context.Context, dst io.Writer, src io.Reader, gcm cipher.AEAD) error {
	var index uint64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		nonce, ciphertext, ok, err := readSessionStreamRecordV1(src, gcm.NonceSize())
		if err != nil {
			return err
		}
		if !ok {
			return requireSessionStreamEOF(src)
		}
		plaintext, err := gcm.Open(nil, nonce, ciphertext, sessionStreamChunkAAD(index))
		if err != nil {
			return fmt.Errorf("decrypt stream chunk: %w", err)
		}
		if _, err := dst.Write(plaintext); err != nil {
			return fmt.Errorf("write decrypted stream chunk: %w", err)
		}
		index++
	}
}

func decryptSessionStreamV2Records(ctx context.Context, dst io.Writer, src io.Reader, gcm cipher.AEAD) error {
	var index uint64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		recordType, nonce, ciphertext, err := readSessionStreamRecordV2(src, gcm.NonceSize())
		if err != nil {
			return err
		}
		switch recordType {
		case sessionStreamRecordTypeData:
			plaintext, err := gcm.Open(nil, nonce, ciphertext, sessionStreamRecordAAD(recordType, index))
			if err != nil {
				return fmt.Errorf("decrypt stream chunk: %w", err)
			}
			if _, err := dst.Write(plaintext); err != nil {
				return fmt.Errorf("write decrypted stream chunk: %w", err)
			}
			index++
		case sessionStreamRecordTypeFinal:
			plaintext, err := gcm.Open(nil, nonce, ciphertext, sessionStreamRecordAAD(recordType, index))
			if err != nil {
				return fmt.Errorf("decrypt stream final record: %w", err)
			}
			if len(plaintext) != 0 {
				return fmt.Errorf("stream final record contained plaintext")
			}
			return requireSessionStreamEOF(src)
		default:
			return fmt.Errorf("unsupported stream record type: %d", recordType)
		}
	}
}

// IsSessionStreamEncrypted は data が versioned streaming session encryption 形式か返す。
func IsSessionStreamEncrypted(data []byte) bool {
	if len(data) < len(SessionStreamEncryptionMagic) {
		return false
	}
	magic := string(data[:len(SessionStreamEncryptionMagic)])
	return magic == SessionStreamEncryptionMagic || magic == sessionStreamEncryptionMagicV1
}

func newSessionStreamGCM(passphrase string, salt []byte) (cipher.AEAD, error) {
	key := pbkdf2.Key([]byte(passphrase), salt, iterations, keySize, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	return gcm, nil
}

func writeSessionStreamRecordV2(dst io.Writer, recordType byte, nonce, ciphertext []byte) error {
	if _, err := dst.Write([]byte{recordType}); err != nil {
		return fmt.Errorf("write stream record type: %w", err)
	}
	return writeSessionStreamRecordV1(dst, nonce, ciphertext)
}

func writeSessionStreamRecordV1(dst io.Writer, nonce, ciphertext []byte) error {
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(ciphertext)))
	if _, err := dst.Write(lenBuf[:]); err != nil {
		return fmt.Errorf("write stream chunk length: %w", err)
	}
	if _, err := dst.Write(nonce); err != nil {
		return fmt.Errorf("write stream nonce: %w", err)
	}
	if _, err := dst.Write(ciphertext); err != nil {
		return fmt.Errorf("write stream ciphertext: %w", err)
	}
	return nil
}

func readSessionStreamRecordV1(src io.Reader, nonceSize int) ([]byte, []byte, bool, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(src, lenBuf[:]); err != nil {
		return nil, nil, false, fmt.Errorf("read stream chunk length: %w", err)
	}
	size := binary.BigEndian.Uint32(lenBuf[:])
	if size == 0 {
		return nil, nil, false, nil
	}
	if size > maxSessionStreamCiphertextChunkBytes {
		return nil, nil, false, fmt.Errorf("ciphertext chunk too large: %d", size)
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(src, nonce); err != nil {
		return nil, nil, false, fmt.Errorf("read stream nonce: %w", err)
	}
	ciphertext := make([]byte, size)
	if _, err := io.ReadFull(src, ciphertext); err != nil {
		return nil, nil, false, fmt.Errorf("read stream ciphertext: %w", err)
	}
	return nonce, ciphertext, true, nil
}

func readSessionStreamRecordV2(src io.Reader, nonceSize int) (byte, []byte, []byte, error) {
	var typeBuf [1]byte
	if _, err := io.ReadFull(src, typeBuf[:]); err != nil {
		return 0, nil, nil, fmt.Errorf("read stream record type: %w", err)
	}
	nonce, ciphertext, ok, err := readSessionStreamRecordV1(src, nonceSize)
	if err != nil {
		return 0, nil, nil, err
	}
	if !ok {
		return 0, nil, nil, fmt.Errorf("stream v2 record must not use unauthenticated terminator")
	}
	return typeBuf[0], nonce, ciphertext, nil
}

func requireSessionStreamEOF(src io.Reader) error {
	var extra [1]byte
	for {
		n, err := src.Read(extra[:])
		if n > 0 {
			return fmt.Errorf("trailing data after stream final record")
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read after stream final record: %w", err)
		}
	}
}

func sessionStreamChunkAAD(index uint64) []byte {
	var aad [8]byte
	binary.BigEndian.PutUint64(aad[:], index)
	return aad[:]
}

func sessionStreamRecordAAD(recordType byte, index uint64) []byte {
	var aad [9]byte
	aad[0] = recordType
	binary.BigEndian.PutUint64(aad[1:], index)
	return aad[:]
}
