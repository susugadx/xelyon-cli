package probe

import (
	"strings"
	"testing"
)

func TestBuildProbeCommandTransition(t *testing.T) {
	tests := []struct {
		name          string
		status        ReviewProbeStatus
		wantStop      bool
		wantStatus    ReviewProbeStatus
		wantErrorHead string
	}{
		{
			name:          "blocked",
			status:        ReviewProbeBlocked,
			wantStop:      true,
			wantStatus:    ReviewProbeBlocked,
			wantErrorHead: "probe command blocked:",
		},
		{
			name:          "timed out",
			status:        ReviewProbeTimedOut,
			wantStop:      true,
			wantStatus:    ReviewProbeTimedOut,
			wantErrorHead: "probe command timed out:",
		},
		{
			name:          "failed",
			status:        ReviewProbeFailed,
			wantStop:      true,
			wantStatus:    ReviewProbeFailed,
			wantErrorHead: "probe command failed:",
		},
		{
			name:       "passed",
			status:     ReviewProbePassed,
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
