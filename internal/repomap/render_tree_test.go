package repomap

import (
	"strings"
	"testing"
)

func TestWriteRenderedSymbol_MultiLineSignature(t *testing.T) {
	var b strings.Builder
	writeRenderedSymbol(&b, "  ", Symbol{
		Line:      10,
		EndLine:   12,
		Signature: "func Build(\n\tctx context.Context,\n) error",
	})

	rendered := b.String()
	if !strings.Contains(rendered, "10-12: func Build(") {
		t.Fatalf("writeRenderedSymbol() missing location header, got %q", rendered)
	}
	if !strings.Contains(rendered, "\tctx context.Context,") {
		t.Fatalf("writeRenderedSymbol() missing multiline continuation, got %q", rendered)
	}
}
