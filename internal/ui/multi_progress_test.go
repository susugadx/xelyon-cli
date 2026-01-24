package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestMultiProgress_AddTask(t *testing.T) {
	mp := NewMultiProgress()
	mp.AddTask(1, "Test task 1")
	mp.AddTask(2, "Test task 2")

	if len(mp.tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(mp.tasks))
	}

	if mp.tasks[0].ID != 1 || mp.tasks[0].Message != "Test task 1" {
		t.Errorf("First task mismatch: %+v", mp.tasks[0])
	}

	if mp.tasks[1].ID != 2 || mp.tasks[1].Message != "Test task 2" {
		t.Errorf("Second task mismatch: %+v", mp.tasks[1])
	}
}

func TestMultiProgress_StatusTransitions(t *testing.T) {
	mp := NewMultiProgress()
	mp.AddTask(1, "Test task")

	// 初期状態
	if mp.tasks[0].Status != TaskPending {
		t.Errorf("Expected TaskPending, got %v", mp.tasks[0].Status)
	}

	// Running
	mp.Start(1)
	if mp.tasks[0].Status != TaskRunning {
		t.Errorf("Expected TaskRunning, got %v", mp.tasks[0].Status)
	}
	if mp.tasks[0].StartTime.IsZero() {
		t.Error("StartTime should be set")
	}

	// Done
	mp.Done(1)
	if mp.tasks[0].Status != TaskDone {
		t.Errorf("Expected TaskDone, got %v", mp.tasks[0].Status)
	}
	if mp.tasks[0].EndTime.IsZero() {
		t.Error("EndTime should be set")
	}
}

func TestMultiProgress_Fail(t *testing.T) {
	mp := NewMultiProgress()
	mp.AddTask(1, "Test task")
	mp.Start(1)
	mp.Fail(1, "something went wrong")

	if mp.tasks[0].Status != TaskError {
		t.Errorf("Expected TaskError, got %v", mp.tasks[0].Status)
	}
	if mp.tasks[0].Error != "something went wrong" {
		t.Errorf("Expected error message, got %s", mp.tasks[0].Error)
	}
}

func TestMultiProgress_Render(t *testing.T) {
	var buf bytes.Buffer
	mp := NewMultiProgressWithWriter(&buf)

	mp.AddTask(1, "Read file")
	mp.AddTask(2, "Search code")
	mp.AddTask(3, "Write output")

	mp.Start(1)
	mp.Done(1)
	mp.Start(2)
	mp.Render()

	output := buf.String()

	// Check for task status indicators
	if !strings.Contains(output, "[✓]") {
		t.Error("Expected [✓] for done task")
	}
	if !strings.Contains(output, "[⠋]") {
		t.Error("Expected [⠋] for running task")
	}
	if !strings.Contains(output, "[○]") {
		t.Error("Expected [○] for pending task")
	}

	// Check for task descriptions
	if !strings.Contains(output, "Read file") {
		t.Error("Expected task message 'Read file'")
	}
	if !strings.Contains(output, "Search code") {
		t.Error("Expected task message 'Search code'")
	}
}

func TestMultiProgress_IsAllDone(t *testing.T) {
	mp := NewMultiProgress()
	mp.AddTask(1, "Task 1")
	mp.AddTask(2, "Task 2")

	// Not all done
	mp.Start(1)
	if mp.IsAllDone() {
		t.Error("Expected IsAllDone to be false when tasks are running")
	}

	// Still not all done
	mp.Done(1)
	if mp.IsAllDone() {
		t.Error("Expected IsAllDone to be false when some tasks are pending")
	}

	// Now all done
	mp.Start(2)
	mp.Done(2)
	if !mp.IsAllDone() {
		t.Error("Expected IsAllDone to be true when all tasks are done")
	}
}

func TestMultiProgress_GetTask(t *testing.T) {
	mp := NewMultiProgress()
	mp.AddTask(1, "Task 1")
	mp.AddTask(2, "Task 2")

	task := mp.GetTask(1)
	if task == nil {
		t.Fatal("Expected task, got nil")
	}
	if task.ID != 1 {
		t.Errorf("Expected ID 1, got %d", task.ID)
	}

	task = mp.GetTask(999)
	if task != nil {
		t.Error("Expected nil for non-existent task")
	}
}

func TestMultiProgress_Clear(t *testing.T) {
	var buf bytes.Buffer
	mp := NewMultiProgressWithWriter(&buf)

	mp.AddTask(1, "Task 1")
	mp.AddTask(2, "Task 2")
	mp.Clear()

	if len(mp.tasks) != 0 {
		t.Errorf("Expected 0 tasks after Clear, got %d", len(mp.tasks))
	}
}

func TestMultiProgress_StartStopRendering(t *testing.T) {
	var buf bytes.Buffer
	mp := NewMultiProgressWithWriter(&buf)

	mp.AddTask(1, "Task 1")
	mp.AddTask(2, "Task 2")

	mp.StartRendering()
	mp.Start(1)

	// Let it render a few times
	time.Sleep(450 * time.Millisecond)

	mp.StopRendering()

	// Output should contain task info
	output := buf.String()
	if !strings.Contains(output, "Task 1") {
		t.Error("Expected output to contain 'Task 1'")
	}
}

func TestMultiProgress_DoubleStartRendering(t *testing.T) {
	mp := NewMultiProgress()

	mp.StartRendering()
	mp.StartRendering() // Should not panic

	mp.StopRendering()
}

func TestMultiProgress_StopWithoutStart(t *testing.T) {
	mp := NewMultiProgress()

	mp.StopRendering() // Should not panic
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is a very long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}
