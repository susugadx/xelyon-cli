package uitoolview

import "testing"

func TestSpinnerMessageForTool(t *testing.T) {
	tests := []struct {
		toolName string
		want     string
	}{
		{toolName: "write_file", want: "Writing file..."},
		{toolName: "str_replace", want: "Editing file..."},
		{toolName: "bash", want: "Preparing..."},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			got := SpinnerMessageForTool(tt.toolName)
			if got != tt.want {
				t.Errorf("SpinnerMessageForTool(%q) = %q; want %q", tt.toolName, got, tt.want)
			}
		})
	}
}
