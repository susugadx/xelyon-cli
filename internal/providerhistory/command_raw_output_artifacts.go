package providerhistory

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandoutputs"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

func recordProviderHistoryArtifactBackedCommandCandidate(report *CommandEditDryRunReport, policy Policy, entry *CommandEditDryRunCandidate, command, content string, decision commandoutputs.Decision) {
	entry.Classifier = decision.Classifier
	entry.SafetyStatus = "data_bearing"
	entry.FreshnessStatus = "old_enough"
	if policy.RawOutputArtifactsMode == RawOutputArtifactsDisabled {
		entry.Reason = providerHistoryCommandCandidateReasonFromKeepReason(decision.KeepReason)
		entry.KeepReason = decision.KeepReason
		entry.FailClosedReason = decision.KeepReason
		report.Kept = append(report.Kept, *entry)
		return
	}
	entry.Reason = "artifact_backed_data_bearing_command_output"
	entry.ArtifactBackedCandidate = true
	report.ArtifactBackedCommandCandidates++
	if rawoutputs.LooksSensitiveContent(command) {
		entry.SafetyStatus = "sensitive"
		entry.KeepReason = string(rawoutputs.ReasonSensitiveArtifactForbidden)
		entry.FailClosedReason = entry.KeepReason
		entry.ArtifactGateStatus = "sensitive_command"
		recordArtifactBackedKept(report, entry.KeepReason)
		report.Kept = append(report.Kept, *entry)
		return
	}
	if !rawOutputArtifactMaterializationAllowed(policy) {
		entry.KeepReason = providerHistoryRawOutputMaterializationReadOnlyReason
		entry.FailClosedReason = providerHistoryRawOutputMaterializationReadOnlyReason
		entry.ArtifactGateStatus = "read_only"
		recordArtifactBackedKept(report, entry.KeepReason)
		report.Kept = append(report.Kept, *entry)
		return
	}
	if strings.TrimSpace(policy.SessionID) == "" {
		entry.KeepReason = "raw_output_ref_missing"
		entry.FailClosedReason = "raw_output_ref_missing"
		entry.ArtifactGateStatus = "missing_session_id"
		recordArtifactBackedKept(report, entry.KeepReason)
		report.Kept = append(report.Kept, *entry)
		return
	}
	if policy.RawOutputArtifactStore == nil {
		entry.KeepReason = "raw_output_artifact_missing"
		entry.FailClosedReason = "raw_output_artifact_missing"
		entry.ArtifactGateStatus = "missing_store"
		recordArtifactBackedKept(report, entry.KeepReason)
		report.Kept = append(report.Kept, *entry)
		return
	}
	if rawoutputs.LooksSensitiveContent(content) {
		entry.SafetyStatus = "sensitive"
		entry.KeepReason = string(rawoutputs.ReasonSensitiveArtifactForbidden)
		entry.FailClosedReason = entry.KeepReason
		entry.ArtifactGateStatus = "sensitive_body"
		recordArtifactBackedKept(report, entry.KeepReason)
		report.Kept = append(report.Kept, *entry)
		return
	}

	createReq := rawoutputs.CreateRequest{
		Surface:   rawoutputs.SurfaceCommandOutput,
		SessionID: policy.SessionID,
		Source: rawoutputs.SourceMetadata{
			CommandHash:    commandHash(command),
			CommandPreview: command,
			ToolName:       entry.ToolName,
			ToolCallID:     entry.ToolCallID,
			EventID:        fmt.Sprintf("history:%d", entry.HistoryIndex),
			HistoryIndex:   entry.HistoryIndex,
		},
		Classification: rawoutputs.ClassificationMetadata{
			SemanticRole: string(decision.SemanticRole),
			Family:       decision.Family,
			Subfamily:    decision.Subfamily,
			Classifier:   decision.Classifier,
		},
		Body:          strings.NewReader(content),
		SizeHintBytes: int64(len(content)),
	}
	exactSourceID, ambiguous := providerHistoryLegacyRawOutputExactSourceID(policy.SessionID, rawoutputs.SurfaceCommandOutput, createReq.Source, content)
	createResult, err := policy.RawOutputArtifactStore.MaterializeLegacy(context.Background(), rawoutputs.LegacyMaterializeRequest{
		CreateRequest: createReq,
		ExactSourceID: exactSourceID,
		Ambiguous:     ambiguous,
	})
	if err != nil {
		reason := providerHistoryRawOutputCreateFailureReason(err)
		entry.KeepReason = reason
		entry.FailClosedReason = reason
		entry.ArtifactGateStatus = "create_failed"
		recordArtifactBackedKept(report, reason)
		report.Kept = append(report.Kept, *entry)
		return
	}
	verifyResult, err := policy.RawOutputArtifactStore.Verify(context.Background(), createResult.Ref)
	if err != nil || !verifyResult.OK {
		reason := providerHistoryRawOutputVerifyFailureReason(verifyResult, err)
		entry.KeepReason = reason
		entry.FailClosedReason = reason
		entry.ArtifactGateStatus = "verify_failed"
		recordArtifactBackedKept(report, reason)
		report.Kept = append(report.Kept, *entry)
		return
	}
	entry.RawOutputRefID = createResult.Ref.RefID
	entry.ArtifactGateStatus = "verified"
	report.RawOutputRefs = append(report.RawOutputRefs, createResult.Ref)

	replacementText := buildProviderHistoryArtifactBackedCommandPlaceholder(createResult.Ref, content)
	entry.SuggestedReplacementKind = "compact_old_data_bearing_command_output"
	entry.SuggestedReplacementText = replacementText
	savedBytes, savedTokens, thresholdStatus, thresholdOK := providerHistoryArtifactBackedReplacementEligibility(content, replacementText)
	entry.ThresholdStatus = thresholdStatus
	if savedBytes > 0 && savedTokens > 0 {
		entry.EstimatedSavedBytes = savedBytes
		entry.ApproxEstimatedSavedTokens = savedTokens
		report.ArtifactBackedCommandDryRunEstimatedSavedBytes += savedBytes
		report.ApproxArtifactBackedCommandDryRunEstimatedSavedTokens += savedTokens
	}
	if !thresholdOK {
		entry.KeepReason = thresholdStatus
		entry.FailClosedReason = thresholdStatus
		recordArtifactBackedKept(report, thresholdStatus)
		report.Kept = append(report.Kept, *entry)
		return
	}
	if !policy.RawOutputRehydrateContextEnabled || !policy.ActiveContextTransportAvailable {
		entry.RehydrateGateStatus = "unsupported"
		entry.KeepReason = "raw_output_rehydrate_unsupported"
		entry.FailClosedReason = "raw_output_rehydrate_unsupported"
		recordArtifactBackedKept(report, entry.KeepReason)
		report.Kept = append(report.Kept, *entry)
		return
	}
	entry.RehydrateGateStatus = "available"
	entry.ArtifactBackedApplyEligible = policy.RawOutputArtifactsMode == RawOutputArtifactsApply && policy.Mode == Apply
	if entry.ArtifactBackedApplyEligible {
		report.ArtifactBackedCommandApplyEligible++
		return
	}
	if policy.RawOutputApplyDisabledReason != "" {
		entry.KeepReason = policy.RawOutputApplyDisabledReason
		entry.FailClosedReason = policy.RawOutputApplyDisabledReason
	} else if policy.Mode != Apply {
		entry.KeepReason = "raw_output_parent_apply_mode_disabled"
		entry.FailClosedReason = "raw_output_parent_apply_mode_disabled"
	} else {
		entry.KeepReason = "raw_output_artifacts_apply_mode_disabled"
		entry.FailClosedReason = "raw_output_artifacts_apply_mode_disabled"
	}
	recordArtifactBackedKept(report, entry.KeepReason)
	report.Kept = append(report.Kept, *entry)
}

func applyProviderHistoryArtifactBackedCommandReplacementCandidate(report *CommandEditDryRunReport, policy Policy, candidateIndex int, projection []api.Message) {
	if report == nil || candidateIndex < 0 || candidateIndex >= len(report.Candidates) {
		return
	}
	candidate := report.Candidates[candidateIndex]
	if !candidate.ArtifactBackedApplyEligible || candidate.SuggestedReplacementText == "" {
		return
	}
	if candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(projection) {
		return
	}
	if !providerHistoryCommandProjectionMessageMatchesCandidate(projection[candidate.HistoryIndex], candidate) {
		return
	}
	if !rawOutputArtifactMaterializationAllowed(policy) {
		reason := providerHistoryRawOutputMaterializationReadOnlyReason
		report.Candidates[candidateIndex].KeepReason = reason
		report.Candidates[candidateIndex].FailClosedReason = reason
		recordArtifactBackedKept(report, reason)
		return
	}
	ref, reason, ok := providerHistoryRawOutputRefForCandidate(report.RawOutputRefs, candidate.RawOutputRefID)
	if !ok {
		report.Candidates[candidateIndex].KeepReason = reason
		report.Candidates[candidateIndex].FailClosedReason = reason
		recordArtifactBackedKept(report, reason)
		return
	}
	if policy.RawOutputArtifactStore == nil {
		reason := "raw_output_artifact_missing"
		report.Candidates[candidateIndex].KeepReason = reason
		report.Candidates[candidateIndex].FailClosedReason = reason
		recordArtifactBackedKept(report, reason)
		return
	}
	verifyResult, err := policy.RawOutputArtifactStore.Verify(context.Background(), ref)
	if err != nil || !verifyResult.OK {
		reason := providerHistoryRawOutputVerifyFailureReason(verifyResult, err)
		report.Candidates[candidateIndex].KeepReason = reason
		report.Candidates[candidateIndex].FailClosedReason = reason
		recordArtifactBackedKept(report, reason)
		return
	}

	replacementText := candidate.SuggestedReplacementText
	savedBytes, savedTokens, thresholdStatus, thresholdOK := providerHistoryArtifactBackedReplacementEligibility(projection[candidate.HistoryIndex].Content, replacementText)
	if !thresholdOK {
		report.Candidates[candidateIndex].KeepReason = thresholdStatus
		report.Candidates[candidateIndex].FailClosedReason = thresholdStatus
		recordArtifactBackedKept(report, thresholdStatus)
		return
	}

	applyProviderHistoryCommandReplacementProjection(&projection[candidate.HistoryIndex], candidate, replacementText)
	report.Candidates[candidateIndex].ReplacementApplied = true
	report.ArtifactBackedCommandReplacedCount++
	report.ArtifactBackedCommandReplacementSavedBytes += savedBytes
	report.ApproxArtifactBackedCommandReplacementSavedTokens += savedTokens
	report.ReplacementStatus = providerHistoryCommandEditReplacementStatusPartialApply
}
