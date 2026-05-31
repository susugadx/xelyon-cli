package finalcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"sort"
)

// TargetInput は final check の対象ファイルと fallback 進捗 fingerprint。
type TargetInput struct {
	Files               []string
	ProgressFingerprint string
}

// TargetSnapshot は final check retry 判定に使う対象ファイル snapshot。
type TargetSnapshot struct {
	Files               []string
	ProgressFingerprint string
}

// BuildTargetSnapshot は対象ファイル内容を優先し、未特定なら入力 fingerprint に fallback する。
func BuildTargetSnapshot(input TargetInput) TargetSnapshot {
	files := append([]string(nil), input.Files...)
	progressFingerprint := FingerprintTargetFiles(files)
	if progressFingerprint == "" {
		progressFingerprint = input.ProgressFingerprint
	}

	return TargetSnapshot{
		Files:               files,
		ProgressFingerprint: progressFingerprint,
	}
}

// FingerprintTargetFiles は final_checks.commands の対象ファイル内容から進捗 fingerprint を作る。
func FingerprintTargetFiles(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)

	hasher := sha256.New()
	for _, path := range sorted {
		_, _ = hasher.Write([]byte(path))
		_, _ = hasher.Write([]byte{'\n'})

		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				_, _ = hasher.Write([]byte("<missing>"))
				_, _ = hasher.Write([]byte{'\n'})
				continue
			}
			_, _ = hasher.Write([]byte("<unreadable>"))
			_, _ = hasher.Write([]byte{'\n'})
			continue
		}

		_, _ = hasher.Write(data)
		_, _ = hasher.Write([]byte{'\n'})
	}

	return hex.EncodeToString(hasher.Sum(nil))
}
