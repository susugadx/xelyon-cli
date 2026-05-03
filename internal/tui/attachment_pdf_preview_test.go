package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	pdf "rsc.io/pdf"
)

func TestExtractAttachedPDFPreviewWithPageReader_PageLimitStopsRead(t *testing.T) {
	calls := 0
	preview, err := extractAttachedPDFPreviewWithPageReader(100, 20, 30000, func(pageIndex int) (string, error) {
		calls++
		return fmt.Sprintf("p%d", pageIndex), nil
	})
	if err != nil {
		t.Fatalf("extractAttachedPDFPreviewWithPageReader() error = %v, want nil", err)
	}
	if calls != 20 {
		t.Fatalf("read calls = %d, want 20", calls)
	}
	if !preview.truncated {
		t.Fatal("preview.truncated = false, want true")
	}
	if strings.Contains(preview.text, "p21") {
		t.Fatalf("preview.text should not include page 21: %q", preview.text)
	}
}

func TestExtractAttachedPDFPreviewWithPageReader_CharLimitStopsRead(t *testing.T) {
	calls := 0
	preview, err := extractAttachedPDFPreviewWithPageReader(10, 20, 3, func(pageIndex int) (string, error) {
		calls++
		return "abcdef", nil
	})
	if err != nil {
		t.Fatalf("extractAttachedPDFPreviewWithPageReader() error = %v, want nil", err)
	}
	if calls != 1 {
		t.Fatalf("read calls = %d, want 1", calls)
	}
	if !preview.truncated {
		t.Fatal("preview.truncated = false, want true")
	}
	if got, want := preview.text, "abc"; got != want {
		t.Fatalf("preview.text = %q, want %q", got, want)
	}
}

func TestReadPDFPageContentWithRecover(t *testing.T) {
	_, err := readPDFPageContentWithRecover(func() pdf.Content {
		panic("boom")
	})
	if err == nil {
		t.Fatal("readPDFPageContentWithRecover() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "panic while reading PDF page content") {
		t.Fatalf("error = %q, want panic marker", err.Error())
	}
}

func TestBuildPDFTextFromParts_ProducesReadableFlow(t *testing.T) {
	parts := []pdf.Text{
		{S: "World", X: 60, Y: 100, W: 20, FontSize: 10},
		{S: "Hello", X: 10, Y: 100, W: 40, FontSize: 10},
		{S: "Next", X: 10, Y: 80, W: 20, FontSize: 10},
	}
	got := buildPDFTextFromParts(parts)
	if got != "Hello World\nNext" {
		t.Fatalf("buildPDFTextFromParts() = %q, want %q", got, "Hello World\\nNext")
	}
}

func TestBuildAttachedPDFContext(t *testing.T) {
	prev := readAttachedPDFPreviewForContext
	t.Cleanup(func() {
		readAttachedPDFPreviewForContext = prev
	})

	t.Run("successful preview with truncation note", func(t *testing.T) {
		readAttachedPDFPreviewForContext = func(path string) (attachedPDFPreview, error) {
			return attachedPDFPreview{text: "line1\nline2", truncated: true}, nil
		}
		got := buildAttachedPDFContext("docs/spec.pdf")
		if !strings.Contains(got, "[Attached file: ") {
			t.Fatalf("context = %q, want attachment header", got)
		}
		if !strings.Contains(got, "line1\nline2") {
			t.Fatalf("context = %q, want preview text", got)
		}
		wantTrunc := "<PDF content truncated: first 20 pages / 30000 chars shown>"
		if !strings.Contains(got, wantTrunc) {
			t.Fatalf("context = %q, want %q", got, wantTrunc)
		}
	})

	t.Run("failed preview", func(t *testing.T) {
		readAttachedPDFPreviewForContext = func(path string) (attachedPDFPreview, error) {
			return attachedPDFPreview{}, errors.New("boom")
		}
		got := buildAttachedPDFContext("docs/spec.pdf")
		if !strings.Contains(got, "<failed to read PDF: boom>") {
			t.Fatalf("context = %q, want failed read marker", got)
		}
	})

	t.Run("no extractable text", func(t *testing.T) {
		readAttachedPDFPreviewForContext = func(path string) (attachedPDFPreview, error) {
			return attachedPDFPreview{text: " \n\t ", truncated: false}, nil
		}
		got := buildAttachedPDFContext("docs/spec.pdf")
		if !strings.Contains(got, "<no extractable text in PDF>") {
			t.Fatalf("context = %q, want no text marker", got)
		}
	})
}

func TestBuildAttachedFileContext_RoutesPDFExtension(t *testing.T) {
	prev := readAttachedPDFPreviewForContext
	readAttachedPDFPreviewForContext = func(path string) (attachedPDFPreview, error) {
		return attachedPDFPreview{text: "pdf-body", truncated: false}, nil
	}
	t.Cleanup(func() {
		readAttachedPDFPreviewForContext = prev
	})

	got := buildAttachedFileContext("README.PDF")
	if !strings.Contains(got, "pdf-body") {
		t.Fatalf("context = %q, want routed PDF preview body", got)
	}
}
