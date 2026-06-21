package probe

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestBuildProbeCommandTransition(t *testing.T) {
	tests := []struct {
		name          string
		status        domain.ReviewProbeStatus
		wantStop      bool
		wantStatus    domain.ReviewProbeStatus
		wantErrorHead string
	}{
		{
			name:          "blocked",
			status:        domain.ReviewProbeBlocked,
			wantStop:      true,
			wantStatus:    domain.ReviewProbeBlocked,
			wantErrorHead: "probe command blocked:",
		},
		{
			name:          "timed out",
			status:        domain.ReviewProbeTimedOut,
			wantStop:      true,
			wantStatus:    domain.ReviewProbeTimedOut,
			wantErrorHead: "probe command timed out:",
		},
		{
			name:          "failed",
			status:        domain.ReviewProbeFailed,
			wantStop:      true,
			wantStatus:    domain.ReviewProbeFailed,
			wantErrorHead: "probe command failed:",
		},
		{
			name:       "passed",
			status:     domain.ReviewProbePassed,
			wantStop:   false,
			wantStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, message, stop := buildProbeCommandTransition(tt.status, "git", []string{"status", "--short"})
			if stop != tt.wantStop {
				t.Fatalf("stop = %v, want %v", stop, tt.wantStop)
			}
			if next != tt.wantStatus {
				t.Fatalf("next = %q, want %q", next, tt.wantStatus)
			}
			if tt.wantErrorHead == "" {
				if message != "" {
					t.Fatalf("message = %q, want empty", message)
				}
				return
			}
			if !strings.HasPrefix(message, tt.wantErrorHead) {
				t.Fatalf("message = %q, want prefix %q", message, tt.wantErrorHead)
			}
		})
	}
}
