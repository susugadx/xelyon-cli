package review

import (
	"errors"
)

func (r *ReviewRunner) validate() error {
	if r == nil {
		return errors.New("review runner is nil")
	}
	return validateReviewRunnerOptions(ReviewRunnerOptions{
		EvidenceBuilder: r.evidenceBuilder,
		ProbeRunner:     r.probeRunner,
		Model:           r.model,
	})
}

func validateReviewRunnerOptions(opts ReviewRunnerOptions) error {
	if opts.Model == nil {
		return errReviewRunnerModelNil
	}
	if opts.EvidenceBuilder == nil {
		return errReviewRunnerEvidenceBuilderNil
	}
	if opts.ProbeRunner == nil {
		return errReviewRunnerProbeRunnerNil
	}
	return nil
}
