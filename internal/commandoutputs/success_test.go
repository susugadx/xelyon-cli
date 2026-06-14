package commandoutputs

import (
	"strings"
	"testing"
)

func TestBuildReplacementValidationSuccessPlaceholder(t *testing.T) {
	output := strings.Repeat("ok\tgithub.com/acme/project\t0.001s\n", 40)

	replacement, reason, ok := BuildReplacement(NewRequest("go test ./...", output))

	if !ok || reason != "" {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want validation replacement", replacement, reason, ok)
	}
	if replacement.Reason() != "validation_success" || replacement.Classifier() != "validation" {
		t.Fatalf("replacement reason/classifier = %q/%q", replacement.Reason(), replacement.Classifier())
	}
	if text := replacement.Text(); strings.Contains(text, "\n") || !strings.Contains(text, `command="go test ./..."`) || !strings.Contains(text, "classifier=validation") {
		t.Fatalf("replacement text = %q, want single-line validation placeholder", text)
	}
}

func TestBuildReplacementObservationEvidenceKeepsRawOutputForArtifactBackedCompaction(t *testing.T) {
	output := numberedLines("match", 100)

	replacement, reason, ok := BuildReplacement(NewRequest(`rg "ProviderHistory" internal`, output))

	if ok || reason != observationEvidenceKeepReason {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want artifact-backed observation keep reason", replacement, reason, ok)
	}
}

func TestBuildReplacementEvidenceBearingSuccessIgnoresFailureWordsAndKeepsRawOutput(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		output     string
		keepReason string
		classifier string
	}{
		{
			name:       "observation grep output",
			command:    `rg "error:" internal`,
			output:     strings.Repeat(`internal/a.go:12: return fmt.Errorf("error: expected sentinel")`+"\n", 120),
			keepReason: observationEvidenceKeepReason,
			classifier: "observation",
		},
		{
			name:       "file dump output",
			command:    "cat internal/errors.txt",
			output:     strings.Repeat("FAIL\t./pkg\nundefined: generated example\nerror: literal fixture text\n", 120),
			keepReason: fileDumpEvidenceKeepReason,
			classifier: "file_dump",
		},
		{
			name:    "git diff output",
			command: "git diff",
			output: strings.Repeat(
				"diff --git a/internal/a.go b/internal/a.go\n@@ -1,2 +1,4 @@ func f()\n-old\n+return fmt.Errorf(\"error: %w\", err)\n+// fatal: example fixture text\n",
				80,
			),
			keepReason: gitDiffEvidenceKeepReason,
			classifier: "git_diff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := Decide(NewRequest(tt.command, tt.output))
			replacement, reason, ok := BuildReplacement(NewRequest(tt.command, tt.output))

			if ok || reason != tt.keepReason {
				t.Fatalf("BuildReplacement() = (%#v, %q, %v), want keep reason %q", replacement, reason, ok, tt.keepReason)
			}
			if decision.Action != DecisionArtifactBackedCandidate ||
				decision.SemanticRole != SemanticRoleDataBearing ||
				decision.Classifier != tt.classifier ||
				!decision.ArtifactPolicy.RequiredForApply {
				t.Fatalf("Decision = %#v, want artifact-backed evidence classifier %q", decision, tt.classifier)
			}
			if replacement.Text() != "" || replacement.SavedBytes() != 0 || replacement.SavedTokens() != 0 {
				t.Fatalf("replacement = %#v, want empty replacement for artifact-backed evidence", replacement)
			}
		})
	}
}

func TestBuildReplacementSensitiveSuccessKeepsRawOutputOutsideArtifactOptimization(t *testing.T) {
	output := strings.Repeat("TOKEN=secret-value\nPATH=/tmp/bin\n", 120)

	replacement, reason, ok := BuildReplacement(NewRequest("env", output))

	if ok || reason != sensitiveOutputKeepReason {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want sensitive raw keep reason", replacement, reason, ok)
	}
}

func TestBuildReplacementDataBearingSuccessKeepsRawOutput(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		prefix     string
		wantReason string
	}{
		{
			name:       "network fetch",
			command:    "curl 'https://api.example.test/items?token=secret-value'",
			prefix:     "api-result",
			wantReason: networkDataBearingKeepReason,
		},
		{
			name:       "database query",
			command:    `sqlite3 app.db "select * from audit_log"`,
			prefix:     "row-result",
			wantReason: databaseDataBearingKeepReason,
		},
		{
			name:       "git show diff",
			command:    "git show HEAD",
			prefix:     "diff-result",
			wantReason: gitShowEvidenceKeepReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := numberedLines(tt.prefix, 120)

			replacement, reason, ok := BuildReplacement(NewRequest(tt.command, output))

			if ok || reason != tt.wantReason {
				t.Fatalf("BuildReplacement() = (%#v, %q, %v), want keep reason %q", replacement, reason, ok, tt.wantReason)
			}
			if replacement.Text() != "" || replacement.SavedBytes() != 0 || replacement.SavedTokens() != 0 {
				t.Fatalf("replacement = %#v, want empty replacement for data-bearing keep", replacement)
			}
		})
	}
}

func TestBuildReplacementDatabaseOperationSuccessPlaceholder(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		output    string
		subfamily DatabaseSubfamily
		role      SemanticRole
	}{
		{
			name:      "migration success",
			command:   "npx prisma migrate deploy",
			output:    strings.Repeat("Migration 20260101000000_init applied\nProcess exited with code 0\n", 90),
			subfamily: DatabaseSubfamilyMigrationLog,
			role:      SemanticRoleSideEffect,
		},
		{
			name:      "operation success",
			command:   `sqlite3 app.db "vacuum"`,
			output:    strings.Repeat("database operation completed successfully\nProcess exited with code 0\n", 90),
			subfamily: DatabaseSubfamilyOperationLog,
			role:      SemanticRoleOperationLog,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := Decide(NewRequest(tt.command, tt.output))
			replacement, reason, ok := BuildReplacement(NewRequest(tt.command, tt.output))

			if !ok || reason != "" {
				t.Fatalf("BuildReplacement() = (%#v, %q, %v), want database operation success placeholder", replacement, reason, ok)
			}
			if replacement.Kind() != "omit_successful_database_operation_command_output" ||
				replacement.Reason() != "database_operation_success" ||
				replacement.Classifier() != "database_operation" {
				t.Fatalf("replacement = %#v, want database operation success placeholder", replacement)
			}
			if decision.Action != DecisionInlineCompact ||
				decision.SemanticRole != tt.role ||
				decision.Subfamily != string(tt.subfamily) ||
				decision.Evidence.SuccessSignal != "success" {
				t.Fatalf("Decision = %#v, want %s success inline compact", decision, tt.subfamily)
			}
		})
	}
}

func TestBuildReplacementDatabaseOperationSmallFailureLogKeepsRawOutput(t *testing.T) {
	output := "migration started\npartial output: command interrupted before completion\n"

	replacement, reason, ok := BuildReplacement(NewRequest("prisma migrate deploy", output))

	if ok || reason != "database_failure_not_large" {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want small database failure keep", replacement, reason, ok)
	}
	if replacement.Text() != "" || replacement.SavedBytes() != 0 || replacement.SavedTokens() != 0 {
		t.Fatalf("replacement = %#v, want empty replacement for small failure", replacement)
	}
}

func TestBuildReplacementSideEffectSuccessPlaceholder(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		output     string
		kind       string
		reason     string
		classifier string
	}{
		{
			name:       "package install",
			command:    "npm install",
			output:     strings.Repeat("added 120 packages\nProcess exited with code 0\n", 90),
			kind:       "omit_successful_package_command_output",
			reason:     "package_success",
			classifier: "package_install",
		},
		{
			name:       "deploy",
			command:    "vercel deploy",
			output:     strings.Repeat("deployment completed successfully\nProcess exited with code 0\n", 90),
			kind:       "omit_successful_deploy_command_output",
			reason:     "deploy_success",
			classifier: "deploy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacement, reason, ok := BuildReplacement(NewRequest(tt.command, tt.output))

			if !ok || reason != "" {
				t.Fatalf("BuildReplacement() = (%#v, %q, %v), want side-effect success placeholder", replacement, reason, ok)
			}
			if replacement.Kind() != tt.kind ||
				replacement.Reason() != tt.reason ||
				replacement.Classifier() != tt.classifier {
				t.Fatalf("replacement = %#v, want %s/%s/%s", replacement, tt.kind, tt.reason, tt.classifier)
			}
		})
	}
}

func TestBuildReplacementSideEffectSmallFailureLogKeepsRawOutput(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		output     string
		wantReason string
	}{
		{
			name:       "package interrupted",
			command:    "npm install",
			output:     "resolving packages\ncommand interrupted before completion\n",
			wantReason: "package_failure_not_large",
		},
		{
			name:       "deploy partial",
			command:    "vercel deploy",
			output:     "uploading deployment\npartial output: connection closed\n",
			wantReason: "deploy_failure_not_large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacement, reason, ok := BuildReplacement(NewRequest(tt.command, tt.output))

			if ok || reason != tt.wantReason {
				t.Fatalf("BuildReplacement() = (%#v, %q, %v), want small side-effect failure keep %q", replacement, reason, ok, tt.wantReason)
			}
		})
	}
}

func TestBuildReplacementUnknownSuccessSkips(t *testing.T) {
	_, reason, ok := BuildReplacement(NewRequest("custom-tool --dump", strings.Repeat("important domain output\n", 200)))
	if ok || reason != "command_output_unknown_skip" {
		t.Fatalf("BuildReplacement() reason=%q ok=%v, want unknown skip", reason, ok)
	}
}
