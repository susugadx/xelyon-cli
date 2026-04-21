package navigation

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func TestApplySnippetReferenceHints_BranchMatrix(t *testing.T) {
	t.Run("empty snippet or symbol keeps existing values", func(t *testing.T) {
		class, nodeType, selectorKind, receiverType := applySnippetReferenceHints("", "Build", ast.ClassRef, "node", "kind", "recv")
		if class != ast.ClassRef || nodeType != "node" || selectorKind != "kind" || receiverType != "recv" {
			t.Fatalf("applySnippetReferenceHints(empty snippet) = (%v, %q, %q, %q)", class, nodeType, selectorKind, receiverType)
		}

		class, nodeType, selectorKind, receiverType = applySnippetReferenceHints("Build()", "", ast.ClassRef, "node", "kind", "recv")
		if class != ast.ClassRef || nodeType != "node" || selectorKind != "kind" || receiverType != "recv" {
			t.Fatalf("applySnippetReferenceHints(empty symbol) = (%v, %q, %q, %q)", class, nodeType, selectorKind, receiverType)
		}
	})

	t.Run("bare call becomes identifier call", func(t *testing.T) {
		class, nodeType, selectorKind, receiverType := applySnippetReferenceHints("return Build()", "Build", ast.ClassUnknown, "", "", "")
		if class != ast.ClassCall {
			t.Fatalf("class = %v, want %v", class, ast.ClassCall)
		}
		if nodeType != "identifier" {
			t.Fatalf("nodeType = %q, want %q", nodeType, "identifier")
		}
		if selectorKind != "" || receiverType != "" {
			t.Fatalf("selector/receiver = (%q, %q), want empty", selectorKind, receiverType)
		}
	})

	t.Run("selector ref infers method receiver", func(t *testing.T) {
		class, nodeType, selectorKind, receiverType := applySnippetReferenceHints("return Client.Build", "Build", ast.ClassUnknown, "", "", "")
		if class != ast.ClassRef {
			t.Fatalf("class = %v, want %v", class, ast.ClassRef)
		}
		if nodeType != "field_identifier" {
			t.Fatalf("nodeType = %q, want %q", nodeType, "field_identifier")
		}
		if selectorKind != "method" {
			t.Fatalf("selectorKind = %q, want %q", selectorKind, "method")
		}
		if receiverType != "Client" {
			t.Fatalf("receiverType = %q, want %q", receiverType, "Client")
		}
	})

	t.Run("package selector stays package call", func(t *testing.T) {
		class, _, selectorKind, receiverType := applySnippetReferenceHints("return pkg.Build()", "Build", ast.ClassUnknown, "", "", "")
		if class != ast.ClassCall {
			t.Fatalf("class = %v, want %v", class, ast.ClassCall)
		}
		if selectorKind != "package" {
			t.Fatalf("selectorKind = %q, want %q", selectorKind, "package")
		}
		if receiverType != "" {
			t.Fatalf("receiverType = %q, want empty", receiverType)
		}
	})

	t.Run("definition snippet becomes class def", func(t *testing.T) {
		class, _, _, _ := applySnippetReferenceHints("func Build() {}", "Build", ast.ClassUnknown, "", "", "")
		if class != ast.ClassDef {
			t.Fatalf("class = %v, want %v", class, ast.ClassDef)
		}
	})

	t.Run("plain symbol mention becomes ref without overriding preset selector data", func(t *testing.T) {
		class, nodeType, selectorKind, receiverType := applySnippetReferenceHints("Build result", "Build", ast.ClassUnknown, "", "method", "Client")
		if class != ast.ClassRef {
			t.Fatalf("class = %v, want %v", class, ast.ClassRef)
		}
		if nodeType != "identifier" {
			t.Fatalf("nodeType = %q, want %q", nodeType, "identifier")
		}
		if selectorKind != "method" || receiverType != "Client" {
			t.Fatalf("selector/receiver = (%q, %q), want preserved values", selectorKind, receiverType)
		}
	})

	t.Run("non-promotable classes remain unchanged", func(t *testing.T) {
		class, nodeType, selectorKind, receiverType := applySnippetReferenceHints("Build()", "Build", ast.ClassComment, "", "", "")
		if class != ast.ClassComment {
			t.Fatalf("class = %v, want %v", class, ast.ClassComment)
		}
		if nodeType != "identifier" {
			t.Fatalf("nodeType = %q, want %q", nodeType, "identifier")
		}
		if selectorKind != "" || receiverType != "" {
			t.Fatalf("selector/receiver = (%q, %q), want empty", selectorKind, receiverType)
		}
	})
}

func TestSnippetOperandHelpers(t *testing.T) {
	if !containsBareSymbolCall("return Build()", "Build") {
		t.Fatal("containsBareSymbolCall() should detect bare call")
	}
	if containsBareSymbolCall("return pkg.Build()", "Build") {
		t.Fatal("containsBareSymbolCall() should ignore selector call")
	}

	if !looksLikePackageOperand("pkg") {
		t.Fatal("looksLikePackageOperand(pkg) = false, want true")
	}
	if looksLikePackageOperand("Client") {
		t.Fatal("looksLikePackageOperand(Client) = true, want false")
	}
	if looksLikePackageOperand("pkg.sub") {
		t.Fatal("looksLikePackageOperand(pkg.sub) = true, want false")
	}
	if looksLikePackageOperand("") {
		t.Fatal("looksLikePackageOperand(\"\") = true, want false")
	}
	if looksLikePackageOperand("pkg[0]") {
		t.Fatal("looksLikePackageOperand(pkg[0]) = true, want false")
	}
	if !looksLikePackageOperand("(&pkg)") {
		t.Fatal("looksLikePackageOperand((&pkg)) = false, want true after trim")
	}
	if looksLikePackageOperand("1pkg") {
		t.Fatal("looksLikePackageOperand(1pkg) = true, want false")
	}

	if got := inferReceiverTypeFromSnippetOperand("&Client{}"); got != "Client" {
		t.Fatalf("inferReceiverTypeFromSnippetOperand(&Client{}) = %q, want %q", got, "Client")
	}
	if got := inferReceiverTypeFromSnippetOperand("&Client"); got != "Client" {
		t.Fatalf("inferReceiverTypeFromSnippetOperand(&Client) = %q, want %q", got, "Client")
	}
	if got := inferReceiverTypeFromSnippetOperand("client"); got != "" {
		t.Fatalf("inferReceiverTypeFromSnippetOperand(client) = %q, want empty", got)
	}
	if got := inferReceiverTypeFromSnippetOperand("pkg.Client"); got != "" {
		t.Fatalf("inferReceiverTypeFromSnippetOperand(pkg.Client) = %q, want empty", got)
	}
}

func TestParseRipgrepLine_UnsupportedFileStillCompletesBySnippet(t *testing.T) {
	cache := newReferenceParseCache()
	ref := parseRipgrepLine("notes.txt:3:return pkg.Build()", "Build", cache)
	if ref == nil {
		t.Fatal("parseRipgrepLine() returned nil")
	}
	if ref.Class != ast.ClassCall {
		t.Fatalf("class = %v, want %v", ref.Class, ast.ClassCall)
	}
	if ref.NodeType != "field_identifier" {
		t.Fatalf("nodeType = %q, want %q", ref.NodeType, "field_identifier")
	}
	if ref.SelectorKind != "package" {
		t.Fatalf("selectorKind = %q, want %q", ref.SelectorKind, "package")
	}
}
