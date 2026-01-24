package search

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExecuteAstGrep executes ast-grep for structural code search
// ast-grep is a Tree-sitter based structural code search tool
// Pattern examples:
//   - 'func $NAME($ARGS)' - matches Go functions
//   - 'async function $NAME' - matches JS async functions
//   - 'def $NAME($ARGS):' - matches Python functions
func ExecuteAstGrep(pattern, lang, path string) string {
	if pattern == "" {
		return "Error: pattern is required\n\n" + getAstGrepHelp()
	}

	// Check ast-grep binary
	if _, err := exec.LookPath("ast-grep"); err != nil {
		return "Error: ast-grep is not installed.\n\nInstall with one of:\n  brew install ast-grep\n  cargo install ast-grep --locked\n  npm i @ast-grep/cli -g\n  pip install ast-grep-cli"
	}

	green.Printf("🔍 ast-grep: searching for pattern '%s'\n", pattern)

	// Build command
	// ast-grep run --pattern '<pattern>' [--lang <lang>] [path]
	args := []string{"run", "--pattern", pattern}
	if lang != "" {
		args = append(args, "--lang", lang)
	}
	if path != "" {
		args = append(args, path)
	} else {
		args = append(args, ".")
	}

	cmd := exec.Command("ast-grep", args...)
	output, err := cmd.CombinedOutput()

	result := string(output)
	if err != nil {
		// ast-grep may return exit code 1 even with no matches
		if result == "" {
			return fmt.Sprintf("No matches found for pattern '%s'", pattern)
		}
		// Return error message as-is
		if strings.Contains(result, "error") || strings.Contains(result, "Error") {
			return fmt.Sprintf("ast-grep error: %s", result)
		}
	}

	// Empty result
	if strings.TrimSpace(result) == "" {
		return fmt.Sprintf("No matches found for pattern '%s'", pattern)
	}

	// Truncate long results
	lines := strings.Split(result, "\n")
	if len(lines) > 100 {
		result = strings.Join(lines[:100], "\n") +
			fmt.Sprintf("\n... (%d more lines)", len(lines)-100)
	}

	return result
}

// getAstGrepHelp returns ast-grep pattern examples
func getAstGrepHelp() string {
	return `ast-grep pattern examples:
  Go:     'func $NAME($ARGS)'              - function definition
  Go:     'if err != nil { $$$ }'          - error handling block
  Go:     'func $NAME($ARGS) error'        - function returning error
  JS/TS:  'async function $NAME($ARGS)'    - async function
  JS/TS:  'console.log($ARG)'              - console.log calls
  JS/TS:  'useState($INIT)'                - React useState hook
  JS/TS:  'try { $$$ } catch ($E) { $$$ }' - try-catch block
  Python: 'def $NAME($ARGS):'              - function definition
  Python: 'async def $NAME($ARGS):'        - async function
  Rust:   'fn $NAME($ARGS) -> Result<$T, $E>' - function returning Result

Metavariables:
  $NAME  - matches a single AST node (identifier, expression, etc.)
  $$$    - matches zero or more AST nodes (for body, statements)
  $_     - matches any single node (unnamed/don't care)

Language codes: go, js, ts, tsx, py, rs, java, rb, c, cpp, kt, swift, cs, scala, php

More info: https://ast-grep.github.io/`
}

// PatternSuggestion is a pattern suggestion struct
type PatternSuggestion struct {
	Pattern string
	Lang    string
	Hint    string
}

// GetPatternSuggestion returns recommended pattern from natural language query (for LLM assistance)
// This function can be used as reference when LLM converts natural language to ast-grep patterns
func GetPatternSuggestion(query string) PatternSuggestion {
	query = strings.ToLower(query)

	// Go patterns
	if strings.Contains(query, "go") || strings.Contains(query, "golang") {
		if strings.Contains(query, "error") && strings.Contains(query, "handling") {
			return PatternSuggestion{"if err != nil { $$$ }", "go", "Matches error handling blocks"}
		}
		if strings.Contains(query, "function") || strings.Contains(query, "func") {
			if strings.Contains(query, "error") {
				return PatternSuggestion{"func $NAME($ARGS) error", "go", "Matches functions returning error"}
			}
			return PatternSuggestion{"func $NAME($ARGS)", "go", "Matches function definitions"}
		}
		if strings.Contains(query, "method") {
			return PatternSuggestion{"func ($R $TYPE) $NAME($ARGS)", "go", "Matches method definitions"}
		}
		if strings.Contains(query, "struct") {
			return PatternSuggestion{"type $NAME struct { $$$ }", "go", "Matches struct definitions"}
		}
		if strings.Contains(query, "interface") {
			return PatternSuggestion{"type $NAME interface { $$$ }", "go", "Matches interface definitions"}
		}
	}

	// JavaScript/TypeScript patterns
	if strings.Contains(query, "javascript") || strings.Contains(query, "js") ||
		strings.Contains(query, "typescript") || strings.Contains(query, "ts") ||
		strings.Contains(query, "react") {
		if strings.Contains(query, "async") {
			return PatternSuggestion{"async function $NAME($ARGS)", "js", "Matches async functions"}
		}
		if strings.Contains(query, "usestate") || strings.Contains(query, "hook") {
			return PatternSuggestion{"useState($INIT)", "tsx", "Matches useState hooks"}
		}
		if strings.Contains(query, "console") {
			return PatternSuggestion{"console.log($ARG)", "js", "Matches console.log calls"}
		}
		if strings.Contains(query, "try") || strings.Contains(query, "catch") {
			return PatternSuggestion{"try { $$$ } catch ($E) { $$$ }", "js", "Matches try-catch blocks"}
		}
		if strings.Contains(query, "class") {
			return PatternSuggestion{"class $NAME { $$$ }", "js", "Matches class definitions"}
		}
		if strings.Contains(query, "function") {
			return PatternSuggestion{"function $NAME($ARGS)", "js", "Matches function definitions"}
		}
	}

	// Python patterns
	if strings.Contains(query, "python") || strings.Contains(query, "py") {
		if strings.Contains(query, "async") {
			return PatternSuggestion{"async def $NAME($ARGS):", "py", "Matches async functions"}
		}
		if strings.Contains(query, "function") || strings.Contains(query, "def") {
			return PatternSuggestion{"def $NAME($ARGS):", "py", "Matches function definitions"}
		}
		if strings.Contains(query, "class") {
			return PatternSuggestion{"class $NAME:", "py", "Matches class definitions"}
		}
	}

	// Rust patterns
	if strings.Contains(query, "rust") || strings.Contains(query, "rs") {
		if strings.Contains(query, "result") {
			return PatternSuggestion{"fn $NAME($ARGS) -> Result<$T, $E>", "rs", "Matches functions returning Result"}
		}
		if strings.Contains(query, "function") || strings.Contains(query, "fn") {
			return PatternSuggestion{"fn $NAME($ARGS)", "rs", "Matches function definitions"}
		}
		if strings.Contains(query, "struct") {
			return PatternSuggestion{"struct $NAME { $$$ }", "rs", "Matches struct definitions"}
		}
		if strings.Contains(query, "impl") {
			return PatternSuggestion{"impl $TYPE { $$$ }", "rs", "Matches impl blocks"}
		}
	}

	// Default: no specific pattern
	return PatternSuggestion{"", "", "No specific pattern found. Try: 'func $NAME($ARGS)' for Go, 'function $NAME($ARGS)' for JS"}
}
