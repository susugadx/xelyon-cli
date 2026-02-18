package planning

import (
	"os"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestGetPlanTool_Run(t *testing.T) {
	// テスト用の一時ディレクトリ
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldDir) }()

	storage, _ := plan.NewPlanStorage()
	tool := NewGetPlanTool(storage)

	// テスト用の計画を作成
	p := &plan.Plan{
		ID:        "test-uuid-123",
		Title:     "Test Plan",
		Summary:   "Test summary",
		Status:    plan.PlanStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Steps: []plan.PlanStep{
			{ID: 1, Description: "Step 1", Status: "pending"},
		},
	}
	filename, _ := storage.Save(p)

	tests := []struct {
		name    string
		args    map[string]string
		wantErr bool
	}{
		{
			name:    "IDで取得",
			args:    map[string]string{"id": "test-uuid-123"},
			wantErr: false,
		},
		{
			name:    "ファイル名で取得",
			args:    map[string]string{"filename": filename},
			wantErr: false,
		},
		{
			name:    "存在しないID",
			args:    map[string]string{"id": "not-exist"},
			wantErr: true,
		},
		{
			name:    "ID/filename両方なし",
			args:    map[string]string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := tool.Run(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == "" {
				t.Error("Run() returned empty result")
			}
		})
	}
}

func TestGetPlanTool_LastPlanIDFallback(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldDir) }()

	storage, _ := plan.NewPlanStorage()
	tool := NewGetPlanTool(storage)

	// テスト用の計画を作成
	p := &plan.Plan{
		ID:        "get-fallback-uuid",
		Title:     "Get Fallback Test",
		Summary:   "Test summary",
		Status:    plan.PlanStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Steps:     []plan.PlanStep{{ID: 1, Description: "Step 1", Status: "pending"}},
	}
	_, _ = storage.Save(p)

	// Case 1: LastPlanID set → success
	storage.SetLastPlanID("get-fallback-uuid")
	result, _, err := tool.Run(map[string]string{})
	if err != nil {
		t.Errorf("Run() with LastPlanID fallback should succeed, got error: %v", err)
	}
	if result == "" {
		t.Error("Run() with LastPlanID fallback returned empty result")
	}

	// Case 2: LastPlanID empty → error
	storage.ClearLastPlanID()
	_, _, err = tool.Run(map[string]string{})
	if err == nil {
		t.Error("Run() without id/filename/LastPlanID should return error")
	}
}
