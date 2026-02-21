package agent

import (
	"reflect"
	"testing"
)

func TestExtractTextPlan(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     []string
	}{
		{
			name: "Standard numbered list",
			response: `Here is the plan:
1. Create test file
2. Fix lint errors
3. Update README`,
			want: []string{"Create test file", "Fix lint errors", "Update README"},
		},
		{
			name: "Less than 3 steps",
			response: `1. Fix typo
2. Commit`,
			want: nil,
		},
		{
			name: "Step N format",
			response: `Step 1: Create
Step 2: Fix
Step 3: Update`,
			want: []string{"Create", "Fix", "Update"},
		},
		{
			name: "Japanese format",
			response: `ステップ1: テスト作成
ステップ2: 修正
ステップ3: 更新`,
			want: []string{"テスト作成", "修正", "更新"},
		},
		{
			name: "Non-sequential numbers",
			response: `1. Do A
3. Do B
5. Do C`,
			want: nil,
		},
		{
			name:     "No steps",
			response: `Just a normal text response.`,
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTextPlan(tt.response)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractTextPlan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSequential(t *testing.T) {
	tests := []struct {
		name    string
		matches [][]string
		want    bool
	}{
		{
			name: "Sequential",
			matches: [][]string{
				{"1. A", "1", "A"},
				{"2. B", "2", "B"},
				{"3. C", "3", "C"},
			},
			want: true,
		},
		{
			name: "Non-sequential",
			matches: [][]string{
				{"1. A", "1", "A"},
				{"3. B", "3", "B"},
				{"5. C", "5", "C"},
			},
			want: false,
		},
		{
			name: "Start from 0",
			matches: [][]string{
				{"0. A", "0", "A"},
				{"1. B", "1", "B"},
				{"2. C", "2", "C"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSequential(tt.matches); got != tt.want {
				t.Errorf("isSequential() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsActionPlan(t *testing.T) {
	tests := []struct {
		name  string
		steps []string
		want  bool
	}{
		{
			name:  "All actions (Japanese)",
			steps: []string{"テストファイルを作成する", "lintエラーを修正する", "READMEを更新する"},
			want:  true,
		},
		{
			name:  "All analysis (Japanese)",
			steps: []string{"型の不一致が原因", "nilポインタの可能性", "importについて確認"},
			want:  false,
		},
		{
			name:  "All actions (English)",
			steps: []string{"Create test file", "Fix lint errors", "Update README"},
			want:  true,
		},
		{
			name:  "All analysis (English)",
			steps: []string{"The cause is X", "Could be Y", "Investigate Z"},
			want:  false,
		},
		{
			name:  "Mixed (mostly action)",
			steps: []string{"Create file", "Update config", "Consider refactoring"},
			want:  true, // 2/3 actions
		},
		{
			name:  "Mixed (mostly analysis)",
			steps: []string{"Investigate logs", "Analyze database", "Fix if needed"},
			want:  false, // 1/3 actions (Fix)
		},
		{
			name:  "New English verbs",
			steps: []string{"Optimize performance", "Debug issue", "Remove unused code"},
			want:  true,
		},
		{
			name:  "Merge and Check",
			steps: []string{"Merge branch", "Check status", "Deploy to prod"},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isActionPlan(tt.steps); got != tt.want {
				t.Errorf("isActionPlan() = %v, want %v", got, tt.want)
			}
		})
	}
}
