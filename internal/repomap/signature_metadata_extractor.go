package repomap

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func signatureMetadataForPath(path, sig string) (string, string, bool) {
	name, kind, ok := extractSignatureMetadataForLang(sig, patternLangForPath(path))
	if !ok {
		return "", "", false
	}
	exported := ast.IsSupportedFile(path) && isExportedName(name)
	return name, kind, exported
}

func extractSignatureMetadata(sig string) (string, string, bool) {
	if strings.Contains(sig, "=>") {
		if name, kind, ok := extractSignatureMetadataForLang(sig, "js"); ok {
			return name, kind, true
		}
	}
	return extractSignatureMetadataForLang(sig, "")
}

func extractSignatureMetadataForLang(sig, lang string) (string, string, bool) {
	if lang == "" || lang == "js" {
		if name, kind, ok := extractJSArrowFunctionMetadata(sig); ok {
			return name, kind, true
		}
	}

	for _, pattern := range signaturePatterns {
		if lang != "" && pattern.lang != "" && pattern.lang != lang {
			continue
		}
		matches := pattern.re.FindStringSubmatch(sig)
		if len(matches) != 2 {
			continue
		}
		name := strings.TrimSpace(matches[1])
		if name == "" {
			continue
		}
		return name, pattern.kind, true
	}
	return "", "", false
}
