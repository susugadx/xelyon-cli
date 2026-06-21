package review

import (
	"context"
	"errors"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

type runnerFakeEvidenceBuilder struct {
	bundle reviewevidence.ReviewEvidenceBundle
	err    error
	calls  int
	events *[]string
}

func (b *runnerFakeEvidenceBuilder) BuildCurrentChanges(context.Context) (reviewevidence.ReviewEvidenceBundle, error) {
	b.calls++
	if b.events != nil {
		*b.events = append(*b.events, "evidence")
	}
	if b.err != nil {
		return reviewevidence.ReviewEvidenceBundle{}, b.err
	}
	return b.bundle, nil
}

type runnerFakeProbeRunner struct {
	results map[string]reviewprobe.ReviewProbeResult
	errors  map[string]error
	calls   []reviewprobe.ReviewProbeRequest
	events  *[]string
}

func (r *runnerFakeProbeRunner) Run(_ context.Context, req reviewprobe.ReviewProbeRequest) (reviewprobe.ReviewProbeResult, error) {
	r.calls = append(r.calls, req)
	if r.events != nil {
		*r.events = append(*r.events, "probe:"+req.ID)
	}
	if err := r.errors[req.ID]; err != nil {
		return reviewprobe.ReviewProbeResult{}, err
	}
	result, ok := r.results[req.ID]
	if !ok {
		return reviewprobe.ReviewProbeResult{
			ID:     req.ID,
			Mode:   req.Mode,
			Status: domain.ReviewProbePassed,
		}, nil
	}
	if result.ID == "" {
		result.ID = req.ID
	}
	if result.Mode == "" {
		result.Mode = req.Mode
	}
	if result.Status == "" {
		result.Status = domain.ReviewProbePassed
	}
	return result, nil
}

type runnerFakeModel struct {
	responses []runnerFakeModelResponse
	requests  []ReviewModelRequest
	events    *[]string
}

type runnerFakeModelResponse struct {
	content string
	err     error
}

func (m *runnerFakeModel) CompleteReview(_ context.Context, req ReviewModelRequest) (ReviewModelResponse, error) {
	m.requests = append(m.requests, req)
	if m.events != nil {
		*m.events = append(*m.events, "model:"+string(req.Phase))
	}
	index := len(m.requests) - 1
	if index >= len(m.responses) {
		return ReviewModelResponse{}, errors.New("unexpected review model call")
	}
	response := m.responses[index]
	if response.err != nil {
		return ReviewModelResponse{}, response.err
	}
	return ReviewModelResponse{Content: response.content}, nil
}
