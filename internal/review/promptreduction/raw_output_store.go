package promptreduction

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

// ReviewRawOutputArtifactStore は review prompt reduction が使う rawoutputs store 境界。
type ReviewRawOutputArtifactStore interface {
	Create(ctx context.Context, req rawoutputs.CreateRequest) (rawoutputs.CreateResult, error)
	Verify(ctx context.Context, ref rawoutputs.RawOutputRef) (rawoutputs.VerifyResult, error)
	Resolve(ctx context.Context, ref rawoutputs.RawOutputRef) (rawoutputs.ResolvedArtifact, error)
}
