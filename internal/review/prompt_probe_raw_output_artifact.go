package review

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
)

func (r *ReviewRunner) createReviewProbeRawOutputArtifact(ctx context.Context, phase ReviewModelPhase, source reviewProbeRawOutputSource) (rawoutputs.RawOutputRef, string, bool) {
	if source.body == "" {
		return rawoutputs.RawOutputRef{}, reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefMissing, false
	}
	if reviewProbeRawOutputLooksSensitive(source.body) ||
		reviewProbeRawOutputLooksSensitive(source.command.Command) ||
		reviewProbeRawOutputLooksSensitive(strings.Join(source.command.Args, " ")) {
		return rawoutputs.RawOutputRef{}, reviewpromptreduction.ReviewProbeRawOutputReasonSensitiveOrPrivateKeep, false
	}
	req := rawoutputs.CreateRequest{
		Surface:   rawoutputs.SurfaceReviewProbeResult,
		SessionID: r.rawOutputSessionID,
		Source: rawoutputs.SourceMetadata{
			CommandHash:    reviewProbeRawOutputCommandHash(source),
			CommandPreview: rawoutputs.SanitizeDisplayPreview(reviewProbeRawOutputCommandPreview(source), reviewpromptreduction.ReviewProbeRawOutputCommandPreviewRunes),
			ToolName:       "review_probe_result",
			ToolCallID:     reviewProbeRawOutputToolCallID(source),
			EventID:        reviewProbeRawOutputEventID(r.reviewRunID, phase, source),
		},
		Classification: rawoutputs.ClassificationMetadata{
			SemanticRole: "data_bearing",
			Family:       "review_probe",
			Subfamily:    reviewProbeRawOutputSubfamily(source),
			Classifier:   "review_probe_result",
		},
		Body:          bytes.NewReader([]byte(source.body)),
		SizeHintBytes: int64(len(source.body)),
		Retention: rawoutputs.RetentionPolicy{
			Policy:    "session",
			CreatedAt: time.Now().UTC(),
		},
	}
	result, err := r.rawOutputArtifactStore.Create(ctx, req)
	if err != nil {
		return rawoutputs.RawOutputRef{}, reviewProbeRawOutputReasonFromRawOutputError(err), false
	}
	verify, err := r.rawOutputArtifactStore.Verify(ctx, result.Ref)
	if err != nil || !verify.OK {
		return rawoutputs.RawOutputRef{}, reviewProbeRawOutputVerifyReason(verify, err), false
	}
	return result.Ref, "", true
}

func reviewProbeRawOutputBodyForProbe(result reviewprobe.ReviewProbeResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "probe_id: %s\nmode: %s\nstatus: %s\nmutated_worktree: %t\noutput_truncated: %t\n", result.ID, result.Mode, result.Status, result.MutatedWorktree, result.OutputTruncated)
	if result.Error != "" {
		fmt.Fprintf(&b, "error:\n%s\n", result.Error)
	}
	for i, command := range result.CommandResults {
		fmt.Fprintf(&b, "\ncommand[%d]: %s\n", i, reviewProbeRawOutputCommandDisplay(command))
		fmt.Fprintf(&b, "status: %s\nexit_code: %d\nwork_dir: %s\noutput_truncated: %t\n", command.Status, command.ExitCode, command.WorkDir, command.OutputTruncated)
		if command.Output != "" {
			fmt.Fprintf(&b, "output:\n%s\n", command.Output)
		}
		if command.Error != "" {
			fmt.Fprintf(&b, "stderr:\n%s\n", command.Error)
		}
	}
	return strings.TrimSpace(b.String())
}

func reviewProbeRawOutputBodyForCommand(command reviewprobe.ReviewProbeCommandResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "command: %s\n", reviewProbeRawOutputCommandDisplay(command))
	fmt.Fprintf(&b, "status: %s\nexit_code: %d\nwork_dir: %s\noutput_truncated: %t\n", command.Status, command.ExitCode, command.WorkDir, command.OutputTruncated)
	if command.Output != "" {
		fmt.Fprintf(&b, "output:\n%s\n", command.Output)
	}
	if command.Error != "" {
		fmt.Fprintf(&b, "stderr:\n%s\n", command.Error)
	}
	return strings.TrimSpace(b.String())
}

func reviewProbeRawOutputCommandDisplay(command reviewprobe.ReviewProbeCommandResult) string {
	return reviewprobe.FormatProbeCommand(command.Command, command.Args)
}

func reviewProbeRawOutputCommandPreview(source reviewProbeRawOutputSource) string {
	if source.commandIndex == nil {
		return "probe_result " + source.probeID
	}
	return reviewProbeRawOutputCommandDisplay(source.command)
}

func reviewProbeRawOutputCommandHash(source reviewProbeRawOutputSource) string {
	hash := sha256.Sum256([]byte(strings.Join([]string{
		source.probeID,
		reviewProbeRawOutputCommandIndexHashKey(source.commandIndex),
		reviewProbeRawOutputCommandDisplay(source.command),
		source.command.WorkDir,
	}, "\x00")))
	return "sha256:" + hex.EncodeToString(hash[:])
}

func reviewProbeRawOutputCommandIndexHashKey(commandIndex *int) string {
	if commandIndex == nil {
		return "probe"
	}
	return fmt.Sprintf("command:%d", *commandIndex)
}

func reviewProbeRawOutputToolCallID(source reviewProbeRawOutputSource) string {
	if source.commandIndex == nil {
		return "probe:" + source.probeID
	}
	return fmt.Sprintf("probe:%s:command:%d", source.probeID, *source.commandIndex)
}

func reviewProbeRawOutputEventID(reviewRunID string, phase ReviewModelPhase, source reviewProbeRawOutputSource) string {
	if source.commandIndex == nil {
		return fmt.Sprintf("review:%s:%s:probe:%s", reviewRunID, phase, source.probeID)
	}
	return fmt.Sprintf("review:%s:%s:probe:%s:command:%d", reviewRunID, phase, source.probeID, *source.commandIndex)
}

func reviewProbeRawOutputSubfamily(source reviewProbeRawOutputSource) string {
	if source.commandIndex == nil {
		return "probe_result"
	}
	return "probe_command_result"
}

func reviewProbeRawOutputLooksSensitive(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return rawoutputs.RedactDisplaySecrets(value) != value
}

func reviewProbeRawOutputReasonFromRawOutputError(err error) string {
	switch rawoutputs.ReasonOf(err) {
	case rawoutputs.ReasonArtifactHashMismatch:
		return reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefHashInvalid
	case rawoutputs.ReasonArtifactQuarantined:
		return reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefQuarantined
	case rawoutputs.ReasonSensitiveArtifactForbidden:
		return reviewpromptreduction.ReviewProbeRawOutputReasonSensitiveOrPrivateKeep
	case rawoutputs.ReasonArtifactTooLarge, rawoutputs.ReasonSessionQuotaExceeded:
		return reviewpromptreduction.ReviewProbeRawOutputReasonBudgetRequiresRevision
	default:
		return reviewpromptreduction.ReviewProbeRawOutputReasonArtifactMissing
	}
}

func reviewProbeRawOutputVerifyReason(result rawoutputs.VerifyResult, err error) string {
	if err != nil {
		return reviewProbeRawOutputReasonFromRawOutputError(err)
	}
	switch result.Reason {
	case rawoutputs.ReasonArtifactHashMismatch:
		return reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefHashInvalid
	case rawoutputs.ReasonArtifactQuarantined:
		return reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefQuarantined
	case rawoutputs.ReasonArtifactTombstoned, rawoutputs.ReasonArtifactMissing:
		return reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefMissing
	default:
		return reviewpromptreduction.ReviewProbeRawOutputReasonArtifactMissing
	}
}

func reviewProbeRawOutputResolveReason(err error) string {
	switch rawoutputs.ReasonOf(err) {
	case rawoutputs.ReasonArtifactHashMismatch:
		return reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefHashInvalid
	case rawoutputs.ReasonArtifactQuarantined:
		return reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefQuarantined
	case rawoutputs.ReasonArtifactTombstoned, rawoutputs.ReasonArtifactMissing:
		return reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefMissing
	default:
		return reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefMissing
	}
}
