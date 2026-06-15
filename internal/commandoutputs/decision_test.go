package commandoutputs

import (
	"strings"
	"testing"
)

func TestDecideDataBearingPrecedesFailureCompact(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		output        string
		wantFamily    string
		wantSubfamily string
		wantKeep      string
	}{
		{
			name:       "network response with failure-like body",
			command:    "curl https://api.example.test/items",
			output:     strings.Repeat("Error: upstream returned domain error, but this is response body\n{\"error\":\"domain\"}\n", 120),
			wantFamily: "network",
			wantKeep:   networkDataBearingKeepReason,
		},
		{
			name:          "database query result with failure-like body",
			command:       `sqlite3 app.db "select * from audit_log"`,
			output:        strings.Repeat("Error: stored application error row\n1|error|expected fixture value\n", 120),
			wantFamily:    "database",
			wantSubfamily: string(DatabaseSubfamilyQueryResult),
			wantKeep:      databaseDataBearingKeepReason,
		},
		{
			name:          "row-like database output without select command",
			command:       "psql app",
			output:        strings.Repeat("id | status | note\n1 | failed | domain error fixture\n2 | ok | Error: expected row value\n", 80),
			wantFamily:    "database",
			wantSubfamily: string(DatabaseSubfamilyQueryResult),
			wantKeep:      databaseDataBearingKeepReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := Decide(NewRequest(tt.command, tt.output))
			if decision.Action != DecisionArtifactBackedCandidate ||
				decision.SemanticRole != SemanticRoleDataBearing ||
				decision.Family != tt.wantFamily ||
				decision.Subfamily != tt.wantSubfamily ||
				decision.KeepReason != tt.wantKeep ||
				!decision.ArtifactPolicy.RequiredForApply {
				t.Fatalf("Decision = %#v, want data-bearing artifact-backed candidate keep %q", decision, tt.wantKeep)
			}

			replacement, reason, ok := BuildReplacement(NewRequest(tt.command, tt.output))
			if ok || reason != tt.wantKeep || replacement.Text() != "" {
				t.Fatalf("BuildReplacement() = (%#v, %q, %v), want raw keep reason %q", replacement, reason, ok, tt.wantKeep)
			}
		})
	}
}

func TestDecideDatabaseConnectionErrorUsesFailurePath(t *testing.T) {
	output := strings.Repeat("Error: could not connect to server: connection refused\nProcess exited with code 2\n", 120)

	decision := Decide(NewRequest(`psql "postgres://db.example.test/app"`, output))

	if decision.Action != DecisionInlineCompact ||
		decision.SemanticRole != SemanticRoleOperationLog ||
		decision.Subfamily != string(DatabaseSubfamilyConnectionError) ||
		decision.Classifier != "database_failure" {
		t.Fatalf("Decision = %#v, want database connection failure compact", decision)
	}
	replacement, ok := decision.Replacement()
	if !ok || replacement.Classifier() != "database_failure" || !strings.Contains(replacement.Text(), "[compacted old failed command output") {
		t.Fatalf("Decision.Replacement() = (%#v, %v), want database failure compact", replacement, ok)
	}
}

func TestDecideSensitivePrecedesArtifactCandidate(t *testing.T) {
	output := strings.Repeat("TOKEN=secret-value\nPATH=/tmp/bin\n", 120)

	decision := Decide(NewRequest("env", output))

	if decision.Action != DecisionKeepRaw ||
		decision.SemanticRole != SemanticRoleSensitive ||
		decision.KeepReason != sensitiveOutputKeepReason ||
		decision.ArtifactPolicy.RequiredForApply {
		t.Fatalf("Decision = %#v, want sensitive raw keep without artifact policy", decision)
	}
	if replacement, ok := decision.Replacement(); ok {
		t.Fatalf("Decision.Replacement() = %#v, want no sensitive inline replacement", replacement)
	}
}

func TestBuildReplacementIsDerivedFromDecide(t *testing.T) {
	output := strings.Repeat("ok\tgithub.com/acme/project\t0.001s\n", 80)

	decision := Decide(NewRequest("go test ./...", output))
	decisionReplacement, decisionOK := decision.Replacement()
	replacement, reason, ok := BuildReplacement(NewRequest("go test ./...", output))

	if !decisionOK || !ok || reason != "" {
		t.Fatalf("Decision/BuildReplacement = (%#v, %v) / (%#v, %q, %v), want inline replacement", decisionReplacement, decisionOK, replacement, reason, ok)
	}
	if decisionReplacement.Kind() != replacement.Kind() ||
		decisionReplacement.Reason() != replacement.Reason() ||
		decisionReplacement.Classifier() != replacement.Classifier() ||
		decisionReplacement.Text() != replacement.Text() {
		t.Fatalf("BuildReplacement diverged from Decide:\n decision=%#v\n replacement=%#v", decisionReplacement, replacement)
	}
}
