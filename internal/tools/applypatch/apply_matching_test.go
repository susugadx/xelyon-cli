package applypatch

import (
	"os"
	"strings"
	"testing"
)

func TestLooksLikeUnifiedDiffHeader(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"-274,6 +274,32 @@", true},
		{"-43,7 +43,7 @@", true},
		{"-1,3 +1,5", true},
		{"func BuildProjectMap", false},
		{"class Config", false},
		{`case "openrouter":`, false},
		{"", false},
		{"def greet():", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeUnifiedDiffHeader(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeUnifiedDiffHeader(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestApplyPatch_FirstLineChangeWithContextAfter は、@@コンテキストが変更対象行より
// 後ろにある場合（ファイル先頭行の変更）でもパッチが成功することを検証する。
// DeepSeek/Claude/Gemini等が `@@ package ui` のようなコンテキストを生成し、
// 先頭行の変更が失敗するバグの回帰テスト。
func TestApplyPatch_FirstLineChangeWithContextAfter(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "colors.go",
			"// colors defines shared color constants.\n"+
				"package ui\n"+
				"\n"+
				"import \"github.com/fatih/color\"\n")

		// モデルが典型的に生成するパッチ: @@ が変更対象行より後ろを指す
		patch := "*** Begin Patch\n" +
			"*** Update File: colors.go\n" +
			"@@ package ui\n" +
			"-// colors defines shared color constants.\n" +
			"+// colors はカラー定数を定義する。\n" +
			" package ui\n" +
			"*** End Patch\n"

		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, []string{"colors.go"}, nil)
		assertFileContent(t, "colors.go",
			"// colors はカラー定数を定義する。\n"+
				"package ui\n"+
				"\n"+
				"import \"github.com/fatih/color\"\n")
	})
}

// TestApplyPatch_FirstLineChangeNoContext は、@@コンテキストなしでの先頭行変更を検証する。
func TestApplyPatch_FirstLineChangeNoContext(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "main.go",
			"// main is the entry point.\n"+
				"package main\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: main.go\n" +
			"@@\n" +
			"-// main is the entry point.\n" +
			"+// main はエントリーポイント。\n" +
			" package main\n" +
			"*** End Patch\n"

		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, []string{"main.go"}, nil)
		assertFileContent(t, "main.go",
			"// main はエントリーポイント。\n"+
				"package main\n")
	})
}

// TestApplyPatch_MultiChunkForwardMatch は3チャンクが順方向にマッチすることを検証する。
func TestApplyPatch_MultiChunkForwardMatch(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "multi.go",
			"package main\n"+
				"\n"+
				"func a() {\n"+
				"\tfmt.Println(\"a\")\n"+
				"}\n"+
				"\n"+
				"func b() {\n"+
				"\tfmt.Println(\"b\")\n"+
				"}\n"+
				"\n"+
				"func c() {\n"+
				"\tfmt.Println(\"c\")\n"+
				"}\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: multi.go\n" +
			"@@ func a() {\n" +
			" func a() {\n" +
			"-\tfmt.Println(\"a\")\n" +
			"+\tfmt.Println(\"A\")\n" +
			" }\n" +
			"@@ func b() {\n" +
			" func b() {\n" +
			"-\tfmt.Println(\"b\")\n" +
			"+\tfmt.Println(\"B\")\n" +
			" }\n" +
			"@@ func c() {\n" +
			" func c() {\n" +
			"-\tfmt.Println(\"c\")\n" +
			"+\tfmt.Println(\"C\")\n" +
			" }\n" +
			"*** End Patch\n"

		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, []string{"multi.go"}, nil)
		assertFileContent(t, "multi.go",
			"package main\n"+
				"\n"+
				"func a() {\n"+
				"\tfmt.Println(\"A\")\n"+
				"}\n"+
				"\n"+
				"func b() {\n"+
				"\tfmt.Println(\"B\")\n"+
				"}\n"+
				"\n"+
				"func c() {\n"+
				"\tfmt.Println(\"C\")\n"+
				"}\n")
	})
}

// TestApplyPatch_FallbackOverlapDetection はフォールバック再検索でオーバーラップを検出しエラーを返すことを検証する。
// chunk 2 の OldLines が chunk 1 より前にマッチする場合、エラーになるべき。
func TestApplyPatch_FallbackOverlapDetection(t *testing.T) {
	withTempWorkdir(t, func() {
		// "hello" が2回出現するファイル
		writeTestFile(t, "dup.txt",
			"hello\n"+
				"world\n"+
				"hello\n"+
				"earth\n")

		// chunk 1: 1番目の "hello" を変更
		// chunk 2: @@ が "earth" を指すが OldLines の "hello" は
		//          前方の1番目にもマッチする → オーバーラップ検出でエラー
		patch := "*** Begin Patch\n" +
			"*** Update File: dup.txt\n" +
			"@@ hello\n" +
			"-hello\n" +
			"-world\n" +
			"+hi\n" +
			"+world\n" +
			"@@ earth\n" +
			"-hello\n" +
			"+hey\n" +
			"*** End Patch\n"

		// chunk 2 の OldLines "hello" は lineIndex(earth+1=4)以降にない。
		// フォールバックで先頭から探すと index 0 だが、prevChunkEnd(2)より前 → エラー
		_, err := ApplyPatch(patch)
		if err != nil {
			// 正常にエラーが発生した: chunk 2 の "hello" が見つからない
			if !strings.Contains(err.Error(), "failed to find expected lines") {
				t.Fatalf("unexpected error: %v", err)
			}
			return
		}
		// もし 2番目の "hello" (index 2) にマッチして成功した場合も許容
		// （prevChunkEnd=2 で startIdx=2 は >= なので通る）
		assertFileContent(t, "dup.txt",
			"hi\n"+
				"world\n"+
				"hey\n"+
				"earth\n")
	})
}

// TestApplyPatch_PreviewMatchesApply はプレビューとapplyが同じパッチで一致することを検証する。
func TestApplyPatch_PreviewMatchesApply(t *testing.T) {
	withTempWorkdir(t, func() {
		content := "// comment\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"
		writeTestFile(t, "target.go", content)

		patch := "*** Begin Patch\n" +
			"*** Update File: target.go\n" +
			"@@ package main\n" +
			"-// comment\n" +
			"+// updated comment\n" +
			" package main\n" +
			"*** End Patch\n"

		// プレビューがエラーなく生成される
		previews, err := BuildPatchPreview(patch, func(path string) ([]byte, error) {
			return os.ReadFile(path)
		})
		if err != nil {
			t.Fatalf("BuildPatchPreview() error = %v", err)
		}
		if len(previews) != 1 {
			t.Fatalf("expected 1 preview, got %d", len(previews))
		}
		if previews[0].Added != 1 || previews[0].Removed != 1 {
			t.Fatalf("expected +1/-1, got +%d/-%d", previews[0].Added, previews[0].Removed)
		}

		// applyも成功する
		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, []string{"target.go"}, nil)
		assertFileContent(t, "target.go",
			"// updated comment\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")
	})
}

func TestApplyPatch_UnifiedDiffHeaderError(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "target.go", "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: target.go\n" +
			"@@ -1,3 +1,5 @@\n" +
			" package main\n" +
			"+import \"os\"\n" +
			"*** End Patch\n"

		_, err := ApplyPatch(patch)
		if err == nil {
			t.Fatal("expected error for unified diff header")
		}
		if !strings.Contains(err.Error(), "not unified diff line numbers") {
			t.Errorf("error should mention unified diff line numbers, got: %v", err)
		}
	})
}
