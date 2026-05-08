package review

import "path/filepath"

type hostReadOnlySearchScenario struct {
	name    string
	command string
	args    []string
}

type hostReadOnlyRunnerBlockedScenario struct {
	name          string
	id            string
	command       string
	args          []string
	errorContains string
}

func hostReadOnlySearchCommandPolicyAllowScenarios() []hostReadOnlySearchScenario {
	return []hostReadOnlySearchScenario{
		{name: "rg pattern", command: "rg", args: []string{"pattern"}},
		{name: "rg absolute-like positional pattern", command: "rg", args: []string{"/etc"}},
		{name: "rg pattern with repo path", command: "rg", args: []string{"pattern", "internal"}},
		{name: "rg pattern with separator and repo path", command: "rg", args: []string{"pattern", "--", "internal"}},
		{name: "rg regexp option with absolute-like pattern", command: "rg", args: []string{"-e", "/etc", "--", "internal"}},
		{name: "rg regexp equals with absolute-like pattern", command: "rg", args: []string{"--regexp=/etc", "--", "internal"}},
		{name: "rg ignore-file inside repo", command: "rg", args: []string{"--ignore-file", "internal/ignore", "pattern"}},
		{name: "rg ignore-file equals inside repo", command: "rg", args: []string{"--ignore-file=internal/ignore", "pattern"}},
		{name: "grep pattern with separator and repo path", command: "grep", args: []string{"pattern", "--", "internal/file.go"}},
		{name: "grep recursive pattern with repo path", command: "grep", args: []string{"-r", "pattern", "internal"}},
		{name: "grep traversal-like positional pattern", command: "grep", args: []string{"../todo", "internal/file.go"}},
		{name: "grep regexp option with absolute-like pattern", command: "grep", args: []string{"-e", "/etc", "--", "internal/file.go"}},
		{name: "grep regexp equals with absolute-like pattern", command: "grep", args: []string{"--regexp=/etc", "--", "internal/file.go"}},
		{name: "grep exclude-from inside repo", command: "grep", args: []string{"--exclude-from", "internal/patterns", "pattern"}},
		{name: "grep exclude-from equals inside repo", command: "grep", args: []string{"--exclude-from=internal/patterns", "pattern"}},
	}
}

func hostReadOnlySearchPathPolicyAllowScenarios(repoRoot string) []hostReadOnlySearchScenario {
	allow := append([]hostReadOnlySearchScenario(nil), hostReadOnlySearchCommandPolicyAllowScenarios()...)
	allow = append(allow,
		hostReadOnlySearchScenario{name: "rg absolute repo-local path", command: "rg", args: []string{"pattern", filepath.Join(repoRoot, "internal")}},
		hostReadOnlySearchScenario{name: "rg files mode with repo-local absolute path", command: "rg", args: []string{"--files", filepath.Join(repoRoot, "internal")}},
		hostReadOnlySearchScenario{name: "rg post-separator regexp-like token with repo-local path", command: "rg", args: []string{"--", "--regexp", filepath.Join(repoRoot, "internal")}},
		hostReadOnlySearchScenario{name: "rg with iglob and explicit pattern", command: "rg", args: []string{"--iglob", "*.go", "-e", "pattern", filepath.Join(repoRoot, "internal")}},
		hostReadOnlySearchScenario{name: "grep recursive with explicit pattern after path", command: "grep", args: []string{"-r", filepath.Join(repoRoot, "internal"), "-e", "pattern"}},
		hostReadOnlySearchScenario{name: "grep recursive with pattern before path", command: "grep", args: []string{"-r", "pattern", filepath.Join(repoRoot, "internal")}},
		hostReadOnlySearchScenario{name: "grep directories recurse long option", command: "grep", args: []string{"--directories=recurse", "pattern", filepath.Join(repoRoot, "internal")}},
		hostReadOnlySearchScenario{name: "grep directories recurse short attached option", command: "grep", args: []string{"-drecurse", "pattern", filepath.Join(repoRoot, "internal")}},
		hostReadOnlySearchScenario{name: "grep absolute repo-local path", command: "grep", args: []string{"pattern", filepath.Join(repoRoot, "internal", "file.go")}},
		hostReadOnlySearchScenario{name: "grep post-separator short regexp-like token with repo-local path", command: "grep", args: []string{"--", "-e", filepath.Join(repoRoot, "internal", "file.go")}},
	)
	return allow
}

func hostReadOnlySearchPathPolicyBlockedOutsideScenarios() []hostReadOnlySearchScenario {
	return []hostReadOnlySearchScenario{
		{name: "rg absolute outside after separator", command: "rg", args: []string{"pattern", "--", "/etc"}},
		{name: "rg absolute outside without separator", command: "rg", args: []string{"pattern", "/etc"}},
		{name: "rg iglob with explicit pattern outside path", command: "rg", args: []string{"--iglob", "*.go", "-e", "pattern", "/etc"}},
		{name: "rg ignore-file outside path", command: "rg", args: []string{"--ignore-file", "/etc/ignore", "pattern"}},
		{name: "rg ignore-file equals outside path", command: "rg", args: []string{"--ignore-file=/etc/ignore", "pattern"}},
		{name: "rg pattern file short cluster outside path", command: "rg", args: []string{"-nf/usr/share/patterns", "needle", "internal"}},
		{name: "rg files mode outside path", command: "rg", args: []string{"--files", "/etc"}},
		{name: "rg post-separator regexp-like token with outside path", command: "rg", args: []string{"--", "--regexp", "/etc"}},
		{name: "grep absolute outside after separator", command: "grep", args: []string{"pattern", "--", "/etc/passwd"}},
		{name: "grep absolute outside without separator", command: "grep", args: []string{"pattern", "/etc/passwd"}},
		{name: "grep recursive with explicit pattern outside path", command: "grep", args: []string{"-r", "/etc", "-e", "pattern"}},
		{name: "grep recursive with pattern before outside path", command: "grep", args: []string{"-r", "root", "/etc/passwd"}},
		{name: "grep exclude-from outside path", command: "grep", args: []string{"--exclude-from", "/etc/patterns", "pattern"}},
		{name: "grep exclude-from equals outside path", command: "grep", args: []string{"--exclude-from=/etc/patterns", "pattern"}},
		{name: "grep pattern file short cluster outside path", command: "grep", args: []string{"-nf/usr/share/patterns", "needle", "internal"}},
		{name: "grep directories recurse long option outside path", command: "grep", args: []string{"--directories=recurse", "pattern", "/etc"}},
		{name: "grep directories recurse short attached option outside path", command: "grep", args: []string{"-drecurse", "pattern", "/etc"}},
		{name: "grep post-separator short regexp-like token with outside path", command: "grep", args: []string{"--", "-e", "/etc"}},
	}
}

func hostReadOnlyRunnerBlockedSearchOutsideScenarios() []hostReadOnlyRunnerBlockedScenario {
	return []hostReadOnlyRunnerBlockedScenario{
		{
			name:          "blocked rg obvious outside path",
			id:            "probe-blocked-rg-outside-path",
			command:       "rg",
			args:          []string{"pattern", "/etc"},
			errorContains: `rg path "/etc" is outside repository root`,
		},
		{
			name:          "blocked rg ignore-file outside path",
			id:            "probe-blocked-rg-ignore-file-outside-path",
			command:       "rg",
			args:          []string{"--ignore-file", "/etc/ignore", "pattern"},
			errorContains: `rg path "/etc/ignore" is outside repository root`,
		},
	}
}
