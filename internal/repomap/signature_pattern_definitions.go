package repomap

import "regexp"

type signaturePattern struct {
	re   *regexp.Regexp
	kind string
	lang string // 対象言語（"go","js","py","rs","java","rb","php","c","swift","scala","sh"、"" で全言語）
}

var signaturePatterns = []signaturePattern{
	// Go
	{re: regexp.MustCompile(`^func\s+\([^)]*\)\s*([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[[^\]]+\])?\s*\(`), kind: "method", lang: "go"},
	{re: regexp.MustCompile(`^func\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[[^\]]+\])?\s*\(`), kind: "function", lang: "go"},
	{re: regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)\s+struct\b`), kind: "struct", lang: "go"},
	{re: regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)\s+interface\b`), kind: "interface", lang: "go"},
	{re: regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "type", lang: "go"},
	{re: regexp.MustCompile(`^const\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "const", lang: "go"},
	{re: regexp.MustCompile(`^var\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "var", lang: "go"},
	// JS/TS
	{re: regexp.MustCompile(`^export\s+default\s+(?:async\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "js"},
	{re: regexp.MustCompile(`^export\s+default\s+class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "js"},
	{re: regexp.MustCompile(`^export\s+(?:abstract\s+class|class)\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "js"},
	{re: regexp.MustCompile(`^export\s+(?:async\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "js"},
	{re: regexp.MustCompile(`^export\s+const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:async\s+)?\([^)]*\)(?:\s*:\s*.+)?\s*=>`), kind: "function", lang: "js"},
	{re: regexp.MustCompile(`^export\s+const\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "const", lang: "js"},
	{re: regexp.MustCompile(`^export\s+interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface", lang: "js"},
	{re: regexp.MustCompile(`^export\s+type\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "type", lang: "js"},
	{re: regexp.MustCompile(`^export\s+enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum", lang: "js"},
	{re: regexp.MustCompile(`^(?:async\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "js"},
	{re: regexp.MustCompile(`^class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "js"},
	{re: regexp.MustCompile(`^(?:const|let)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:async\s+)?\([^)]*\)(?:\s*:\s*.+)?\s*=>`), kind: "function", lang: "js"},
	{re: regexp.MustCompile(`^(?:const|let)\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "var", lang: "js"},
	{re: regexp.MustCompile(`^interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface", lang: "js"},
	// Python
	{re: regexp.MustCompile(`^(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`), kind: "function", lang: "py"},
	{re: regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "py"},
	// Rust
	{re: regexp.MustCompile(`^(?:pub\s+)?(?:async\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "rs"},
	{re: regexp.MustCompile(`^(?:pub\s+)?struct\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "struct", lang: "rs"},
	{re: regexp.MustCompile(`^(?:pub\s+)?enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum", lang: "rs"},
	{re: regexp.MustCompile(`^(?:pub\s+)?trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "trait", lang: "rs"},
	{re: regexp.MustCompile(`^impl(?:<[^>]*>)?\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "impl", lang: "rs"},
	// Java/Kotlin
	{re: regexp.MustCompile(`^(?:public |private |protected )?(?:static )?(?:abstract )?(?:class|record) ([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "java"},
	{re: regexp.MustCompile(`^(?:public |private |protected )?(?:static )?(?:abstract )?interface ([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface", lang: "java"},
	{re: regexp.MustCompile(`^(?:public |private |protected )?(?:static )?(?:abstract )?enum ([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum", lang: "java"},
	{re: regexp.MustCompile(`^(?:public |private |protected )?(?:static )?(?:abstract )?[A-Za-z0-9_<>,\[\]?]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`), kind: "function", lang: "java"},
	{re: regexp.MustCompile(`^(?:suspend\s+)?fun\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "java"},
	{re: regexp.MustCompile(`^(?:data |sealed |value )?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "java"},
	{re: regexp.MustCompile(`^object\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "object", lang: "java"},
	// Ruby
	{re: regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_]*[!?=]?)\b`), kind: "function", lang: "rb"},
	{re: regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_:]*)\b`), kind: "class", lang: "rb"},
	{re: regexp.MustCompile(`^\s*module\s+([A-Za-z_][A-Za-z0-9_:]*)\b`), kind: "module", lang: "rb"},
	// PHP
	{re: regexp.MustCompile(`^\s*(?:public |private |protected )?(?:static )?function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "php"},
	{re: regexp.MustCompile(`^\s*(?:abstract )?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "php"},
	{re: regexp.MustCompile(`^\s*(?:abstract )?interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface", lang: "php"},
	{re: regexp.MustCompile(`^\s*trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "trait", lang: "php"},
	{re: regexp.MustCompile(`^\s*enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum", lang: "php"},
	// C#
	{re: regexp.MustCompile(`^\s*(?:public |private |protected |internal )?(?:(?:static |abstract |sealed |partial )*)?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "csharp"},
	{re: regexp.MustCompile(`^\s*(?:public |private |protected |internal )?(?:static )?interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface", lang: "csharp"},
	{re: regexp.MustCompile(`^\s*(?:public |private |protected |internal )?enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum", lang: "csharp"},
	{re: regexp.MustCompile(`^\s*(?:public |private |protected |internal )?(?:readonly )?struct\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "struct", lang: "csharp"},
	{re: regexp.MustCompile(`^\s*(?:public |private |protected |internal )?(?:(?:static |virtual |abstract |override |async )*)?[A-Za-z0-9_<>,\[\]?]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`), kind: "function", lang: "csharp"},
	// C/C++
	{re: regexp.MustCompile(`^(?:typedef\s+)?struct\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "struct", lang: "c"},
	{re: regexp.MustCompile(`^(?:typedef\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "c"},
	{re: regexp.MustCompile(`^(?:typedef\s+)?(?:enum|union)\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "type", lang: "c"},
	{re: regexp.MustCompile(`^#define\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "const", lang: "c"},
	{re: regexp.MustCompile(`^namespace\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "namespace", lang: "c"},
	// Swift
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?func\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "swift"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "swift"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?struct\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "struct", lang: "swift"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "enum", lang: "swift"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?protocol\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "interface", lang: "swift"},
	{re: regexp.MustCompile(`^\s*(?:public |private |internal |open )?extension\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "impl", lang: "swift"},
	// Scala
	{re: regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "scala"},
	{re: regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "scala"},
	{re: regexp.MustCompile(`^\s*object\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "object", lang: "scala"},
	{re: regexp.MustCompile(`^\s*trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "trait", lang: "scala"},
	{re: regexp.MustCompile(`^\s*case\s+class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "class", lang: "scala"},
	{re: regexp.MustCompile(`^\s*case\s+object\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "object", lang: "scala"},
	{re: regexp.MustCompile(`^\s*sealed\s+trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "trait", lang: "scala"},
	// Elixir
	{re: regexp.MustCompile(`^\s*defmodule\s+([A-Za-z_][A-Za-z0-9_.]*)\b`), kind: "module", lang: "elixir"},
	{re: regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_!?]*)\b`), kind: "function", lang: "elixir"},
	{re: regexp.MustCompile(`^\s*defp\s+([A-Za-z_][A-Za-z0-9_!?]*)\b`), kind: "function", lang: "elixir"},
	{re: regexp.MustCompile(`^\s*defprotocol\s+([A-Za-z_][A-Za-z0-9_.]*)\b`), kind: "protocol", lang: "elixir"},
	{re: regexp.MustCompile(`^\s*defmacro\s+([A-Za-z_][A-Za-z0-9_!?]*)\b`), kind: "macro", lang: "elixir"},
	// Lua
	{re: regexp.MustCompile(`^(?:local\s+)?function\s+([A-Za-z_][A-Za-z0-9_.]*)\s*\(`), kind: "function", lang: "lua"},
	{re: regexp.MustCompile(`^(?:local\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*function\s*\(`), kind: "function", lang: "lua"},
	// Shell
	{re: regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\(\)`), kind: "function", lang: "sh"},
	{re: regexp.MustCompile(`^function\s+([A-Za-z_][A-Za-z0-9_]*)\b`), kind: "function", lang: "sh"},
}
