package providerhistory

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

func TestProjectApplyCompactsRunSkillScriptResultWithRefAndRehydrateGate(t *testing.T) {
	output := providerHistoryTestNumberedLines("skill-script-output", 6000)
	history := providerHistoryTestRunSkillScriptHistory(t, "call_script", output, map[string]string{
		"skill":     "coverage-audit",
		"script":    "scripts/report.sh",
		"args_json": `["--format","json","--path","internal/providerhistory"]`,
	})

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             Apply,
			RawOutputArtifactsMode:           RawOutputArtifactsApply,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-run-skill-script-apply",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	projected := result.History[1].Content
	if projected == output ||
		!strings.Contains(projected, "[compacted old run_skill_script result;") ||
		!strings.Contains(projected, "raw_output_ref=") ||
		!strings.Contains(projected, "surface=command_output;") ||
		!strings.Contains(projected, "family=run_skill_script;") ||
		!strings.Contains(projected, "classifier=skill_script_output;") {
		t.Fatalf("projected run_skill_script output = %q, want artifact-backed placeholder", projected)
	}
	for _, reject := range []string{"--format", "internal/providerhistory", "args_json"} {
		if strings.Contains(projected, reject) {
			t.Fatalf("projected placeholder leaked argument detail %q:\n%s", reject, projected)
		}
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_script")
	if candidate == nil ||
		candidate.CandidateOnly ||
		!candidate.ArtifactBackedCandidate ||
		!candidate.ArtifactBackedApplyEligible ||
		!candidate.ReplacementApplied ||
		candidate.RawOutputRefID == "" ||
		candidate.ArtifactBackedActualSavedBytes <= 0 ||
		candidate.ApproxArtifactBackedActualSavedTokens < providerHistoryRawOutputArtifactMinSavedTokens {
		t.Fatalf("run_skill_script candidate = %#v, want applied artifact-backed candidate", candidate)
	}
	if result.Report.ReplacedCount != 1 ||
		result.Report.RawOutputRefCount != 1 ||
		result.Report.DataBearingCandidateCount != 1 ||
		result.Report.ArtifactBackedActualSavedBytes != candidate.ArtifactBackedActualSavedBytes ||
		result.Report.EstimatedSavedBytes != candidate.ArtifactBackedActualSavedBytes ||
		!result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want run_skill_script artifact actual savings and response chain disable", result.Report)
	}
	ref := result.Report.RawOutputRefs[0]
	if ref.Surface != string(rawoutputs.SurfaceCommandOutput) ||
		ref.ToolName != "run_skill_script" ||
		ref.Family != "run_skill_script" ||
		ref.Classifier != "skill_script_output" ||
		!strings.Contains(ref.CommandPreview, "skill=coverage-audit") ||
		!strings.Contains(ref.CommandPreview, "script=scripts/report.sh") {
		t.Fatalf("raw output ref = %#v, want command_output run_skill_script metadata", ref)
	}
	for _, reject := range []string{"--format", "internal/providerhistory", "args_json"} {
		if strings.Contains(ref.CommandPreview, reject) {
			t.Fatalf("raw output command preview leaked argument detail %q: %#v", reject, ref)
		}
	}
}

func TestProjectDryRunReportsRunSkillScriptRawRefWithoutChangingPayload(t *testing.T) {
	output := providerHistoryTestNumberedLines("skill-script-output", 6000)
	history := providerHistoryTestRunSkillScriptHistory(t, "call_script", output, map[string]string{
		"skill":  "coverage-audit",
		"script": "scripts/report.sh",
		"args":   "--summary",
	})

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             DryRun,
			RawOutputArtifactsMode:           RawOutputArtifactsDryRun,
			RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
			SessionID:                        "session-run-skill-script-dry-run",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("dry-run projection changed run_skill_script payload:\n got %#v\nwant %#v", result.History, history)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_script")
	if candidate == nil ||
		candidate.CandidateOnly ||
		!candidate.ArtifactBackedCandidate ||
		candidate.RawOutputRefID == "" ||
		candidate.ReplacementApplied ||
		candidate.ArtifactBackedApplyEligible ||
		candidate.EstimatedSavedBytes <= 0 ||
		candidate.ApproxEstimatedSavedTokens < providerHistoryRawOutputArtifactMinSavedTokens {
		t.Fatalf("run_skill_script candidate = %#v, want artifact-backed dry-run estimate", candidate)
	}
	if result.Report.RawOutputRefCount != 1 ||
		result.Report.DataBearingCandidateCount != 1 ||
		result.Report.ArtifactBackedEstimatedSavedBytes != candidate.EstimatedSavedBytes ||
		result.Report.ArtifactBackedActualSavedBytes != 0 ||
		result.Report.EstimatedSavedBytes != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want dry-run raw ref and separated estimate", result.Report)
	}
}

func TestProjectKeepsSensitiveRunSkillScriptArgsOrOutputOutOfRawOutputArtifacts(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]string
		output     string
		wantStatus string
	}{
		{
			name: "sensitive args",
			args: map[string]string{
				"skill":     "coverage-audit",
				"script":    "scripts/report.sh",
				"args_json": `["--token","secret-value"]`,
			},
			output:     providerHistoryTestNumberedLines("skill-script-output", 6000),
			wantStatus: "sensitive_args",
		},
		{
			name: "sensitive output",
			args: map[string]string{
				"skill":  "coverage-audit",
				"script": "scripts/report.sh",
			},
			output:     strings.Repeat("Authorization: Bearer secret-value\nskill-script-output\n", 240),
			wantStatus: "sensitive_body",
		},
		{
			name: "bare colon secret output",
			args: map[string]string{
				"skill":  "coverage-audit",
				"script": "scripts/report.sh",
			},
			output:     strings.Repeat("token: ghp_secret\nskill-script-output\n", 240),
			wantStatus: "sensitive_body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := providerHistoryTestRunSkillScriptHistory(t, "call_script", tt.output, tt.args)
			result := Project(ProjectionInput{
				Messages: history,
				Policy: Policy{
					Mode:                             Apply,
					RawOutputArtifactsMode:           RawOutputArtifactsApply,
					RawOutputArtifactStore:           panicRawOutputArtifactStore{},
					SessionID:                        "session-run-skill-script-sensitive",
					RawOutputRehydrateContextEnabled: true,
					ActiveContextTransportAvailable:  true,
				},
			})

			if result.History[1].Content != tt.output {
				t.Fatalf("projected run_skill_script output changed:\n got %q\nwant raw output", result.History[1].Content)
			}
			candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_script")
			if candidate == nil ||
				candidate.ReplacementApplied ||
				candidate.RawOutputRefID != "" ||
				candidate.SafetyStatus != "sensitive" ||
				candidate.ArtifactGateStatus != tt.wantStatus ||
				candidate.KeepReason != string(rawoutputs.ReasonSensitiveArtifactForbidden) ||
				candidate.FailClosedReason != string(rawoutputs.ReasonSensitiveArtifactForbidden) {
				t.Fatalf("run_skill_script candidate = %#v, want sensitive fail-closed status %q", candidate, tt.wantStatus)
			}
			if result.Report.RawOutputRefCount != 0 ||
				len(result.Report.RawOutputRefs) != 0 ||
				result.Report.ReplacedCount != 0 ||
				result.Report.ResponsesChainDisabled {
				t.Fatalf("report = %#v, want no raw refs or provider-facing replacement for sensitive run_skill_script", result.Report)
			}
		})
	}
}

func TestProjectApplyKeepsRunSkillScriptRawWhenArtifactGateFails(t *testing.T) {
	largeOutput := providerHistoryTestNumberedLines("skill-script-output", 6000)
	smallOutput := providerHistoryTestNumberedLines("skill-script-output", 200)

	tests := []struct {
		name         string
		output       string
		policy       Policy
		wantReason   string
		wantRawRefs  int
		wantVerified bool
	}{
		{
			name:   "raw artifact mode disabled",
			output: largeOutput,
			policy: Policy{
				Mode:                             Apply,
				RawOutputArtifactsMode:           RawOutputArtifactsDisabled,
				RawOutputArtifactStore:           panicRawOutputArtifactStore{},
				SessionID:                        "session-run-skill-script-disabled",
				RawOutputRehydrateContextEnabled: true,
				ActiveContextTransportAvailable:  true,
			},
			wantReason:  runSkillScriptRawOutputArtifactsDisabledReason,
			wantRawRefs: 0,
		},
		{
			name:   "missing session",
			output: largeOutput,
			policy: Policy{
				Mode:                             Apply,
				RawOutputArtifactsMode:           RawOutputArtifactsApply,
				RawOutputArtifactStore:           panicRawOutputArtifactStore{},
				RawOutputRehydrateContextEnabled: true,
				ActiveContextTransportAvailable:  true,
			},
			wantReason:  "raw_output_ref_missing",
			wantRawRefs: 0,
		},
		{
			name:   "missing store",
			output: largeOutput,
			policy: Policy{
				Mode:                             Apply,
				RawOutputArtifactsMode:           RawOutputArtifactsApply,
				SessionID:                        "session-run-skill-script-missing-store",
				RawOutputRehydrateContextEnabled: true,
				ActiveContextTransportAvailable:  true,
			},
			wantReason:  runSkillScriptRawOutputArtifactMissingReason,
			wantRawRefs: 0,
		},
		{
			name:   "verify failure",
			output: largeOutput,
			policy: Policy{
				Mode:                   Apply,
				RawOutputArtifactsMode: RawOutputArtifactsApply,
				RawOutputArtifactStore: failingRawOutputArtifactStore{
					materializeResult: rawoutputs.CreateResult{Ref: rawoutputs.RawOutputRef{RefID: "rawout_run_skill_script_verify_failed"}},
					verifyResult:      rawoutputs.VerifyResult{Reason: rawoutputs.ReasonArtifactHashMismatch},
				},
				SessionID:                        "session-run-skill-script-verify-failure",
				RawOutputRehydrateContextEnabled: true,
				ActiveContextTransportAvailable:  true,
			},
			wantReason:  string(rawoutputs.ReasonArtifactHashMismatch),
			wantRawRefs: 0,
		},
		{
			name:   "threshold below",
			output: smallOutput,
			policy: Policy{
				Mode:                             Apply,
				RawOutputArtifactsMode:           RawOutputArtifactsApply,
				RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
				SessionID:                        "session-run-skill-script-threshold",
				RawOutputRehydrateContextEnabled: true,
				ActiveContextTransportAvailable:  true,
			},
			wantReason:   "raw_output_artifact_saved_tokens_below_threshold",
			wantRawRefs:  1,
			wantVerified: true,
		},
		{
			name:   "rehydrate unavailable",
			output: largeOutput,
			policy: Policy{
				Mode:                             Apply,
				RawOutputArtifactsMode:           RawOutputArtifactsApply,
				RawOutputArtifactStore:           providerHistoryTestRawOutputStore(t),
				SessionID:                        "session-run-skill-script-no-rehydrate",
				RawOutputRehydrateContextEnabled: false,
				ActiveContextTransportAvailable:  true,
			},
			wantReason:   runSkillScriptRawOutputRehydrateUnavailableReason,
			wantRawRefs:  1,
			wantVerified: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := providerHistoryTestRunSkillScriptHistory(t, "call_script", tt.output, map[string]string{
				"skill":  "coverage-audit",
				"script": "scripts/report.sh",
			})

			result := Project(ProjectionInput{Messages: history, Policy: tt.policy})

			if result.History[1].Content != tt.output {
				t.Fatalf("projected run_skill_script output changed:\n got %q\nwant raw output", result.History[1].Content)
			}
			candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_script")
			if candidate == nil ||
				!candidate.ArtifactBackedCandidate ||
				candidate.ReplacementApplied ||
				candidate.KeepReason != tt.wantReason ||
				candidate.FailClosedReason == "" {
				t.Fatalf("run_skill_script candidate = %#v, want raw keep reason %q", candidate, tt.wantReason)
			}
			if tt.wantVerified && candidate.ArtifactGateStatus != "verified" {
				t.Fatalf("ArtifactGateStatus = %q, want verified", candidate.ArtifactGateStatus)
			}
			if result.Report.RawOutputRefCount != tt.wantRawRefs ||
				result.Report.ReplacedCount != 0 ||
				result.Report.ResponsesChainDisabled {
				t.Fatalf("report = %#v, want raw refs %d and no replacement/chain disable", result.Report, tt.wantRawRefs)
			}
		})
	}
}

func TestProjectKeepsRunSkillScriptRawWhenIdentityOrFreshnessGateFails(t *testing.T) {
	output := providerHistoryTestNumberedLines("skill-script-output", 6000)

	tests := []struct {
		name       string
		history    []api.Message
		wantReason string
		wantCand   bool
	}{
		{
			name: "missing skill",
			history: providerHistoryTestRunSkillScriptHistory(t, "call_script", output, map[string]string{
				"script": "scripts/report.sh",
			}),
			wantReason: runSkillScriptRawOutputMissingIdentityReason,
			wantCand:   true,
		},
		{
			name: "missing script",
			history: providerHistoryTestRunSkillScriptHistory(t, "call_script", output, map[string]string{
				"skill": "coverage-audit",
			}),
			wantReason: runSkillScriptRawOutputMissingIdentityReason,
			wantCand:   true,
		},
		{
			name: "latest result",
			history: []api.Message{
				providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_script", "run_skill_script", map[string]string{"skill": "coverage-audit", "script": "scripts/report.sh"})),
				providerHistoryTestToolResult("call_script", "run_skill_script", output),
				{Role: "user", Content: "next prompt before assistant"},
			},
			wantReason: "latest_tool_result",
		},
		{
			name: "no later assistant",
			history: []api.Message{
				providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_script", "run_skill_script", map[string]string{"skill": "coverage-audit", "script": "scripts/report.sh"})),
				providerHistoryTestToolResult("call_script", "run_skill_script", output),
				{Role: "user", Content: "no later assistant"},
				providerHistoryTestToolResult("call_other", "read_file", "later non-assistant tool result"),
			},
			wantReason: "no_later_assistant_message",
		},
		{
			name: "empty output",
			history: providerHistoryTestRunSkillScriptHistory(t, "call_script", " \n\t", map[string]string{
				"skill":  "coverage-audit",
				"script": "scripts/report.sh",
			}),
			wantReason: "empty_run_skill_script_output",
			wantCand:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Project(ProjectionInput{
				Messages: tt.history,
				Policy: Policy{
					Mode:                             Apply,
					RawOutputArtifactsMode:           RawOutputArtifactsApply,
					RawOutputArtifactStore:           panicRawOutputArtifactStore{},
					SessionID:                        "session-run-skill-script-freshness",
					RawOutputRehydrateContextEnabled: true,
					ActiveContextTransportAvailable:  true,
				},
			})

			if !reflect.DeepEqual(result.History, tt.history) {
				t.Fatalf("projection changed raw-kept history:\n got %#v\nwant %#v", result.History, tt.history)
			}
			if result.Report.RawOutputRefCount != 0 || result.Report.ReplacedCount != 0 || result.Report.ResponsesChainDisabled {
				t.Fatalf("report = %#v, want no raw refs or replacement", result.Report)
			}
			candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_script")
			if tt.wantCand {
				if candidate == nil || candidate.KeepReason != tt.wantReason || candidate.FailClosedReason != tt.wantReason {
					t.Fatalf("candidate = %#v, want kept reason %q", candidate, tt.wantReason)
				}
				return
			}
			if candidate != nil {
				t.Fatalf("candidate = %#v, want no candidate for linkage/freshness keep", candidate)
			}
			kept := providerHistoryTestKeptByToolCallID(result.Report, "call_script")
			if kept == nil || kept.KeepReason != tt.wantReason {
				t.Fatalf("kept = %#v, want kept reason %q", kept, tt.wantReason)
			}
		})
	}
}

func TestProjectRunSkillScriptRawOutputUsesLegacyMaterializeExactSource(t *testing.T) {
	output := providerHistoryTestNumberedLines("skill-script-output", 6000)
	store := providerHistoryTestRawOutputSpyStore(t)
	history := providerHistoryTestRunSkillScriptHistory(t, "call_script", output, map[string]string{
		"skill":  "coverage-audit",
		"script": "scripts/report.sh",
	})

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                             DryRun,
			RawOutputArtifactsMode:           RawOutputArtifactsDryRun,
			RawOutputArtifactStore:           store,
			SessionID:                        "session-run-skill-script-legacy-materialize",
			RawOutputRehydrateContextEnabled: true,
			ActiveContextTransportAvailable:  true,
		},
	})

	if result.Report.RawOutputRefCount != 1 {
		t.Fatalf("RawOutputRefCount = %d, want one run_skill_script raw output ref", result.Report.RawOutputRefCount)
	}
	assertProviderHistoryLegacyMaterialize(t, store, rawoutputs.SurfaceCommandOutput)
}

func providerHistoryTestRunSkillScriptHistory(t *testing.T, callID, output string, args map[string]string) []api.Message {
	t.Helper()
	return []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, callID, "run_skill_script", args)),
		providerHistoryTestToolResult(callID, "run_skill_script", output),
		{Role: "assistant", Content: "after run_skill_script"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}
}
