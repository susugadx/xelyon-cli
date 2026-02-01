package plan

import (
	"os"
	"testing"
	"time"
)

func TestPlanStorage_LoadByID(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	storage, err := NewPlanStorage()
	if err != nil {
		t.Fatalf("NewPlanStorage() error = %v", err)
	}

	// テスト用の計画を作成
	p := &Plan{
		ID:        "loadbyid-test-uuid",
		Title:     "LoadByID Test",
		Status:    PlanStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	expectedFilename, _ := storage.Save(p)

	tests := []struct {
		name         string
		id           string
		wantErr      bool
		wantFilename string
	}{
		{
			name:         "存在するID",
			id:           "loadbyid-test-uuid",
			wantErr:      false,
			wantFilename: expectedFilename,
		},
		{
			name:    "存在しないID",
			id:      "not-exist",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, filename, err := storage.LoadByID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadByID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if plan == nil {
					t.Error("LoadByID() returned nil plan")
				}
				if filename != tt.wantFilename {
					t.Errorf("LoadByID() filename = %v, want %v", filename, tt.wantFilename)
				}
			}
		})
	}
}

func TestPlanStorage_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	storage, err := NewPlanStorage()
	if err != nil {
		t.Fatalf("NewPlanStorage() error = %v", err)
	}

	// テスト用の計画を作成
	p := &Plan{
		ID:          "save-load-test",
		Title:       "Save Load Test",
		Summary:     "Test summary",
		Status:      PlanStatusPending,
		UserRequest: "Test request",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Steps: []PlanStep{
			{ID: 1, Description: "Step 1", Status: "pending", Tools: []string{"read_file"}},
			{ID: 2, Description: "Step 2", Status: "pending", DependsOn: []int{1}},
		},
	}

	// 保存
	filename, err := storage.Save(p)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// 読み込み
	loaded, err := storage.Load(filename)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// 検証
	if loaded.ID != p.ID {
		t.Errorf("ID = %v, want %v", loaded.ID, p.ID)
	}
	if loaded.Title != p.Title {
		t.Errorf("Title = %v, want %v", loaded.Title, p.Title)
	}
	if loaded.Summary != p.Summary {
		t.Errorf("Summary = %v, want %v", loaded.Summary, p.Summary)
	}
	if len(loaded.Steps) != len(p.Steps) {
		t.Errorf("Steps count = %v, want %v", len(loaded.Steps), len(p.Steps))
	}
}

func TestPlanStorage_List(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	storage, err := NewPlanStorage()
	if err != nil {
		t.Fatalf("NewPlanStorage() error = %v", err)
	}

	// 空の状態でリスト取得
	plans, err := storage.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("List() count = %v, want 0", len(plans))
	}

	// 計画を作成
	for i := 0; i < 3; i++ {
		p := &Plan{
			ID:        "list-test-" + string(rune('a'+i)),
			Title:     "Plan " + string(rune('A'+i)),
			Status:    PlanStatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		storage.Save(p)
	}

	// リスト取得
	plans, err = storage.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(plans) != 3 {
		t.Errorf("List() count = %v, want 3", len(plans))
	}
}

func TestPlanStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	storage, err := NewPlanStorage()
	if err != nil {
		t.Fatalf("NewPlanStorage() error = %v", err)
	}

	// 計画を作成
	p := &Plan{
		ID:        "delete-test",
		Title:     "To Delete",
		Status:    PlanStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	filename, _ := storage.Save(p)

	// 削除
	err = storage.Delete(filename)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 読み込み失敗を確認
	_, err = storage.Load(filename)
	if err == nil {
		t.Error("Load() should fail after Delete()")
	}
}
