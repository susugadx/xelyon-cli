package attachments

import (
	"errors"
	"strings"
	"testing"
)

func TestContextBlockSpecsSkipsPrimaryImageAndKeepsFile(t *testing.T) {
	specs := ContextBlockSpecs([]Attachment{
		{Kind: KindImage, Path: "/tmp/primary.png"},
		{Kind: KindImage, Path: "/tmp/extra.png"},
		{Kind: KindFile, Path: "/tmp/notes.txt"},
	}, "/tmp/primary.png")

	if got, want := len(specs), 2; got != want {
		t.Fatalf("len(ContextBlockSpecs()) = %d, want %d", got, want)
	}
	if specs[0].Kind != ContextBlockImagePath || specs[0].Path != "/tmp/extra.png" {
		t.Fatalf("first spec = %#v, want extra image path", specs[0])
	}
	if specs[1].Kind != ContextBlockFile || specs[1].Path != "/tmp/notes.txt" {
		t.Fatalf("second spec = %#v, want file path", specs[1])
	}
}

func TestBuildAttachedContextBlocks(t *testing.T) {
	if got := BuildAttachedImagePathContext("img.png"); got != "[Attached image path]\nimg.png" {
		t.Fatalf("BuildAttachedImagePathContext() = %q", got)
	}

	truncated := BuildAttachedFileContextBlock("notes.txt", "body", true, false, nil, 64)
	if !strings.Contains(truncated, "<content truncated: first 64 bytes shown>") {
		t.Fatalf("truncated file block = %q, want truncation note", truncated)
	}
	binary := BuildAttachedFileContextBlock("bin.dat", "", false, true, nil, 64)
	if !strings.Contains(binary, "<binary file omitted>") {
		t.Fatalf("binary file block = %q, want binary marker", binary)
	}
	failed := BuildAttachedFileContextBlock("notes.txt", "", false, false, errors.New("boom"), 64)
	if !strings.Contains(failed, "<failed to read: boom>") {
		t.Fatalf("failed file block = %q, want error marker", failed)
	}
	pdf := BuildAttachedPDFContextBlock("spec.pdf", "body", true, nil, 20, 30000)
	if !strings.Contains(pdf, "<PDF content truncated: first 20 pages / 30000 chars shown>") {
		t.Fatalf("pdf block = %q, want PDF truncation note", pdf)
	}
	emptyPDF := BuildAttachedPDFContextBlock("spec.pdf", " \n\t ", false, nil, 20, 30000)
	if !strings.Contains(emptyPDF, "<no extractable text in PDF>") {
		t.Fatalf("empty pdf block = %q, want no text marker", emptyPDF)
	}
}
