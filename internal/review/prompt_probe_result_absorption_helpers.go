package review

import (
	"strings"

	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
)

func reviewProbeResultAbsorptionKeepReason(build reviewProbeRawOutputBuild) string {
	if strings.TrimSpace(build.disabledReason) != "" {
		return build.disabledReason
	}
	if strings.TrimSpace(build.ledger.FailClosedReason) != "" {
		return build.ledger.FailClosedReason
	}
	return reviewpromptreduction.ReviewProbeRawOutputReasonRehydrateUnavailable
}
