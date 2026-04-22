package repomap

import (
	"path/filepath"
	"strings"
)

func testSortBase(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "_test.go"):
		return strings.TrimSuffix(lower, "_test.go") + ".go"
	case strings.HasPrefix(lower, "test_") && strings.HasSuffix(lower, ".py"):
		return strings.TrimPrefix(lower, "test_")
	case strings.HasSuffix(lower, "_test.py"):
		return strings.TrimSuffix(lower, "_test.py") + ".py"
	case strings.HasSuffix(lower, ".test.ts"):
		return strings.TrimSuffix(lower, ".test.ts") + ".ts"
	case strings.HasSuffix(lower, ".test.tsx"):
		return strings.TrimSuffix(lower, ".test.tsx") + ".tsx"
	case strings.HasSuffix(lower, ".test.js"):
		return strings.TrimSuffix(lower, ".test.js") + ".js"
	case strings.HasSuffix(lower, ".test.jsx"):
		return strings.TrimSuffix(lower, ".test.jsx") + ".jsx"
	case strings.HasSuffix(lower, ".spec.ts"):
		return strings.TrimSuffix(lower, ".spec.ts") + ".ts"
	case strings.HasSuffix(lower, ".spec.tsx"):
		return strings.TrimSuffix(lower, ".spec.tsx") + ".tsx"
	case strings.HasSuffix(lower, ".spec.js"):
		return strings.TrimSuffix(lower, ".spec.js") + ".js"
	case strings.HasSuffix(lower, ".spec.jsx"):
		return strings.TrimSuffix(lower, ".spec.jsx") + ".jsx"
	default:
		ext := filepath.Ext(name)
		switch strings.ToLower(ext) {
		case ".java", ".kt", ".cs", ".php", ".swift", ".scala", ".c", ".cpp", ".cc":
			if base, ok := stripTestSuffixName(strings.TrimSuffix(name, ext)); ok {
				return strings.ToLower(base) + strings.ToLower(ext)
			}
		}
		return lower
	}
}

func stripTestSuffixName(name string) (string, bool) {
	switch {
	case strings.HasSuffix(name, "Tests"):
		return strings.TrimSuffix(name, "Tests"), true
	case strings.HasSuffix(name, "Test"):
		return strings.TrimSuffix(name, "Test"), true
	case strings.HasSuffix(name, "Spec"):
		return strings.TrimSuffix(name, "Spec"), true
	}

	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "_tests"):
		return name[:len(name)-len("_tests")], true
	case strings.HasSuffix(lower, "_test"):
		return name[:len(name)-len("_test")], true
	case strings.HasSuffix(lower, "_spec"):
		return name[:len(name)-len("_spec")], true
	default:
		return "", false
	}
}
