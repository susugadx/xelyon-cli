package review

import (
	"context"
	"fmt"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
)

func (r *ReviewRunner) completeReviewProbePlan(ctx context.Context, req ReviewRequest, evidenceMarkdown string, bundle ReviewEvidenceBundle) (ReviewProbePlan, error) {
	planPrompt := reviewmodelinput.BuildProbePlanPrompt(reviewmodelinput.ProbePlanPromptInput{
		CustomInstructions: req.CustomInstructions,
		EvidenceMarkdown:   evidenceMarkdown,
	})
	redactor := newReviewRunnerPromptRedactor(bundle, nil)
	r.saveReviewRunTextArtifact("probe_plan_prompt.md", planPrompt, redactor)
	planResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseProbePlan,
		Prompt: planPrompt,
	})
	if err != nil {
		return ReviewProbePlan{}, fmt.Errorf("review runner pass1 model: %w", err)
	}
	r.saveReviewRunTextArtifact("probe_plan_raw.json", planResp.Content, redactor)

	plan, decodeErr := decodeReviewProbePlanJSONAgainstEvidence(planResp.Content, bundle)
	if decodeErr == nil {
		r.saveReviewRunJSONArtifact("probe_plan_final.json", plan, redactor)
		return plan, nil
	}

	repairPrompt := reviewmodelinput.BuildProbePlanRepairPrompt(reviewmodelinput.ProbePlanRepairPromptInput{
		CustomInstructions:    req.CustomInstructions,
		EvidenceMarkdown:      evidenceMarkdown,
		InvalidOutput:         planResp.Content,
		DecodeOrValidationErr: decodeErr,
	})
	r.saveReviewRunTextArtifact("probe_plan_prompt.md", repairPrompt, redactor)
	repairResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseProbePlan,
		Prompt: repairPrompt,
	})
	if err != nil {
		return ReviewProbePlan{}, fmt.Errorf("review runner pass1 model: %w", err)
	}
	r.saveReviewRunTextArtifact("probe_plan_raw.json", repairResp.Content, redactor)

	plan, decodeErr = decodeReviewProbePlanJSONAgainstEvidence(repairResp.Content, bundle)
	if decodeErr != nil {
		return ReviewProbePlan{}, fmt.Errorf("review runner decode probe plan: %w", decodeErr)
	}
	r.saveReviewRunJSONArtifact("probe_plan_final.json", plan, redactor)
	return plan, nil
}

func decodeReviewProbePlanJSONAgainstEvidence(content string, bundle ReviewEvidenceBundle) (ReviewProbePlan, error) {
	plan, err := DecodeReviewProbePlanJSON([]byte(content))
	if err != nil {
		return ReviewProbePlan{}, err
	}
	if err := ValidateReviewProbePlanAgainstEvidence(plan, bundle); err != nil {
		return ReviewProbePlan{}, fmt.Errorf("ValidateReviewProbePlanAgainstEvidence: %w", err)
	}
	return plan, nil
}
