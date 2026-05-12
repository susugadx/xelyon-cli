package review

import (
	"context"
	"testing"
)

type fakeReviewModel struct {
	received ReviewModelRequest
	content  string
}

var _ ReviewModel = (*fakeReviewModel)(nil)

func (m *fakeReviewModel) CompleteReview(_ context.Context, req ReviewModelRequest) (ReviewModelResponse, error) {
	m.received = req
	return ReviewModelResponse{Content: m.content}, nil
}

func TestReviewModelContract(t *testing.T) {
	model := &fakeReviewModel{content: `{"schema_version":"review_probe_plan.v2"}`}
	req := ReviewModelRequest{
		Phase:  ReviewModelPhaseProbePlan,
		Prompt: "complete model input",
	}

	resp, err := model.CompleteReview(context.Background(), req)
	if err != nil {
		t.Fatalf("CompleteReview() error = %v, want nil", err)
	}
	if got, want := model.received.Phase, ReviewModelPhaseProbePlan; got != want {
		t.Fatalf("received Phase = %q, want %q", got, want)
	}
	if got, want := model.received.Prompt, "complete model input"; got != want {
		t.Fatalf("received Prompt = %q, want %q", got, want)
	}
	if got, want := resp.Content, `{"schema_version":"review_probe_plan.v2"}`; got != want {
		t.Fatalf("Content = %q, want %q", got, want)
	}
}
