package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func buildCatalogFingerprint(discover DiscoverResult) string {
	fingerprint, _ := buildCatalogFingerprintWithContent(discover)
	return fingerprint
}

func buildCatalogFingerprintWithContent(discover DiscoverResult) (string, map[string][]byte) {
	hasher := sha256.New()
	skillContents := make(map[string][]byte, len(discover.Skills))
	for _, root := range discover.Roots {
		_, _ = hasher.Write([]byte("root:" + cleanAbsPathOrFallback(root) + "\n"))
	}
	for _, skill := range discover.Skills {
		writeCatalogFingerprintEntry(hasher, "skill", cleanAbsPathOrFallback(skill.SkillPath), skillContents)
		for _, group := range skillResourceGroupOrder {
			writeCatalogResourceListingFingerprint(hasher, cleanAbsPathOrFallback(skill.Directory), group.String())
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), skillContents
}

func writeCatalogFingerprintEntry(hasher interface{ Write([]byte) (int, error) }, kind, path string, skillContents map[string][]byte) {
	if hasher == nil {
		return
	}
	_, _ = hasher.Write([]byte(kind + ":" + path))
	data, err := os.ReadFile(path)
	if err != nil {
		_, _ = hasher.Write([]byte("|err=" + err.Error() + "\n"))
		return
	}
	sum := sha256.Sum256(data)
	if skillContents != nil {
		skillContents[path] = append([]byte(nil), data...)
	}
	_, _ = hasher.Write([]byte("|sha256=" + hex.EncodeToString(sum[:]) + "\n"))
}

func writeCatalogResourceListingFingerprint(hasher interface{ Write([]byte) (int, error) }, skillDir, group string) {
	if hasher == nil {
		return
	}
	target := filepath.Join(skillDir, group)
	entries, err := os.ReadDir(target)
	if err != nil {
		_, _ = hasher.Write([]byte("list:" + target + "|err=" + err.Error() + "\n"))
		return
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	_, _ = hasher.Write([]byte("list:" + target + "|files=" + strings.Join(files, ",") + "\n"))
}
