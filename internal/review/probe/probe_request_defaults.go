package probe

import "time"

const (
	defaultReviewProbeTimeout        = 30 * time.Second
	defaultReviewProbeMaxOutputBytes = 64 * 1024
)

func normalizeProbeRequestExecutionLimits(req ReviewProbeRequest) ReviewProbeRequest {
	if req.Timeout <= 0 {
		req.Timeout = defaultReviewProbeTimeout
	}
	if req.MaxOutputBytes <= 0 {
		req.MaxOutputBytes = defaultReviewProbeMaxOutputBytes
	}
	return req
}
