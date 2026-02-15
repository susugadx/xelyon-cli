package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfig(t *testing.T) {
	// 元のディレクトリを保存
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	t.Run("xelyon.yaml exists in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		content := "context: \"Test Project context\"\nrules:\n  - \"rule 1\"\n"
		err := os.WriteFile(filepath.Join(tmpDir, "xelyon.yaml"), []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}

		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		result := loadProjectConfig()
		if result == "" {
			t.Error("loadProjectConfig() returned empty, want non-empty")
		}
		if !contains(result, "Test Project context") {
			t.Errorf("loadProjectConfig() = %q, want to contain 'Test Project context'", result)
		}
	})

	t.Run("XELYON.md fallback in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		content := "# Test Project\nThis is a test."
		err := os.WriteFile(filepath.Join(tmpDir, "XELYON.md"), []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}

		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		result := loadProjectConfig()
		if result == "" {
			t.Error("loadProjectConfig() returned empty, want non-empty")
		}
		if !contains(result, "Test Project") {
			t.Errorf("loadProjectConfig() = %q, want to contain 'Test Project'", result)
		}
	})

	t.Run("xelyon.yaml exists in parent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		parentDir := filepath.Join(tmpDir, "parent")
		childDir := filepath.Join(parentDir, "child")
		if err := os.MkdirAll(childDir, 0755); err != nil {
			t.Fatal(err)
		}
		content := "context: \"Parent Config\"\nrules:\n  - \"parent rule\"\n"
		err := os.WriteFile(filepath.Join(parentDir, "xelyon.yaml"), []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}

		if err := os.Chdir(childDir); err != nil {
			t.Fatal(err)
		}

		result := loadProjectConfig()
		if result == "" {
			t.Error("loadProjectConfig() returned empty, want non-empty")
		}
		if !contains(result, "Parent Config") {
			t.Errorf("loadProjectConfig() = %q, want to contain 'Parent Config'", result)
		}
	})

	// Note: "no config found" テストは、親ディレクトリにxelyon.yamlやXELYON.mdが
	// 存在する可能性があるため、実行環境に依存する。スキップする。
}

func TestGetModel(t *testing.T) {
	tests := []struct {
		name      string
		modelFlag string
		want      string
	}{
		{
			name:      "model flag set",
			modelFlag: "gpt-4o",
			want:      "gpt-4o",
		},
		{
			name:      "model flag empty uses config default",
			modelFlag: "",
			want:      "", // 設定ファイルに依存するためここではチェックしない
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// グローバル変数を設定
			oldFlag := modelFlag
			modelFlag = tt.modelFlag
			defer func() { modelFlag = oldFlag }()

			result := getModel(nil)

			if tt.modelFlag != "" && result != tt.want {
				t.Errorf("getModel() = %q, want %q", result, tt.want)
			}
			// modelFlag が空の場合は設定ファイルに依存するため、空でないことだけ確認
			if tt.modelFlag == "" && result == "" {
				t.Error("getModel() returned empty when modelFlag is empty")
			}
		})
	}
}

// contains は文字列に部分文字列が含まれるかチェック
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchSubstring(s, substr)))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
