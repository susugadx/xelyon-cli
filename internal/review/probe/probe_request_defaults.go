package probe

func normalizeProbeRequestExecutionLimits(req ReviewProbeRequest) ReviewProbeRequest {
	if req.Timeout <= 0 {
		req.Timeout = defaultReviewProbeTimeout
	}
	if req.MaxOutputBytes <= 0 {
		req.MaxOutputBytes = defaultReviewProbeMaxOutputBytes
	}
	return req
}
