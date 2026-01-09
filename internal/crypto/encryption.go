package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// PBKDF2パラメータ
	keySize    = 32     // AES-256
	saltSize   = 16     // 128 bits
	iterations = 100000 // OWASP推奨
)

// EncryptSession はセッションデータを暗号化（AES-256-GCM）
func EncryptSession(plaintext []byte, passphrase string) ([]byte, error) {
	// ソルト生成
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// PBKDF2で鍵導出
	key := pbkdf2.Key([]byte(passphrase), salt, iterations, keySize, sha256.New)

	// AES-GCM初期化
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// ノンス生成
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 暗号化
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// フォーマット: salt(16) + ciphertext(nonce+encrypted)
	result := append(salt, ciphertext...)
	return result, nil
}

// DecryptSession はセッションデータを復号化
func DecryptSession(encrypted []byte, passphrase string) ([]byte, error) {
	if len(encrypted) < saltSize {
		return nil, fmt.Errorf("encrypted data too short")
	}

	// ソルト抽出
	salt := encrypted[:saltSize]
	ciphertext := encrypted[saltSize:]

	// PBKDF2で鍵導出
	key := pbkdf2.Key([]byte(passphrase), salt, iterations, keySize, sha256.New)

	// AES-GCM初期化
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// ノンス抽出
	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	// 復号化
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// GetOrCreatePassphrase はパスフレーズをファイルから取得（なければ生成）
func GetOrCreatePassphrase() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	keyFile := homeDir + "/.xelyon/.session_key"

	// 既存ファイルがあれば読み込み
	if data, err := os.ReadFile(keyFile); err == nil {
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err == nil && len(decoded) == 32 {
			return string(decoded), nil
		}
	}

	// なければ生成
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}

	// Base64エンコードして保存（0600パーミッション）
	encoded := base64.StdEncoding.EncodeToString(key)
	if err := os.WriteFile(keyFile, []byte(encoded), 0600); err != nil {
		return "", fmt.Errorf("failed to save key: %w", err)
	}

	return string(key), nil
}
