package promptreduction

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

func TestRenderReviewProbeRawOutputContextBuildsLedgerAndRedacts(t *testing.T) {
	commandIndex := 0
	ref := rawoutputs.RawOutputRef{
		RefID:        "rawout_test",
		Surface:      string(rawoutputs.SurfaceReviewProbeResult),
		ContentHash:  "sha256:test",
		ByteSize:     123,
		ApproxTokens: 32,
	}

	text, ledger := RenderReviewProbeRawOutputContext(ReviewProbeRawOutputContextInput{
		Ledger: ReviewProbeRawOutputLedger{
			ReviewRunID:        "review-run",
			BudgetTokens:       4096,
			CanAcceptSaturated: true,
		},
		Entries: []ReviewProbeRawOutputContextEntry{{
			Ref: ref,
			Source: ReviewProbeRawOutputContextSource{
				ProbeID:        "probe-1",
				CommandIndex:   &commandIndex,
				CommandPreview: "cat /tmp/private-output",
				Status:         "passed",
				ExitCode:       0,
				AbsorbedBy:     []string{"scope_coverage.surface.api"},
			},
			Body: "body from /tmp/private-output\n" + strings.Repeat("rehydrated ", 40),
		}},
		Redactor: rawOutputContextTestRedactor{},
	})

	for _, want := range []string{
		"Review Probe Raw Output Context",
		"- ref: rawout_test",
		"  probe_id: probe-1",
		"  command_index: 0",
		"  command_preview: cat [redacted-path]",
		"body from [redacted-path]",
		"scope_coverage.surface.api",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("RenderReviewProbeRawOutputContext() text missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "/tmp/private-output") {
		t.Fatalf("RenderReviewProbeRawOutputContext() leaked unredacted path:\n%s", text)
	}
	if !ledger.CanAcceptSaturated ||
		len(ledger.RehydratedRefs) != 1 ||
		ledger.RehydratedRefs[0].RefID != ref.RefID ||
		ledger.RehydratedRefs[0].Status != "rehydrated" ||
		ledger.RehydratedRefs[0].CommandIndex == nil ||
		*ledger.RehydratedRefs[0].CommandIndex != commandIndex {
		t.Fatalf("RenderReviewProbeRawOutputContext() ledger = %#v, want one rehydrated command ref", ledger)
	}
}

func TestRenderReviewProbeRawOutputContextFailsClosedWhenBudgetTooSmall(t *testing.T) {
	text, ledger := RenderReviewProbeRawOutputContext(ReviewProbeRawOutputContextInput{
		Ledger: ReviewProbeRawOutputLedger{
			BudgetTokens:       1,
			CanAcceptSaturated: true,
		},
		Entries: []ReviewProbeRawOutputContextEntry{{
			Ref: rawoutputs.RawOutputRef{
				RefID:   "rawout_budget",
				Surface: string(rawoutputs.SurfaceReviewProbeResult),
			},
			Source: ReviewProbeRawOutputContextSource{
				ProbeID:        "probe-1",
				CommandPreview: "customtool",
				Status:         "passed",
			},
			Body: strings.Repeat("large raw output ", 80),
		}},
	})

	if text != "" {
		t.Fatalf("RenderReviewProbeRawOutputContext() text = %q, want empty when fail-closed", text)
	}
	if ledger.CanAcceptSaturated ||
		ledger.FailClosedReason != ReviewProbeRawOutputReasonRequiredRefBodyTooSmall ||
		len(ledger.BudgetExhaustedRefs) != 1 ||
		ledger.BudgetExhaustedRefs[0].Status != "budget_exhausted" {
		t.Fatalf("RenderReviewProbeRawOutputContext() ledger = %#v, want budget-exhausted fail-closed ledger", ledger)
	}
}

func TestReviewProbeRawOutputLedgerPtrDropsEmptyLedger(t *testing.T) {
	if got := ReviewProbeRawOutputLedgerPtr(ReviewProbeRawOutputLedger{}); got != nil {
		t.Fatalf("ReviewProbeRawOutputLedgerPtr(empty) = %#v, want nil", got)
	}
	ledger := ReviewProbeRawOutputLedger{FailClosedReason: ReviewProbeRawOutputReasonRequiredRefMissing}
	if got := ReviewProbeRawOutputLedgerPtr(ledger); got == nil || got.FailClosedReason != ledger.FailClosedReason {
		t.Fatalf("ReviewProbeRawOutputLedgerPtr(non-empty) = %#v, want ledger pointer", got)
	}
}

type rawOutputContextTestRedactor struct{}

func (rawOutputContextTestRedactor) RedactText(text string) string {
	return strings.ReplaceAll(text, "/tmp/private-output", "[redacted-path]")
}

func (rawOutputContextTestRedactor) RedactTexts(values []string) []string {
	redacted := make([]string, 0, len(values))
	for _, value := range values {
		redacted = append(redacted, strings.ReplaceAll(value, "/tmp/private-output", "[redacted-path]"))
	}
	return redacted
}

func (rawOutputContextTestRedactor) RedactPath(path string) string {
	return strings.ReplaceAll(path, "/tmp/private-output", "[redacted-path]")
}

func (rawOutputContextTestRedactor) RedactPaths(paths []string) []string {
	redacted := make([]string, 0, len(paths))
	for _, path := range paths {
		redacted = append(redacted, strings.ReplaceAll(path, "/tmp/private-output", "[redacted-path]"))
	}
	return redacted
}
