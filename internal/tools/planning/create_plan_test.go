package planning

import (
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestCreatePlanTool_Run_Success(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldDir) }()

	storage, _ := plan.NewPlanStorage()
	tool := NewCreatePlanTool(storage)

	args := map[string]string{
		"title":          "Test Plan",
		"summary":        "Test summary",
		"user_request":   "Do something",
		"clarifications": `[{"question":"Q1","question_type":"free_text","answer":"A1"}]`,
		"steps":          `[{"id":1,"description":"Step 1","tools":["read_file"]}]`,
	}

	result, _, err := tool.Run(args)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result == "" {
		t.Error("Run() returned empty result")
	}

	lastPlan := tool.LastPlan()
	if lastPlan == nil {
		t.Fatal("LastPlan should not be nil")
	}
	if lastPlan.Title != "Test Plan" {
		t.Errorf("Title = %s, want %s", lastPlan.Title, "Test Plan")
	}
	if len(lastPlan.Steps) != 1 || lastPlan.Steps[0].Status != "pending" {
		t.Errorf("Steps not initialized correctly: %+v", lastPlan.Steps)
	}
	if len(lastPlan.Clarifications) != 1 || lastPlan.Clarifications[0].Answer != "A1" {
		t.Errorf("Clarifications not stored correctly: %+v", lastPlan.Clarifications)
	}

	loaded, _, err := storage.LoadByID(lastPlan.ID)
	if err != nil {
		t.Fatalf("LoadByID() error = %v", err)
	}
	if loaded.Title != lastPlan.Title {
		t.Errorf("Loaded title = %s, want %s", loaded.Title, lastPlan.Title)
	}
}

func TestCreatePlanTool_Run_InvalidClarifications(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldDir) }()

	storage, _ := plan.NewPlanStorage()
	tool := NewCreatePlanTool(storage)

	_, _, err := tool.Run(map[string]string{
		"title":          "Test Plan",
		"summary":        "Test summary",
		"clarifications": "{invalid-json",
		"steps":          `[{"id":1,"description":"Step 1","tools":["read_file"]}]`,
	})
	if err == nil {
		t.Error("Expected error for invalid clarifications JSON")
	}
}

func TestCreatePlanTool_Run_InvalidSteps(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldDir) }()

	storage, _ := plan.NewPlanStorage()
	tool := NewCreatePlanTool(storage)

	_, _, err := tool.Run(map[string]string{
		"title":   "Test Plan",
		"summary": "Test summary",
		"steps":   "{invalid-json",
	})
	if err == nil {
		t.Error("Expected error for invalid steps JSON")
	}
}
