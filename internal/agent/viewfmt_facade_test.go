package agent

import "testing"

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{
			name: "less than 1000",
			n:    999,
			want: "999",
		},
		{
			name: "exactly 1000",
			n:    1000,
			want: "1,000",
		},
		{
			name: "ten thousand",
			n:    10000,
			want: "10,000",
		},
		{
			name: "hundred thousand",
			n:    100000,
			want: "100,000",
		},
		{
			name: "one million",
			n:    1000000,
			want: "1,000,000",
		},
		{
			name: "arbitrary large number",
			n:    1234567,
			want: "1,234,567",
		},
		{
			name: "zero",
			n:    0,
			want: "0",
		},
		{
			name: "small number",
			n:    42,
			want: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNumber(tt.n)
			if got != tt.want {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}
