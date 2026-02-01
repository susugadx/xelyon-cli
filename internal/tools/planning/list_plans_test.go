package planning

import (
	"os"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestListPlansTool_Run(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	storage, _ := plan.NewPlanStorage()
	tool := NewListPlansTool(storage)

	// テスト用の計画を複数作成
	plans := []*plan.Plan{
		{ID: "uuid-1", Title: "Plan 1", Status: plan.PlanStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "uuid-2", Title: "Plan 2", Status: plan.PlanStatusCompleted, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "uuid-3", Title: "Plan 3", Status: plan.PlanStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, p := range plans {
		storage.Save(p)
	}

	tests := []struct {
		name      string
		args      map[string]string
		wantEmpty bool
	}{
		{
			name:      "全件取得",
			args:      map[string]string{},
			wantEmpty: false,
		},
		{
			name:      "ステータスフィルタ（pending）",
			args:      map[string]string{"status": "pending"},
			wantEmpty: false,
		},
		{
			name:      "ステータスフィルタ（存在しない）",
			args:      map[string]string{"status": "draft"},
			wantEmpty: true,
		},
		{
			name:      "件数制限",
			args:      map[string]string{"limit": "1"},
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := tool.Run(tt.args)
			if err != nil {
				t.Errorf("Run() error = %v", err)
			}
			isEmpty := result == "" || result == "保存された計画はありません" || result == "No saved plans"
			if isEmpty != tt.wantEmpty {
				t.Errorf("Run() isEmpty = %v, wantEmpty %v, result = %s", isEmpty, tt.wantEmpty, result)
			}
		})
	}
}
