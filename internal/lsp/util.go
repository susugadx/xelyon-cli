package lsp

import (
	"net/url"
	"path/filepath"
	"strings"
)

// FileToURI converts a file path to an LSP file:// URI
func FileToURI(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Convert to forward slashes (required for URI)
	absPath = filepath.ToSlash(absPath)

	// Ensure leading slash for non-Windows paths
	if !strings.HasPrefix(absPath, "/") {
		absPath = "/" + absPath
	}

	return "file://" + absPath
}

// URIToFile converts an LSP file:// URI to a file path
func URIToFile(uri string) string {
	if !strings.HasPrefix(uri, "file://") {
		return uri
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return strings.TrimPrefix(uri, "file://")
	}

	path := parsed.Path

	// Handle Windows paths: /C:/path -> C:/path
	if len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}

	return filepath.FromSlash(path)
}

// DetectLanguage returns the language ID for a file based on its extension
func DetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	// ===== Existing (4 languages) =====
	case ".go":
		return "go"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".js":
		return "javascript"
	case ".jsx":
		return "javascriptreact"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	// ===== Tier 1: Backend languages (11 languages) =====
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cc", ".cxx", ".hxx":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".kt", ".kts":
		return "kotlin"
	case ".swift":
		return "swift"
	case ".scala":
		return "scala"
	case ".ex", ".exs":
		return "elixir"
	case ".lua":
		return "lua"
	// ===== Tier 2: Frontend languages (4 languages) =====
	case ".css", ".scss", ".sass", ".less":
		return "css"
	case ".html", ".htm":
		return "html"
	case ".vue":
		return "vue"
	case ".svelte":
		return "svelte"
	// ===== Tier 3: Config/Script languages (5 languages) =====
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".sql":
		return "sql"
	case ".sh", ".bash", ".zsh":
		return "bash"
	case ".md", ".markdown":
		return "markdown"
	default:
		return ""
	}
}

// LanguageServerKey returns the key used in LSP configuration for a language
// This maps language IDs to their server configuration keys
func LanguageServerKey(language string) string {
	switch language {
	case "typescript", "typescriptreact", "javascript", "javascriptreact":
		return "typescript" // Both JS and TS use the same server (vtsls)
	case "c", "cpp":
		// C and C++ both use clangd, but are separate configs for flexibility
		return language
	default:
		return language
	}
}
