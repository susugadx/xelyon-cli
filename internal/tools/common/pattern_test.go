package common

import (
	"reflect"
	"testing"
)

func TestFindPatternInLines(t *testing.T) {
	tests := []struct {
		name    string
		lines   []string
		pattern string
		want    PatternMatchResult
	}{
		{
			name:    "exact match",
			lines:   []string{"hello", "world", "test"},
			pattern: "world",
			want: PatternMatchResult{
				MatchIdx:     1,
				MatchCount:   1,
				MatchIndices: []int{1},
			},
		},
		{
			name:    "normalized match (leading whitespace)",
			lines:   []string{"  hello", "\tworld", "test"},
			pattern: "world",
			want: PatternMatchResult{
				MatchIdx:     1,
				MatchCount:   1,
				MatchIndices: []int{1},
			},
		},
		{
			name:    "not found",
			lines:   []string{"hello", "world"},
			pattern: "missing",
			want: PatternMatchResult{
				MatchIdx:     -1,
				MatchCount:   0,
				MatchIndices: nil,
			},
		},
		{
			name:    "multiple exact matches",
			lines:   []string{"hello", "world", "hello", "test"},
			pattern: "hello",
			want: PatternMatchResult{
				MatchIdx:     0,
				MatchCount:   2,
				MatchIndices: []int{0, 2},
			},
		},
		{
			name:    "multiple normalized matches",
			lines:   []string{"  hello", "\thello", "world"},
			pattern: "hello",
			want: PatternMatchResult{
				MatchIdx:     0,
				MatchCount:   2,
				MatchIndices: []int{0, 1},
			},
		},
		{
			name:    "exact match takes precedence over normalized",
			lines:   []string{"  hello", "hello"},
			pattern: "hello",
			want: PatternMatchResult{
				MatchIdx:     1,
				MatchCount:   1,
				MatchIndices: []int{1},
			},
		},
		{
			name:    "empty lines array",
			lines:   []string{},
			pattern: "anything",
			want: PatternMatchResult{
				MatchIdx:     -1,
				MatchCount:   0,
				MatchIndices: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindPatternInLines(tt.lines, tt.pattern)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindPatternInLines() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDisplayPatternNotFound(t *testing.T) {
	// このテストは主に空のlines配列でもpanicしないことを確認する
	t.Run("empty lines array", func(t *testing.T) {
		// 標準出力をキャプチャする代わりに、panicしないことを確認
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DisplayPatternNotFound panicked with empty lines: %v", r)
			}
		}()

		// 空のlines配列で関数を呼び出し
		DisplayPatternNotFound("test pattern", []string{}, 50)
	})

	t.Run("normal lines array", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DisplayPatternNotFound panicked: %v", r)
			}
		}()

		DisplayPatternNotFound("test pattern", []string{"line1", "line2", "line3"}, 2)
	})
}
