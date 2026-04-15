package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"sort"
)

// fingerprintFinalCheckTargetFiles は final_checks.commands の対象ファイル内容から
// 進捗 fingerprint を作る。silent edit も検出したいので、ファイル名集合ではなく
// 実ファイル内容を source of truth とする。
func fingerprintFinalCheckTargetFiles(paths []string) string {
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
			return ""
		}

		_, _ = hasher.Write(data)
		_, _ = hasher.Write([]byte{'\n'})
	}

	return hex.EncodeToString(hasher.Sum(nil))
}
