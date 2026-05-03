package tui

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	pdf "rsc.io/pdf"
)

func TestReadAttachedPDFPreview_RealPDFFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "sample.pdf", buildMinimalPDFWithSingleText("Hello PDF"))

	preview, err := readAttachedPDFPreview(path)
	if err != nil {
		t.Fatalf("readAttachedPDFPreview() error = %v, want nil", err)
	}
	if preview.truncated {
		t.Fatal("preview.truncated = true, want false")
	}
	if strings.TrimSpace(preview.text) == "" {
		t.Fatal("preview.text = empty, want non-empty")
	}
	if !strings.Contains(strings.ReplaceAll(preview.text, " ", ""), "HelloPDF") {
		t.Fatalf("preview.text = %q, want to include HelloPDF", preview.text)
	}
}

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

func buildMinimalPDFWithSingleText(text string) []byte {
	stream := fmt.Sprintf("BT\n/F1 24 Tf\n72 72 Td\n(%s) Tj\nET\n", escapePDFLiteralString(text))
	objects := []string{
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n",
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n",
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 300 144] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n",
		fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%sendstream\nendobj\n", len(stream), stream),
		"5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n",
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = out.Len()
		out.WriteString(obj)
	}

	xrefOffset := out.Len()
	out.WriteString("xref\n")
	fmt.Fprintf(&out, "0 %d\n", len(objects)+1)
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	out.WriteString("trailer\n")
	fmt.Fprintf(&out, "<< /Size %d /Root 1 0 R >>\n", len(objects)+1)
	out.WriteString("startxref\n")
	fmt.Fprintf(&out, "%d\n", xrefOffset)
	out.WriteString("%%EOF\n")

	return out.Bytes()
}

func escapePDFLiteralString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
