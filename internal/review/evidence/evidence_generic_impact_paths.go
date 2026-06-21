package evidence

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func isReviewGenericImpactExcludedPath(path string) bool {
	normalized := filepath.ToSlash(path)
	if normalized == ".xelyon/review-runs" || strings.HasPrefix(normalized, ".xelyon/review-runs/") {
		return true
	}
	if reviewGenericImpactDefaultIgnore.Match(normalized, false) {
		return true
	}
	if isReviewGenericImpactSensitivePath(normalized) {
		return true
	}
	for _, part := range strings.Split(normalized, "/") {
		if _, ok := reviewGenericImpactExcludedPathParts[part]; ok {
			return true
		}
	}
	return false
}

func isReviewGenericImpactTestOrSpecPath(path string) bool {
	base := strings.ToLower(reviewGenericImpactPathBase(path))
	return strings.HasSuffix(base, "_test.go") ||
		strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.")
}

func isReviewGenericImpactNearbyTestPath(path string) bool {
	if isReviewGenericImpactTestOrSpecPath(path) {
		return true
	}
	for _, part := range strings.Split(strings.ToLower(filepath.ToSlash(path)), "/") {
		switch part {
		case "test", "tests", "__tests__":
			return true
		}
	}
	return false
}

func isReviewGenericImpactProjectConfigPath(path string) bool {
	base := strings.ToLower(reviewGenericImpactPathBase(path))
	switch base {
	case "package.json", "pyproject.toml", "cargo.toml", "makefile", "go.mod", "readme.md":
		return true
	}
	return strings.HasPrefix(base, "tsconfig") && strings.HasSuffix(base, ".json") ||
		strings.HasPrefix(base, "vite.config.") ||
		strings.HasPrefix(base, "next.config.")
}

func reviewGenericImpactDocsSearchFilter(path string) bool {
	return isReviewGenericImpactSearchableTextPath(path) && matchDocsReviewInventoryPath(newReviewInventoryPath(path))
}

func reviewGenericImpactTextualSearchFilter(path string) bool {
	return isReviewGenericImpactSearchableTextPath(path) && !reviewGenericImpactDocsSearchFilter(path)
}

func reviewGenericImpactAllSearchFilter(path string) bool {
	return isReviewGenericImpactSearchableTextPath(path)
}

func isReviewGenericImpactSearchableTextPath(path string) bool {
	switch strings.ToLower(reviewGenericImpactPathBase(path)) {
	case ".gitignore", ".ignore":
		return false
	default:
		return !isReviewGenericImpactSensitivePath(path)
	}
}

func isReviewGenericImpactSensitivePath(path string) bool {
	normalized := strings.ToLower(filepath.ToSlash(path))
	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		switch part {
		case ".aws", ".azure", ".gnupg", ".kube", ".ssh", "credential", "credentials", "secret", "secrets":
			return true
		}
	}
	base := reviewGenericImpactPathBase(normalized)
	switch base {
	case ".env", ".envrc", ".netrc", ".npmrc", ".pypirc", "credentials", "credential", "secret", "secrets",
		"id_dsa", "id_ecdsa", "id_ed25519", "id_rsa":
		return true
	}
	if strings.HasPrefix(base, ".env.") ||
		strings.HasSuffix(base, ".env") ||
		strings.HasPrefix(base, "credential.") ||
		strings.HasPrefix(base, "credentials.") ||
		strings.HasPrefix(base, "secret.") ||
		strings.HasPrefix(base, "secrets.") ||
		strings.HasSuffix(base, ".key") ||
		strings.HasSuffix(base, ".pem") ||
		strings.HasSuffix(base, ".p12") ||
		strings.HasSuffix(base, ".pfx") {
		return true
	}
	return false
}

func reviewGenericImpactPathStem(path string) string {
	base := reviewGenericImpactPathBase(path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	lower := strings.ToLower(stem)
	for _, suffix := range []string{".test", ".spec", "_test"} {
		if strings.HasSuffix(lower, suffix) {
			return stem[:len(stem)-len(suffix)]
		}
	}
	return stem
}

func reviewGenericImpactConfigToken(path string) string {
	base := reviewGenericImpactPathBase(path)
	if stem := reviewGenericImpactPathStem(path); stem != "" {
		return stem
	}
	return base
}

func reviewGenericImpactPathBase(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	return parts[len(parts)-1]
}

func reviewGenericImpactPathDir(path string) string {
	normalized := filepath.ToSlash(path)
	index := strings.LastIndex(normalized, "/")
	if index < 0 {
		return "."
	}
	dir := normalized[:index]
	if dir == "" {
		return "."
	}
	return dir
}

func reviewGenericImpactJoinPath(dir, name string) string {
	if dir == "" || dir == "." {
		return name
	}
	return dir + "/" + name
}

func reviewGenericImpactStringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func reviewGenericImpactSortedSet(set map[string]struct{}) []string {
	if len(set) == 0 {
		return []string{}
	}
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func strconvReviewGenericImpactLine(line int) string {
	if line == 0 {
		return "0"
	}
	return strconv.Itoa(line)
}
