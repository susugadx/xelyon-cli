package bundlekeys

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func StableGoSymbolKey(packageDir, receiverNorm, name, kind, signature string) string {
	sigHash := sha256.Sum256([]byte(strings.TrimSpace(signature)))
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		"go",
		filepath.ToSlash(filepath.Clean(packageDir)),
		strings.TrimSpace(receiverNorm),
		strings.TrimSpace(name),
		strings.TrimSpace(kind),
		hex.EncodeToString(sigHash[:8]),
	)
}

func CanonicalGoSymbolKey(symbol navigation.SymbolCandidate) string {
	key := strings.TrimSpace(symbol.StableKey)
	if key == "" {
		key = StableGoSymbolKey(symbol.PackageDir, symbol.ReceiverNorm, symbol.Name, symbol.Kind, symbol.Signature)
	}
	if symbol.StableKeyCollision && strings.TrimSpace(symbol.File) != "" {
		return key + "|file=" + filepath.ToSlash(filepath.Clean(symbol.File))
	}
	return key
}

func CanonicalSymbolKey(lang, file string, line int, displayName string) string {
	return fmt.Sprintf("%s|%s|%d|%s", lang, file, line, displayName)
}
