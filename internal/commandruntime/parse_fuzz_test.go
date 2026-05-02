package commandruntime

import "testing"

func FuzzSplitStrict(f *testing.F) {
	seeds := []string{
		"/attach \"C:\\Users\\me\\file with space.txt\"",
		"/attach \"\\\\server\\share\\file name.txt\"",
		"foo'bar baz'",
		"don't panic",
		`/attach "unterminated`,
		`/attach 'unterminated`,
		`/attach "escaped \"quote\""`,
		"/review",
		"/copy code -n 2",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		parts, status := SplitStrict(input)
		legacy := Split(input)

		if len(parts) != len(legacy) {
			t.Fatalf("Split mismatch: strict=%#v legacy=%#v", parts, legacy)
		}
		for i := range parts {
			if parts[i] != legacy[i] {
				t.Fatalf("Split token mismatch at %d: strict=%q legacy=%q", i, parts[i], legacy[i])
			}
		}

		switch status {
		case SplitStatusOK, SplitStatusUnterminatedQuote:
		default:
			t.Fatalf("unexpected split status: %v", status)
		}
	})
}
