package review

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const defaultWorktreeSnapshotMaxHashBytes int64 = 64 * 1024

func buildWorktreeFingerprint(absPath string) (string, error) {
	info, err := os.Lstat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "missing", nil
		}
		return "", err
	}

	mode := info.Mode()
	switch {
	case mode.IsRegular():
		sum, err := hashFileSHA256Prefix(absPath, info.Size(), defaultWorktreeSnapshotMaxHashBytes)
		if err != nil {
			return "", err
		}
		if info.Size() > defaultWorktreeSnapshotMaxHashBytes {
			return fmt.Sprintf("file-large:%s:%d:%d:%s", mode.String(), info.Size(), info.ModTime().UnixNano(), sum), nil
		}
		return fmt.Sprintf("file:%s:%s", mode.String(), sum), nil
	case mode&os.ModeSymlink != 0:
		target, err := os.Readlink(absPath)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("symlink:%s:%s", mode.String(), target), nil
	default:
		return fmt.Sprintf("other:%s:%d", mode.String(), info.Size()), nil
	}
}

func hashFileSHA256Prefix(path string, size, maxBytes int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	limit := size
	if limit > maxBytes {
		limit = maxBytes
	}
	if _, err := io.Copy(h, io.LimitReader(f, limit)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
