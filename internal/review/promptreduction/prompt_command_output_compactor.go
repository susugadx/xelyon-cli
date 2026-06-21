package promptreduction

import (
	"github.com/susugadx/xelyon-cli/internal/commandoutputs"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
)

const reviewPromptCommandOutputReplacementMinSavedTokens = 128

type reviewPromptCommandOutputCompactorOptions struct {
	mode  ReviewPromptReductionMode
	stats *Stats
}

// NewReviewPromptCommandOutputCompactor は command output compaction 境界を構築する。
func NewReviewPromptCommandOutputCompactor(mode ReviewPromptReductionMode, stats *Stats) reviewmodelinput.CommandOutputCompactor {
	return reviewPromptCommandOutputCompactor{opts: reviewPromptCommandOutputCompactorOptions{
		mode:  NormalizeReviewPromptReductionMode(mode),
		stats: stats,
	}}
}

type reviewPromptCommandOutputCompactor struct {
	opts reviewPromptCommandOutputCompactorOptions
}

func (c reviewPromptCommandOutputCompactor) CompactCommandOutput(command, output string) (reviewmodelinput.CommandOutputCompactResult, bool) {
	replacement, _, ok := commandoutputs.BuildReplacement(commandoutputs.NewRequest(command, output))
	if !ok {
		return reviewmodelinput.CommandOutputCompactResult{}, false
	}
	if replacement.SavedBytes() <= 0 || replacement.SavedTokens() < reviewPromptCommandOutputReplacementMinSavedTokens {
		return reviewmodelinput.CommandOutputCompactResult{}, false
	}
	applied := c.opts.mode.compactCommandOutputs()
	c.opts.stats.RecordCandidate(replacement.Classifier(), replacement.SavedBytes(), replacement.SavedTokens(), applied)
	if !applied {
		return reviewmodelinput.CommandOutputCompactResult{}, false
	}
	return reviewmodelinput.CommandOutputCompactResult{
		Output:     replacement.Text(),
		Classifier: replacement.Classifier(),
	}, true
}
