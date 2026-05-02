package review

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

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
		sum, err := hashFileSHA256(absPath)
		if err != nil {
			return "", err
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

func hashFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
