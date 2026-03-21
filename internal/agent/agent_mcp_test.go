package agent

import (
	"testing"
)

// --- detectGitHubIntent additional tests ---

func TestDetectGitHubIntent_Keywords(t *testing.T) {
	positives := []struct {
		name  string
		input string
	}{
		{name: "issue", input: "create an issue for this"},
		{name: "PR", input: "open a PR for this change"},
		{name: "pull request", input: "review the pull request"},
		{name: "プルリクエスト", input: "プルリクエストを作成して"},
		{name: "プルリク", input: "プルリクを確認して"},
		{name: "イシュー", input: "イシューを作って"},
		{name: "actions", input: "check GitHub Actions"},
		{name: "workflow", input: "the workflow is failing"},
		{name: "ワークフロー", input: "ワークフローを確認"},
		{name: "CI", input: "CI is broken"},
		{name: "repo", input: "list my repos"},
		{name: "repository", input: "clone the repository"},
		{name: "リポジトリ", input: "リポジトリを確認"},
		{name: "github", input: "check github"},
		{name: "ギットハブ", input: "ギットハブを見て"},
		{name: "gh", input: "use gh to list issues"},
		{name: "issues plural", input: "show all issues"},
	}

	for _, tt := range positives {
		t.Run(tt.name, func(t *testing.T) {
			if !detectGitHubIntent(tt.input) {
				t.Errorf("detectGitHubIntent(%q) = false, want true", tt.input)
			}
		})
	}
}

func TestDetectGitHubIntent_NegativeCases(t *testing.T) {
	negatives := []struct {
		name  string
		input string
	}{
		{name: "code fix", input: "fix the bug in main.go"},
		{name: "code review", input: "review my code changes"},
		{name: "read file", input: "read the config file"},
		{name: "empty", input: ""},
		{name: "generic", input: "hello, how are you?"},
		{name: "build", input: "build the binary"},
		{name: "test", input: "run all tests"},
	}

	for _, tt := range negatives {
		t.Run(tt.name, func(t *testing.T) {
			if detectGitHubIntent(tt.input) {
				t.Errorf("detectGitHubIntent(%q) = true, want false", tt.input)
			}
		})
	}
}

func TestDetectGitHubIntent_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "GITHUB uppercase", input: "GITHUB actions status", want: true},
		{name: "Issue mixed case", input: "Get Issue #123", want: true},
		{name: "Pr uppercase", input: "Create a PR", want: true},
		{name: "ci lowercase", input: "ci is failing", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectGitHubIntent(tt.input); got != tt.want {
				t.Errorf("detectGitHubIntent(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
