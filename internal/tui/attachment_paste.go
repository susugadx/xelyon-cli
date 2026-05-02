package tui

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandruntime"
)

const maxDroppedAttachments = 12

func (m *Model) tryAttachDroppedPaths(content string) bool {
	paths, ok := parseDroppedPaths(content)
	if !ok {
		return false
	}

	added := 0
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return false
		}
		kind := composerAttachmentFile
		if isImageAttachmentPath(path) {
			kind = composerAttachmentImage
		}
		if m.appendAttachment(composerAttachment{Kind: kind, Path: path, Size: info.Size()}) {
			added++
		}
	}
	if added == 0 {
		return false
	}

	m.setTransientStatus(fmt.Sprintf("Attached %d item(s)", added))
	m.chromeDirty = true
	return true
}

func parseDroppedPaths(content string) ([]string, bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, false
	}

	lines := strings.Split(strings.ReplaceAll(trimmed, "\r\n", "\n"), "\n")
	var candidates []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		tokens := parsePastedPathTokens(line)
		if len(tokens) == 0 {
			return nil, false
		}
		for _, token := range tokens {
			path, ok := normalizePastedPathToken(token)
			if !ok {
				return nil, false
			}
			candidates = append(candidates, path)
		}
	}

	if len(candidates) == 0 || len(candidates) > maxDroppedAttachments {
		return nil, false
	}

	dedup := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, path := range candidates {
		if _, ok := seen[path]; ok {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return nil, false
		}
		seen[path] = struct{}{}
		dedup = append(dedup, path)
	}
	if len(dedup) == 0 {
		return nil, false
	}
	return dedup, true
}

func parsePastedPathTokens(line string) []string {
	if strings.Contains(line, `\ `) {
		return []string{line}
	}
	tokens := commandruntime.Split(line)
	if len(tokens) > 0 {
		return tokens
	}
	return []string{line}
}

func normalizePastedPathToken(token string) (string, bool) {
	path := strings.TrimSpace(token)
	if path == "" {
		return "", false
	}

	path = trimPathQuotes(path)
	path = strings.ReplaceAll(path, `\ `, " ")
	if path == "" {
		return "", false
	}

	if strings.HasPrefix(strings.ToLower(path), "file://") {
		decoded, ok := decodeFileURIPath(path)
		if !ok {
			return "", false
		}
		path = decoded
	}

	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}

	if isTUIWSLEnvironment() && looksLikeWindowsPath(path) {
		converted, err := convertWindowsPathToWSL(path)
		if err == nil && converted != "" {
			path = converted
		}
	}

	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return path, true
}

func decodeFileURIPath(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "file" {
		return "", false
	}

	decoded, err := url.PathUnescape(u.Path)
	if err != nil {
		return "", false
	}

	host := strings.TrimSpace(u.Host)
	switch {
	case host == "" || strings.EqualFold(host, "localhost"):
		// no-op
	case looksLikeWindowsDriveHost(host):
		decoded = host + decoded
	default:
		decoded = "//" + host + decoded
	}

	decoded = trimWindowsDriveLeadingSlash(decoded)
	if decoded == "" {
		return "", false
	}
	return decoded, true
}

func trimPathQuotes(path string) string {
	for {
		if len(path) < 2 {
			return path
		}
		if (strings.HasPrefix(path, `"`) && strings.HasSuffix(path, `"`)) ||
			(strings.HasPrefix(path, `'`) && strings.HasSuffix(path, `'`)) {
			path = strings.TrimSpace(path[1 : len(path)-1])
			continue
		}
		return path
	}
}

func trimWindowsDriveLeadingSlash(path string) string {
	if len(path) < 4 || path[0] != '/' {
		return path
	}
	drive := path[1]
	if !isASCIILetter(drive) || path[2] != ':' {
		return path
	}
	if path[3] != '\\' && path[3] != '/' {
		return path
	}
	return path[1:]
}

func looksLikeWindowsDriveHost(host string) bool {
	return len(host) == 2 && isASCIILetter(host[0]) && host[1] == ':'
}

func isASCIILetter(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func looksLikeWindowsPath(path string) bool {
	if len(path) < 3 {
		return false
	}
	drive := path[0]
	if !((drive >= 'A' && drive <= 'Z') || (drive >= 'a' && drive <= 'z')) {
		return false
	}
	if path[1] != ':' {
		return false
	}
	return path[2] == '\\' || path[2] == '/'
}

func isImageAttachmentPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	default:
		return false
	}
}
