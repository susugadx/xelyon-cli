package planning

import (
	"os"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestUpdatePlanTool_Run(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldDir) }()

	storage, _ := plan.NewPlanStorage()
	tool := NewUpdatePlanTool(storage)

	// テスト用の計画を作成
	p := &plan.Plan{
		ID:        "test-uuid",
		Title:     "Original Title",
		Summary:   "Original summary",
		Status:    plan.PlanStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Steps: []plan.PlanStep{
			{ID: 1, Description: "Step 1", Status: "pending"},
		},
	}
	_, _ = storage.Save(p)

	tests := []struct {
		name    string
		args    map[string]string
		wantErr bool
	}{
		{
			name: "ステータス変更",
			args: map[string]string{
				"id":     "test-uuid",
				"action": "set_status",
				"status": "completed",
			},
			wantErr: false,
		},
		{
			name: "サマリー変更",
			args: map[string]string{
				"id":      "test-uuid",
				"action":  "set_summary",
				"summary": "New summary",
			},
			wantErr: false,
		},
		{
			name: "ステップ追加",
			args: map[string]string{
				"id":     "test-uuid",
				"action": "add_step",
				"step":   `{"id": 2, "description": "New step"}`,
			},
			wantErr: false,
		},
		{
			name: "不明なアクション",
			args: map[string]string{
				"id":     "test-uuid",
				"action": "unknown",
			},
			wantErr: true,
		},
		{
			name: "ID/filenameなし",
			args: map[string]string{
				"action": "set_status",
				"status": "completed",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := tool.Run(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdatePlanTool_TitleChangeDeletesOldFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldDir) }()

	storage, _ := plan.NewPlanStorage()
	tool := NewUpdatePlanTool(storage)

	// 計画を作成
	p := &plan.Plan{
		ID:        "title-test-uuid",
		Title:     "Old Title",
		Status:    plan.PlanStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	oldFilename, _ := storage.Save(p)

	// タイトル変更
	_, _, err := tool.Run(map[string]string{
		"id":     "title-test-uuid",
		"action": "set_title",
		"title":  "New Title",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// 古いファイルが削除されているか確認
	_, err = storage.Load(oldFilename)
	if err == nil {
		t.Error("Old file should be deleted after title change")
	}
}

func TestUpdatePlanTool_LastPlanIDFallback(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldDir) }()

	storage, _ := plan.NewPlanStorage()
	tool := NewUpdatePlanTool(storage)

	// テスト用の計画を作成
	p := &plan.Plan{
		ID:        "fallback-test-uuid",
		Title:     "Fallback Test",
		Summary:   "Test summary",
		Status:    plan.PlanStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Steps:     []plan.PlanStep{{ID: 1, Description: "Step 1", Status: "pending"}},
	}
	_, _ = storage.Save(p)

	// Case 1: LastPlanID set → success
	storage.SetLastPlanID("fallback-test-uuid")
	_, _, err := tool.Run(map[string]string{
		"action": "set_status",
		"status": "running",
	})
	if err != nil {
		t.Errorf("Run() with LastPlanID fallback should succeed, got error: %v", err)
	}

	// Case 2: LastPlanID empty → error
	storage.ClearLastPlanID()
	_, _, err = tool.Run(map[string]string{
		"action": "set_status",
		"status": "running",
	})
	if err == nil {
		t.Error("Run() without id/filename/LastPlanID should return error")
	}
}
