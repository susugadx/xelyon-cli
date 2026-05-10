package repomap

import (
	"path/filepath"
	"strings"
)

func isTestFile(path string) bool {
	origBase := filepath.Base(path)
	base := strings.ToLower(origBase)
	ext := strings.ToLower(filepath.Ext(base))
	switch {
	case strings.HasSuffix(base, "_test.go"):
		return true
	case strings.HasPrefix(base, "test_") && ext == ".py":
		return true
	case strings.HasSuffix(base, "_test.py"):
		return true
	case base == "conftest.py":
		return true
	case strings.HasSuffix(base, ".test.ts"), strings.HasSuffix(base, ".test.tsx"),
		strings.HasSuffix(base, ".test.js"), strings.HasSuffix(base, ".test.jsx"),
		strings.HasSuffix(base, ".spec.ts"), strings.HasSuffix(base, ".spec.tsx"),
		strings.HasSuffix(base, ".spec.js"), strings.HasSuffix(base, ".spec.jsx"):
		return true
	case isInTestsDir(path):
		return true
	case ext == ".java" && isTestSuffixName(origBase):
		return true
	case ext == ".kt" && isTestSuffixName(origBase):
		return true
	case ext == ".cs" && isTestSuffixName(origBase):
		return true
	case ext == ".swift" && isTestSuffixName(origBase):
		return true
	case ext == ".scala" && isTestSuffixName(origBase):
		return true
	case ext == ".php" && isTestSuffixName(origBase):
		return true
	case ext == ".rb" && (strings.HasSuffix(base, "_spec.rb") || strings.HasSuffix(base, "_test.rb")):
		return true
	case ext == ".exs" && strings.HasSuffix(base, "_test.exs"):
		return true
	case ext == ".lua" && (strings.HasSuffix(base, "_test.lua") || strings.HasSuffix(base, "_spec.lua")):
		return true
	case (ext == ".c" || ext == ".cpp" || ext == ".cc") && isTestSuffixName(origBase):
		return true
	default:
		return false
	}
}

// isTestSuffixName は *Test.ext / *Tests.ext / *Spec.ext のパターンを判定する。
// origBase は大文字小文字を保持した元のファイル名（PascalCase 判定に必要）。
func isTestSuffixName(origBase string) bool {
	ext := filepath.Ext(origBase)
	nameNoExt := strings.TrimSuffix(origBase, ext)
	// PascalCase: UserServiceTest, UserServiceTests, UserServiceSpec
	if strings.HasSuffix(nameNoExt, "Test") || strings.HasSuffix(nameNoExt, "Tests") || strings.HasSuffix(nameNoExt, "Spec") {
		return true
	}
	// snake_case: user_service_test, user_service_spec
	lower := strings.ToLower(nameNoExt)
	return strings.HasSuffix(lower, "_test") || strings.HasSuffix(lower, "_tests") || strings.HasSuffix(lower, "_spec")
}

// isInTestsDir はファイルパスが tests/ または test/ ディレクトリ配下かどうか判定する。
func isInTestsDir(path string) bool {
	normalized := filepath.ToSlash(path)
	return strings.HasPrefix(normalized, "tests/") ||
		strings.HasPrefix(normalized, "test/") ||
		strings.HasPrefix(normalized, "__tests__/") ||
		strings.Contains(normalized, "/tests/") ||
		strings.Contains(normalized, "/test/") ||
		strings.Contains(normalized, "/__tests__/")
}

// IsTestFile はテストファイルかどうかを返す。
func IsTestFile(path string) bool {
	return isTestFile(path)
}
