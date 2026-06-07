package crypto

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		plaintext  string
		passphrase string
	}{
		{
			name:       "empty data",
			plaintext:  "",
			passphrase: "test-passphrase",
		},
		{
			name:       "simple text",
			plaintext:  "Hello, World!",
			passphrase: "my-secret-key",
		},
		{
			name:       "multiline JSON",
			plaintext:  `{"role":"user","content":"test message"}`,
			passphrase: "another-key",
		},
		{
			name:       "large data (1KB)",
			plaintext:  strings.Repeat("a", 1024),
			passphrase: "large-data-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 暗号化
			encrypted, err := EncryptSession([]byte(tt.plaintext), tt.passphrase)
			if err != nil {
				t.Fatalf("EncryptSession failed: %v", err)
			}

			// 復号化
			decrypted, err := DecryptSession(encrypted, tt.passphrase)
			if err != nil {
				t.Fatalf("DecryptSession failed: %v", err)
			}

			// 元データと一致することを確認
			if string(decrypted) != tt.plaintext {
				t.Errorf("Decrypted data does not match original:\nGot: %s\nExpected: %s", string(decrypted), tt.plaintext)
			}
		})
	}
}

func TestDecryptWithWrongPassphrase(t *testing.T) {
	plaintext := "secret message"
	correctPass := "correct-password"
	wrongPass := "wrong-password"

	// 正しいパスフレーズで暗号化
	encrypted, err := EncryptSession([]byte(plaintext), correctPass)
	if err != nil {
		t.Fatalf("EncryptSession failed: %v", err)
	}

	// 間違ったパスフレーズで復号化 → エラーになるべき
	_, err = DecryptSession(encrypted, wrongPass)
	if err == nil {
		t.Error("DecryptSession should fail with wrong passphrase, but succeeded")
	}

	if !strings.Contains(err.Error(), "decryption failed") {
		t.Errorf("Expected 'decryption failed' error, got: %v", err)
	}
}

func TestEncryptRandomness(t *testing.T) {
	plaintext := "same data"
	passphrase := "same-key"

	// 同じデータを2回暗号化
	encrypted1, err := EncryptSession([]byte(plaintext), passphrase)
	if err != nil {
		t.Fatalf("First EncryptSession failed: %v", err)
	}

	encrypted2, err := EncryptSession([]byte(plaintext), passphrase)
	if err != nil {
		t.Fatalf("Second EncryptSession failed: %v", err)
	}

	// 暗号文は異なるべき（ソルト・ノンスがランダムなため）
	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("Encrypted data should be different for same plaintext (due to random salt/nonce)")
	}

	// どちらも復号化できることを確認
	decrypted1, _ := DecryptSession(encrypted1, passphrase)
	decrypted2, _ := DecryptSession(encrypted2, passphrase)

	if string(decrypted1) != plaintext || string(decrypted2) != plaintext {
		t.Error("Both encrypted data should decrypt to same plaintext")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	plaintext := "important data"
	passphrase := "my-key"

	encrypted, err := EncryptSession([]byte(plaintext), passphrase)
	if err != nil {
		t.Fatalf("EncryptSession failed: %v", err)
	}

	// 暗号文を改ざん（最後のバイトを変更）
	if len(encrypted) > 0 {
		encrypted[len(encrypted)-1] ^= 0xFF
	}

	// 復号化 → GCM認証タグ検証でエラーになるべき
	_, err = DecryptSession(encrypted, passphrase)
	if err == nil {
		t.Error("DecryptSession should fail with tampered ciphertext, but succeeded")
	}
}

func TestDecryptShortData(t *testing.T) {
	tests := []struct {
		name      string
		encrypted []byte
		wantErr   string
	}{
		{
			name:      "empty data",
			encrypted: []byte{},
			wantErr:   "encrypted data too short",
		},
		{
			name:      "only salt (no ciphertext)",
			encrypted: make([]byte, 16), // saltSize
			wantErr:   "ciphertext too short",
		},
		{
			name:      "partial data",
			encrypted: []byte{1, 2, 3, 4, 5},
			wantErr:   "encrypted data too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecryptSession(tt.encrypted, "any-pass")
			if err == nil {
				t.Error("DecryptSession should fail with short data, but succeeded")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing '%s', got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestSessionStreamEncryptionRoundTrip(t *testing.T) {
	plaintext := strings.Repeat("stream-body-", 1024)
	var encrypted bytes.Buffer
	writer, err := NewSessionStreamEncryptWriter(&encrypted, "stream-pass")
	if err != nil {
		t.Fatalf("NewSessionStreamEncryptWriter() error = %v", err)
	}
	for _, chunk := range []string{plaintext[:17], plaintext[17:2048], plaintext[2048:]} {
		if _, err := writer.Write([]byte(chunk)); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if bytes.Contains(encrypted.Bytes(), []byte(plaintext[:64])) {
		t.Fatal("encrypted stream contains plaintext")
	}
	if !IsSessionStreamEncrypted(encrypted.Bytes()) {
		t.Fatal("IsSessionStreamEncrypted() = false, want true")
	}

	var decrypted bytes.Buffer
	if err := DecryptSessionStream(context.Background(), &decrypted, bytes.NewReader(encrypted.Bytes()), "stream-pass"); err != nil {
		t.Fatalf("DecryptSessionStream() error = %v", err)
	}
	if decrypted.String() != plaintext {
		t.Fatalf("decrypted stream mismatch")
	}
}

func TestSessionStreamDecryptRejectsTamperedCiphertext(t *testing.T) {
	var encrypted bytes.Buffer
	writer, err := NewSessionStreamEncryptWriter(&encrypted, "stream-pass")
	if err != nil {
		t.Fatalf("NewSessionStreamEncryptWriter() error = %v", err)
	}
	if _, err := writer.Write([]byte("stream-secret")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	tampered := append([]byte(nil), encrypted.Bytes()...)
	tampered[len(tampered)-2] ^= 0xff

	var decrypted bytes.Buffer
	if err := DecryptSessionStream(context.Background(), &decrypted, bytes.NewReader(tampered), "stream-pass"); err == nil {
		t.Fatal("DecryptSessionStream(tampered) error = nil, want failure")
	}
}

func TestSessionStreamEncryptionRoundTripEmpty(t *testing.T) {
	var encrypted bytes.Buffer
	writer, err := NewSessionStreamEncryptWriter(&encrypted, "stream-pass")
	if err != nil {
		t.Fatalf("NewSessionStreamEncryptWriter() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	var decrypted bytes.Buffer
	if err := DecryptSessionStream(context.Background(), &decrypted, bytes.NewReader(encrypted.Bytes()), "stream-pass"); err != nil {
		t.Fatalf("DecryptSessionStream(empty) error = %v", err)
	}
	if decrypted.String() != "" {
		t.Fatalf("decrypted empty stream = %q, want empty", decrypted.String())
	}
}

func TestSessionStreamDecryptRejectsTrailingBytesAfterFinalRecord(t *testing.T) {
	encrypted := encryptSessionStreamForTest(t, "stream-secret", "stream-pass")
	tampered := append(append([]byte(nil), encrypted...), []byte("trailing")...)

	var decrypted bytes.Buffer
	if err := DecryptSessionStream(context.Background(), &decrypted, bytes.NewReader(tampered), "stream-pass"); err == nil {
		t.Fatal("DecryptSessionStream(trailing bytes) error = nil, want failure")
	}
}

func TestSessionStreamDecryptRejectsTruncatedFinalRecord(t *testing.T) {
	encrypted := encryptSessionStreamForTest(t, "stream-secret", "stream-pass")
	truncated := append([]byte(nil), encrypted[:len(encrypted)-1]...)

	var decrypted bytes.Buffer
	if err := DecryptSessionStream(context.Background(), &decrypted, bytes.NewReader(truncated), "stream-pass"); err == nil {
		t.Fatal("DecryptSessionStream(truncated final record) error = nil, want failure")
	}
}

func TestSessionStreamDecryptRejectsWrongPassphrase(t *testing.T) {
	encrypted := encryptSessionStreamForTest(t, "stream-secret", "stream-pass")

	var decrypted bytes.Buffer
	if err := DecryptSessionStream(context.Background(), &decrypted, bytes.NewReader(encrypted), "wrong-pass"); err == nil {
		t.Fatal("DecryptSessionStream(wrong passphrase) error = nil, want failure")
	}
}

func TestSessionStreamDecryptRejectsLegacyTrailingBytesAfterTerminator(t *testing.T) {
	legacy := legacySessionStreamForTest(t, "legacy-secret", "stream-pass")
	tampered := append(append([]byte(nil), legacy...), 'x')

	var decrypted bytes.Buffer
	if err := DecryptSessionStream(context.Background(), &decrypted, bytes.NewReader(tampered), "stream-pass"); err == nil {
		t.Fatal("DecryptSessionStream(legacy trailing bytes) error = nil, want failure")
	}
}

func encryptSessionStreamForTest(t *testing.T, plaintext, passphrase string) []byte {
	t.Helper()
	var encrypted bytes.Buffer
	writer, err := NewSessionStreamEncryptWriter(&encrypted, passphrase)
	if err != nil {
		t.Fatalf("NewSessionStreamEncryptWriter() error = %v", err)
	}
	if _, err := writer.Write([]byte(plaintext)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return encrypted.Bytes()
}

func legacySessionStreamForTest(t *testing.T, plaintext, passphrase string) []byte {
	t.Helper()
	var encrypted bytes.Buffer
	salt := bytes.Repeat([]byte{0x42}, saltSize)
	gcm, err := newSessionStreamGCM(passphrase, salt)
	if err != nil {
		t.Fatalf("newSessionStreamGCM() error = %v", err)
	}
	if _, err := encrypted.Write([]byte(sessionStreamEncryptionMagicV1)); err != nil {
		t.Fatalf("write legacy magic: %v", err)
	}
	if _, err := encrypted.Write(salt); err != nil {
		t.Fatalf("write legacy salt: %v", err)
	}
	nonce := bytes.Repeat([]byte{0x24}, gcm.NonceSize())
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), sessionStreamChunkAAD(0))
	if err := writeSessionStreamRecordV1(&encrypted, nonce, ciphertext); err != nil {
		t.Fatalf("write legacy record: %v", err)
	}
	if _, err := encrypted.Write([]byte{0, 0, 0, 0}); err != nil {
		t.Fatalf("write legacy terminator: %v", err)
	}
	return encrypted.Bytes()
}

func TestGetOrCreatePassphrase(t *testing.T) {
	// 一時HOMEディレクトリを設定
	tmpHome := testutil.SetupTempHome(t)
	xelyonDir := filepath.Join(tmpHome, ".xelyon")
	keyFile := filepath.Join(xelyonDir, ".session_key")

	// 初回実行 → キーファイルが作成される
	key1, err := GetOrCreatePassphrase()
	if err != nil {
		t.Fatalf("First GetOrCreatePassphrase failed: %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("Key length should be 32 bytes, got: %d", len(key1))
	}

	// キーファイルが存在することを確認
	testutil.AssertFileExists(t, keyFile)

	// ファイルパーミッションが0600であることを確認
	info, err := os.Stat(keyFile)
	if err != nil {
		t.Fatalf("Failed to stat key file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("Key file permission should be 0600, got: %o", info.Mode().Perm())
	}

	// 2回目実行 → 同じキーが返される
	key2, err := GetOrCreatePassphrase()
	if err != nil {
		t.Fatalf("Second GetOrCreatePassphrase failed: %v", err)
	}

	if key1 != key2 {
		t.Error("GetOrCreatePassphrase should return same key on second call")
	}
}

func TestGetOrCreatePassphrase_CorruptedFile(t *testing.T) {
	// 一時HOMEディレクトリを設定
	tmpHome := testutil.SetupTempHome(t)
	xelyonDir := filepath.Join(tmpHome, ".xelyon")
	keyFile := filepath.Join(xelyonDir, ".session_key")

	// ディレクトリ作成
	if err := os.MkdirAll(xelyonDir, 0700); err != nil {
		t.Fatalf("Failed to create .xelyon dir: %v", err)
	}

	// 破損したキーファイルを作成（Base64デコードできない）
	if err := os.WriteFile(keyFile, []byte("invalid-base64!!!"), 0600); err != nil {
		t.Fatalf("Failed to write corrupted key file: %v", err)
	}

	// GetOrCreatePassphrase → 新しいキーを生成するべき
	key, err := GetOrCreatePassphrase()
	if err != nil {
		t.Fatalf("GetOrCreatePassphrase should succeed with corrupted file: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("New key length should be 32 bytes, got: %d", len(key))
	}

	// 新しいキーファイルが正しく保存されていることを確認
	data, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatalf("Failed to read key file: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		t.Errorf("Key file should contain valid base64: %v", err)
	}

	if len(decoded) != 32 {
		t.Errorf("Decoded key should be 32 bytes, got: %d", len(decoded))
	}
}
