package agent

import (
	"testing"
)

func TestContainsString_Found(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		s     string
		want  bool
	}{
		{
			name:  "first element",
			slice: []string{"a", "b", "c"},
			s:     "a",
			want:  true,
		},
		{
			name:  "middle element",
			slice: []string{"a", "b", "c"},
			s:     "b",
			want:  true,
		},
		{
			name:  "last element",
			slice: []string{"a", "b", "c"},
			s:     "c",
			want:  true,
		},
		{
			name:  "single element slice",
			slice: []string{"only"},
			s:     "only",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsString(tt.slice, tt.s)
			if got != tt.want {
				t.Errorf("containsString(%v, %q) = %v, want %v", tt.slice, tt.s, got, tt.want)
			}
		})
	}
}

func TestContainsString_NotFound(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		s     string
	}{
		{
			name:  "not in slice",
			slice: []string{"a", "b", "c"},
			s:     "d",
		},
		{
			name:  "case sensitive mismatch",
			slice: []string{"A", "B", "C"},
			s:     "a",
		},
		{
			name:  "partial match not counted",
			slice: []string{"abc", "def"},
			s:     "ab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsString(tt.slice, tt.s)
			if got {
				t.Errorf("containsString(%v, %q) = true, want false", tt.slice, tt.s)
			}
		})
	}
}

func TestContainsString_EmptySlice(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		s     string
	}{
		{
			name:  "empty slice",
			slice: []string{},
			s:     "a",
		},
		{
			name:  "nil slice",
			slice: nil,
			s:     "a",
		},
		{
			name:  "empty slice with empty string",
			slice: []string{},
			s:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsString(tt.slice, tt.s)
			if got {
				t.Errorf("containsString(%v, %q) = true, want false", tt.slice, tt.s)
			}
		})
	}
}

func TestContainsString_EmptyString(t *testing.T) {
	// Test when looking for empty string in slice that contains empty string
	slice := []string{"a", "", "b"}
	got := containsString(slice, "")
	if !got {
		t.Errorf("containsString(%v, \"\") = false, want true", slice)
	}
}
