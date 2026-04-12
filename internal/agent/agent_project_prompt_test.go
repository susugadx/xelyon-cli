package agent

import (
	"reflect"
	"testing"
)

func TestExtractProjectMapPathsFromInput_QuotedFilename(t *testing.T) {
	t.Parallel()

	input := "please inspect 'design spec.md' and then compare with \"notes.txt\""

	got := extractProjectMapPathsFromInput(input)
	want := []string{"design spec.md", "notes.txt"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractProjectMapPathsFromInput() = %#v, want %#v", got, want)
	}
}
