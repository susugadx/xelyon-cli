package tools

import (
	"os"
	"os/exec"
	"path/filepath"
)

// findProjectRoot searches for a project root file (e.g., go.mod) in the given path
// and its parent directories. Returns the directory containing the file, or "" if not found.
func findProjectRoot(startPath string, configFile string) string {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return ""
	}

	// Check current directory and parent directories
	for {
		if _, err := os.Stat(filepath.Join(absPath, configFile)); err == nil {
			return absPath
		}

		parent := filepath.Dir(absPath)
		if parent == absPath {
			// Reached root directory
			break
		}
		absPath = parent
	}
	return ""
}

// detectTestFramework detects available test framework
// It searches the given path and parent directories for project config files
func detectTestFramework(path string) (framework string, command string) {
	// Go: search for go.mod in path and parent directories
	if goRoot := findProjectRoot(path, "go.mod"); goRoot != "" {
		return "Go", "go test ./..."
	}

	// JavaScript/TypeScript: search for package.json
	if jsRoot := findProjectRoot(path, "package.json"); jsRoot != "" {
		if _, err := os.Stat(filepath.Join(jsRoot, "yarn.lock")); err == nil {
			return "JavaScript (yarn)", "yarn test"
		}
		return "JavaScript (npm)", "npm test"
	}

	// Python: search for pytest.ini or setup.py
	if findProjectRoot(path, "pytest.ini") != "" {
		return "Python (pytest)", "pytest"
	}
	if findProjectRoot(path, "setup.py") != "" {
		return "Python (pytest)", "pytest"
	}

	// Rust: search for Cargo.toml
	if findProjectRoot(path, "Cargo.toml") != "" {
		return "Rust", "cargo test"
	}

	return "", ""
}

// detectFormatter detects available formatter
// It searches the given path and parent directories for project config files
func detectFormatter(path string) (formatter string, command string) {
	// Go: search for go.mod
	if findProjectRoot(path, "go.mod") != "" {
		return "gofmt", "go fmt ./..."
	}

	// Prettier: search for config files
	prettierConfigs := []string{".prettierrc", ".prettierrc.json", ".prettierrc.js", "prettier.config.js"}
	for _, config := range prettierConfigs {
		if findProjectRoot(path, config) != "" {
			return "prettier", "prettier --write ."
		}
	}

	// Python: search for pyproject.toml
	if findProjectRoot(path, "pyproject.toml") != "" {
		if exec.Command("which", "black").Run() == nil {
			return "black", "black ."
		}
	}

	// Rust: search for Cargo.toml
	if findProjectRoot(path, "Cargo.toml") != "" {
		return "rustfmt", "cargo fmt"
	}

	return "", ""
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// hasGlobMatches checks if any files match the glob pattern
func hasGlobMatches(pattern string) bool {
	matches, _ := filepath.Glob(pattern)
	return len(matches) > 0
}

// detectLinter detects available linter for the project
func detectLinter(basePath string) (linterName, checkCmd, fixCmd string) {
	// Go: go.mod存在チェック
	if fileExists(filepath.Join(basePath, "go.mod")) {
		if commandExists("golangci-lint") {
			return "golangci-lint", "golangci-lint run", "golangci-lint run --fix"
		}
		if commandExists("go") {
			return "go vet", "go vet ./...", "" // fixコマンドなし
		}
	}

	// JavaScript/TypeScript: package.json + ESLint
	if fileExists(filepath.Join(basePath, "package.json")) {
		eslintConfigFiles := []string{".eslintrc", ".eslintrc.js", ".eslintrc.json", "eslint.config.js"}
		for _, configFile := range eslintConfigFiles {
			if fileExists(filepath.Join(basePath, configFile)) {
				return "eslint", "eslint .", "eslint . --fix"
			}
		}
	}

	// Python: *.pyファイル存在チェック
	if hasGlobMatches(filepath.Join(basePath, "*.py")) {
		if commandExists("ruff") {
			return "ruff", "ruff check .", "ruff check . --fix"
		}
		if commandExists("pylint") {
			return "pylint", "pylint .", "" // fixコマンドなし
		}
	}

	// Rust: Cargo.toml存在チェック
	if fileExists(filepath.Join(basePath, "Cargo.toml")) {
		return "clippy", "cargo clippy", "cargo clippy --fix --allow-dirty"
	}

	return "", "", "" // リンター未検出
}
