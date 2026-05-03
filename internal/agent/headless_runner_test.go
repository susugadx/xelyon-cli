package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestIsHeadlessToolCallSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   tools.ExecutionResult
		want bool
	}{
		{
			name: "ok result",
			in: tools.ExecutionResult{
				Result: "done",
				Error:  false,
			},
			want: true,
		},
		{
			name: "error flag true",
			in: tools.ExecutionResult{
				Result: "done",
				Error:  true,
			},
			want: false,
		},
		{
			name: "error prefix with leading space",
			in: tools.ExecutionResult{
				Result: " Error: command failed",
				Error:  false,
			},
			want: false,
		},
		{
			name: "error marker in middle is not failure",
			in: tools.ExecutionResult{
				Result: "output contains Error: marker in the middle",
				Error:  false,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isHeadlessToolCallSuccess(tt.in)
			if got != tt.want {
				t.Fatalf("isHeadlessToolCallSuccess(%+v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
