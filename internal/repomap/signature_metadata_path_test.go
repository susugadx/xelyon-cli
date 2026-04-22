package repomap

import "testing"

func TestSignatureMetadataForPath_TypeScriptArrowFunction(t *testing.T) {
	gotName, gotKind, exported := signatureMetadataForPath("app.ts", "const buildMap = async (): Promise<Map<string, string>> =>")
	if gotName != "buildMap" {
		t.Fatalf("name = %q, want %q", gotName, "buildMap")
	}
	if gotKind != "function" {
		t.Fatalf("kind = %q, want %q", gotKind, "function")
	}
	if exported {
		t.Fatal("exported = true, want false")
	}
}

func TestIsExportedName(t *testing.T) {
	if !isExportedName("Builder") {
		t.Fatal("isExportedName(\"Builder\") = false, want true")
	}
	if isExportedName("builder") {
		t.Fatal("isExportedName(\"builder\") = true, want false")
	}
	if isExportedName("") {
		t.Fatal("isExportedName(\"\") = true, want false")
	}
}
