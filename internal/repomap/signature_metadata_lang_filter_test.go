package repomap

import "testing"

func TestExtractSignatureMetadataForLang_RespectsLanguageFilter(t *testing.T) {
	if name, kind, ok := ExtractSignatureMetadataForLang("export function buildMap() {}", "js"); !ok || name != "buildMap" || kind != "function" {
		t.Fatalf("ExtractSignatureMetadataForLang(js) = (%q, %q, %v), want buildMap/function/true", name, kind, ok)
	}
	if name, kind, ok := ExtractSignatureMetadataForLang("type BuildOptions = { verbose: boolean }", "js"); !ok || name != "BuildOptions" || kind != "type" {
		t.Fatalf("ExtractSignatureMetadataForLang(js type alias) = (%q, %q, %v), want BuildOptions/type/true", name, kind, ok)
	}
	if name, kind, ok := ExtractSignatureMetadataForLang("export  const buildMap = () => {}", "js"); !ok || name != "buildMap" || kind != "function" {
		t.Fatalf("ExtractSignatureMetadataForLang(js spaced arrow) = (%q, %q, %v), want buildMap/function/true", name, kind, ok)
	}
	if name, kind, ok := ExtractSignatureMetadataForLang("const\tbuildMap = () => {}", "js"); !ok || name != "buildMap" || kind != "function" {
		t.Fatalf("ExtractSignatureMetadataForLang(js tabbed arrow) = (%q, %q, %v), want buildMap/function/true", name, kind, ok)
	}
	if _, _, ok := ExtractSignatureMetadataForLang("export function buildMap() {}", "go"); ok {
		t.Fatal("ExtractSignatureMetadataForLang(go) unexpectedly matched JS signature")
	}
	if name, kind, ok := ExtractSignatureMetadata("type Builder struct{}"); !ok || name != "Builder" || kind != "struct" {
		t.Fatalf("ExtractSignatureMetadata() = (%q, %q, %v), want Builder/struct/true", name, kind, ok)
	}
}
