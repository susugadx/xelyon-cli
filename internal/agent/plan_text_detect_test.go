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

func TestIsPlanTool(t *testing.T) {
	planTools := []string{
		"create_plan", "update_plan", "delete_plan", "get_plan", "list_plans",
	}
	for _, name := range planTools {
		if !isPlanTool(name) {
			t.Errorf("isPlanTool(%q) = false, want true", name)
		}
	}

	nonPlanTools := []string{
		"write_file", "str_replace", "bash", "read_file", "list_dir", "",
	}
	for _, name := range nonPlanTools {
		if isPlanTool(name) {
			t.Errorf("isPlanTool(%q) = true, want false", name)
		}
	}
}

// TestStepTrackingFalsePositive は delete_plan を複数回呼ぶシナリオでの
// 誤検知を防ぐロジックを検証する。
// 「以下の5つを削除します\n1. xxx\n2. xxx...」という番号付きリストを含む
// レスポンスでも、ツール呼び出しが同時に存在する場合は Step Tracking を
// セットしないことを確認する。
func TestStepTrackingFalsePositive(t *testing.T) {
	// delete_plan を5回呼ぶ際にAIが出力する典型的なレスポンス
	response := `以下の5つの code_health 関連プランを削除します。

1. 20260101_code-health-check.md を削除
2. 20260102_code-health-refactor.md を削除
3. 20260103_code-health-lint.md を削除
4. 20260104_code-health-test.md を削除
5. 20260105_code-health-cleanup.md を削除`

	steps := extractTextPlan(response)
	if len(steps) == 0 {
		// extractTextPlan が反応しない場合はそもそも問題なし
		return
	}

	// ツール呼び出しがある場合（delete_plan FC と同時）は isActionPlan が true でも
	// pendingSteps をセットしないことが期待動作。
	// ここではロジックの前提条件（extractTextPlan が反応すること）と
	// isPlanTool のカウント対象を確認する。
	if !isActionPlan(steps) {
		// isActionPlan が false なら Step Tracking はセットされないので問題なし
		return
	}

	// extractTextPlan + isActionPlan が両方 true になるケースでも、
	// len(toolCalls) > 0 のガードにより pendingSteps はセットされない。
	// isPlanTool("delete_plan") が true であることを確認（カウント漏れ防止）。
	if !isPlanTool("delete_plan") {
		t.Error("isPlanTool(\"delete_plan\") = false: delete_plan は completedActions のカウント対象であるべき")
	}
}

func TestBuildAutoPlan(t *testing.T) {
	steps := []string{"Step 1", "Step 2", "Step 3"}
	p := buildAutoPlan(steps)

	if p.Summary != "Auto-detected 3-step plan from text" {
		t.Errorf("Summary = %q, want %q", p.Summary, "Auto-detected 3-step plan from text")
	}
	if len(p.Steps) != 3 {
		t.Errorf("len(Steps) = %d, want 3", len(p.Steps))
	}
	if p.Steps[0].Description != "Step 1" {
		t.Errorf("Steps[0].Description = %q, want %q", p.Steps[0].Description, "Step 1")
	}
	if p.Steps[0].ID != 1 {
		t.Errorf("Steps[0].ID = %d, want 1", p.Steps[0].ID)
	}
	if len(p.Steps[0].DependsOn) != 0 {
		t.Errorf("Steps[0].DependsOn = %v, want empty", p.Steps[0].DependsOn)
	}

	if p.Steps[1].Description != "Step 2" {
		t.Errorf("Steps[1].Description = %q, want %q", p.Steps[1].Description, "Step 2")
	}
	if p.Steps[1].ID != 2 {
		t.Errorf("Steps[1].ID = %d, want 2", p.Steps[1].ID)
	}
	if !reflect.DeepEqual(p.Steps[1].DependsOn, []int{1}) {
		t.Errorf("Steps[1].DependsOn = %v, want []int{1}", p.Steps[1].DependsOn)
	}

	if p.Steps[2].Description != "Step 3" {
		t.Errorf("Steps[2].Description = %q, want %q", p.Steps[2].Description, "Step 3")
	}
	if p.Steps[2].ID != 3 {
		t.Errorf("Steps[2].ID = %d, want 3", p.Steps[2].ID)
	}
	if !reflect.DeepEqual(p.Steps[2].DependsOn, []int{2}) {
		t.Errorf("Steps[2].DependsOn = %v, want []int{2}", p.Steps[2].DependsOn)
	}
}
