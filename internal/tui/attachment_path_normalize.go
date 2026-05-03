package tui

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type normalizePastedPathStatus int

const (
	normalizePastedPathOK normalizePastedPathStatus = iota
	normalizePastedPathEmpty
	normalizePastedPathInvalidFileURI
)

type normalizePastedPathResult struct {
	status normalizePastedPathStatus
	path   string
}

func (r normalizePastedPathResult) isOK() bool {
	return r.status == normalizePastedPathOK
}

func normalizePastedPathToken(token string) normalizePastedPathResult {
	path := strings.TrimSpace(token)
	if path == "" {
		return normalizePastedPathResult{status: normalizePastedPathEmpty}
	}

	path = trimPathQuotes(path)
	path = strings.ReplaceAll(path, `\ `, " ")
	if path == "" {
		return normalizePastedPathResult{status: normalizePastedPathEmpty}
	}

	if strings.HasPrefix(strings.ToLower(path), "file://") {
		decoded, ok := decodeFileURIPath(path)
		if !ok {
			return normalizePastedPathResult{status: normalizePastedPathInvalidFileURI}
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
	return normalizePastedPathResult{status: normalizePastedPathOK, path: path}
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

func isPDFAttachmentPath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".pdf")
}
