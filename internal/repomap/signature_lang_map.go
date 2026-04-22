package repomap

func patternLangForPath(path string) string {
	switch extensionForPath(path) {
	case ".go":
		return "go"
	case ".py":
		return "py"
	case ".ts", ".tsx", ".js", ".jsx", ".mjs":
		return "js"
	case ".rs":
		return "rs"
	case ".java", ".kt", ".kts":
		return "java"
	case ".rb":
		return "rb"
	case ".php":
		return "php"
	case ".c", ".cpp", ".cc", ".h", ".hpp":
		return "c"
	case ".swift":
		return "swift"
	case ".scala":
		return "scala"
	case ".ex", ".exs":
		return "elixir"
	case ".lua":
		return "lua"
	case ".sh", ".bash", ".zsh":
		return "sh"
	default:
		return ""
	}
}
