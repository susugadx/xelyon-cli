package planning

import (
	"os"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestDeletePlanTool_Run(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldDir) }()

	storage, _ := plan.NewPlanStorage()
	tool := NewDeletePlanTool(storage)

	// テスト用の計画を作成
	p := &plan.Plan{
		ID:        "delete-test-uuid",
		Title:     "To Delete",
		Status:    plan.PlanStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	filename, _ := storage.Save(p)

	tests := []struct {
		name    string
		args    map[string]string
		wantErr bool
	}{
		{
			name:    "IDで削除",
			args:    map[string]string{"id": "delete-test-uuid"},
			wantErr: false,
		},
		{
			name:    "存在しないID",
			args:    map[string]string{"id": "not-exist"},
			wantErr: true,
		},
		{
			name:    "ID/filenameなし",
			args:    map[string]string{},
			wantErr: true,
		},
	}

	// 最初のテスト（削除成功）を実行
	t.Run(tests[0].name, func(t *testing.T) {
		_, _, err := tool.Run(tests[0].args)
		if (err != nil) != tests[0].wantErr {
			t.Errorf("Run() error = %v, wantErr %v", err, tests[0].wantErr)
		}

		// 削除されたか確認
		_, err = storage.Load(filename)
		if err == nil {
			t.Error("Plan should be deleted")
		}
	})

	// 残りのテストを実行
	for _, tt := range tests[1:] {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := tool.Run(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeletePlanTool_LastPlanIDFallback(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldDir) }()

	storage, _ := plan.NewPlanStorage()
	tool := NewDeletePlanTool(storage)

	// テスト用の計画を作成
	p := &plan.Plan{
		ID:        "del-fallback-uuid",
		Title:     "Delete Fallback Test",
		Status:    plan.PlanStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, _ = storage.Save(p)

	// Case 1: LastPlanID set → 削除成功
	storage.SetLastPlanID("del-fallback-uuid")
	_, _, err := tool.Run(map[string]string{})
	if err != nil {
		t.Errorf("Run() with LastPlanID fallback should succeed, got error: %v", err)
	}

	// Case 2: 削除後 LastPlanID クリアされている → error
	_, _, err = tool.Run(map[string]string{})
	if err == nil {
		t.Error("Run() after delete should return error (LastPlanID cleared)")
	}
}
