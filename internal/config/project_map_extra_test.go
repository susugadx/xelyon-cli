package config

import (
	"math"
	"testing"
)

func TestNormalizeProjectMapContextRatio(t *testing.T) {
	tests := []struct {
		name  string
		ratio float64
		want  float64
	}{
		{
			name:  "valid mid-range",
			ratio: 0.10,
			want:  0.10,
		},
		{
			name:  "valid exact min",
			ratio: ProjectMapContextRatioMin,
			want:  ProjectMapContextRatioMin,
		},
		{
			name:  "valid exact max",
			ratio: ProjectMapContextRatioMax,
			want:  ProjectMapContextRatioMax,
		},
		{
			name:  "valid 0.05",
			ratio: 0.05,
			want:  0.05,
		},
		{
			name:  "too small zero",
			ratio: 0.0,
			want:  ProjectMapContextRatioDefault,
		},
		{
			name:  "too small below min",
			ratio: 0.005,
			want:  ProjectMapContextRatioDefault,
		},
		{
			name:  "too large above max",
			ratio: 0.5,
			want:  ProjectMapContextRatioDefault,
		},
		{
			name:  "too large 1.0",
			ratio: 1.0,
			want:  ProjectMapContextRatioDefault,
		},
		{
			name:  "negative",
			ratio: -0.1,
			want:  ProjectMapContextRatioDefault,
		},
		{
			name:  "NaN",
			ratio: math.NaN(),
			want:  ProjectMapContextRatioDefault,
		},
		{
			name:  "positive infinity",
			ratio: math.Inf(1),
			want:  ProjectMapContextRatioDefault,
		},
		{
			name:  "negative infinity",
			ratio: math.Inf(-1),
			want:  ProjectMapContextRatioDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeProjectMapContextRatio(tt.ratio)
			if got != tt.want {
				t.Errorf("NormalizeProjectMapContextRatio(%v) = %v, want %v", tt.ratio, got, tt.want)
			}
		})
	}
}
