package commandoutputs

import (
	"strings"
	"testing"
)

func TestDecideGitListAndFatalFailureEdges(t *testing.T) {
	t.Run("git file list compacts first last entries", func(t *testing.T) {
		output := numberedLines("internal/file", 90)

		decision := Decide(NewRequest("git ls-files", output))
		replacement, reason, ok := BuildReplacement(NewRequest("git ls-files", output))

		if !ok || reason != "" {
			t.Fatalf("BuildReplacement() = (%#v, %q, %v), want git file list compact", replacement, reason, ok)
		}
		if decision.Action != DecisionInlineCompact ||
			decision.Family != string(commandFamilyGitFileList) ||
			decision.SemanticRole != SemanticRoleOperationLog ||
			decision.Evidence.SuccessSignal != "success" {
			t.Fatalf("Decision = %#v, want git file list operation compact", decision)
		}
		text := replacement.Text()
		for _, want := range []string{"classifier=git_file_list", "entries=90", "internal/file-001", "internal/file-090", "[omitted 10 entries]"} {
			if !strings.Contains(text, want) {
				t.Fatalf("git list compact missing %q:\n%s", want, text)
			}
		}
		if strings.Contains(text, "internal/file-045") {
			t.Fatalf("git list compact retained middle entry that should be omitted:\n%s", text)
		}
	})

	t.Run("git show fatal failure is not treated as evidence", func(t *testing.T) {
		output := strings.Repeat("fatal: ambiguous argument 'missing': unknown revision\n", 120)

		decision := Decide(NewRequest("git show missing", output))
		replacement, reason, ok := BuildReplacement(NewRequest("git show missing", output))

		if !ok || reason != "" {
			t.Fatalf("BuildReplacement() = (%#v, %q, %v), want git failure compact", replacement, reason, ok)
		}
		if decision.Action != DecisionInlineCompact ||
			decision.Family != string(commandFamilyGitShow) ||
			decision.Classifier != "git_failure" ||
			decision.ArtifactPolicy.RequiredForApply {
			t.Fatalf("Decision = %#v, want inline git failure before artifact-backed evidence", decision)
		}
		if replacement.Classifier() != "git_failure" || !strings.Contains(replacement.Text(), "fatal: ambiguous argument") {
			t.Fatalf("replacement = %#v, want git failure evidence", replacement)
		}
	})
}

func TestDecideValidationCommandParsingEdges(t *testing.T) {
	t.Run("env assignment still classifies validation command", func(t *testing.T) {
		output := strings.Repeat("ok\tgithub.com/acme/project\t0.001s\n", 80)

		decision := Decide(NewRequest("GOFLAGS=-count=1 go test ./...", output))
		replacement, reason, ok := BuildReplacement(NewRequest("GOFLAGS=-count=1 go test ./...", output))

		if !ok || reason != "" {
			t.Fatalf("BuildReplacement() = (%#v, %q, %v), want validation success compact", replacement, reason, ok)
		}
		if decision.Action != DecisionInlineCompact ||
			decision.Family != string(commandFamilyValidation) ||
			decision.SemanticRole != SemanticRoleValidationLog ||
			replacement.Classifier() != "validation" {
			t.Fatalf("Decision/replacement = %#v / %#v, want validation success for env-prefixed command", decision, replacement)
		}
	})

	t.Run("shell composition keeps raw instead of guessing command family", func(t *testing.T) {
		output := strings.Repeat("ok\tgithub.com/acme/project\t0.001s\n", 80)

		decision := Decide(NewRequest("go test ./... && echo done", output))
		replacement, reason, ok := BuildReplacement(NewRequest("go test ./... && echo done", output))

		if ok || reason != "command_output_unknown_skip" {
			t.Fatalf("BuildReplacement() = (%#v, %q, %v), want unknown raw keep", replacement, reason, ok)
		}
		if decision.Action != DecisionKeepRaw ||
			decision.Family != string(commandFamilyUnknown) ||
			decision.SemanticRole != SemanticRoleUnknown ||
			decision.KeepReason != "command_output_unknown_skip" {
			t.Fatalf("Decision = %#v, want unknown keep for shell composition", decision)
		}
	})

	t.Run("validation command with lint error text uses lint failure classifier", func(t *testing.T) {
		output := strings.Join([]string{
			"internal/a.go:1: error: unexpected console statement",
			strings.Repeat("lint detail\n", 90),
		}, "\n")

		replacement, reason, ok := BuildReplacement(NewRequest("make ci-check", output))

		if !ok || reason != "" {
			t.Fatalf("BuildReplacement() = (%#v, %q, %v), want lint failure compact", replacement, reason, ok)
		}
		if replacement.Classifier() != "lint_failure" {
			t.Fatalf("Classifier = %q, want lint_failure", replacement.Classifier())
		}
		if strings.Contains(replacement.Text(), "validation_success") {
			t.Fatalf("lint failure was mislabeled as validation success:\n%s", replacement.Text())
		}
	})
}
