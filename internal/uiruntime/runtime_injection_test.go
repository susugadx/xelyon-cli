package uiruntime

import (
	"bytes"
	"strings"
	"testing"
)

func TestPager_DisplayWithRuntimeUsesInjectedWriter(t *testing.T) {
	runtime := NewRuntime(strings.NewReader("q\n"), &bytes.Buffer{}, &bytes.Buffer{})
	out := runtime.Output().(*bytes.Buffer)

	pager := NewPagerWithRuntime(runtime)
	pager.pageSize = 2
	pager.Display("line1\nline2\nline3")

	output := out.String()
	if !strings.Contains(output, "line1") || !strings.Contains(output, "line2") {
		t.Fatalf("expected injected output to contain first page, got %q", output)
	}
	if !strings.Contains(output, "--more--") {
		t.Fatalf("expected injected output to contain pager prompt, got %q", output)
	}
	if strings.Contains(output, "line3") {
		t.Fatalf("expected quit input to stop before second page, got %q", output)
	}
}
