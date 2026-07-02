package commandcatalog

import (
	"os"
	"strings"
	"testing"
)

func TestAttachLimitDocumentationConsistency(t *testing.T) {
	paths := []string{
		"../../README.md",
		"../../docs/commands.md",
		"../../docs/usage.md",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", path, err)
			}
			text := string(body)
			if !strings.Contains(text, "/attach") {
				t.Fatalf("%s should mention /attach", path)
			}
			if !strings.Contains(text, "最大12件") && !strings.Contains(text, "最大 12 件") {
				t.Fatalf("%s should mention attachment limit (12)", path)
			}
		})
	}
}

func TestHeadlessCIDocumentationIncludesPullRequestDelta(t *testing.T) {
	body, err := os.ReadFile("../../docs/ci.md")
	if err != nil {
		t.Fatalf("ReadFile(docs/ci.md) error = %v", err)
	}
	text := string(body)
	required := []string{
		"fetch-depth: 0",
		"BASE_SHA",
		"HEAD_SHA",
		"git merge-base",
		"MERGE_BASE",
		"git diff --name-status \"${MERGE_BASE}\" \"${HEAD_SHA}\"",
		"git diff --find-renames \"${MERGE_BASE}\" \"${HEAD_SHA}\"",
		"github.event.pull_request.head.repo.full_name == github.repository",
		"fork PR",
		"repository secrets",
		"skip",
		"delta を prompt へ渡してください",
	}
	for _, want := range required {
		if !strings.Contains(text, want) {
			t.Fatalf("docs/ci.md should include %q so read-only PR review has the pull request delta", want)
		}
	}
}
