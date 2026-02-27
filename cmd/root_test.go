package cmd

import (
	"testing"
)

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
