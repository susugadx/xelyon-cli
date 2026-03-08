package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestProgress_Determinate(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgressWithWriter(100, "Processing", &buf)

	p.Start()
	p.Update(50)
	time.Sleep(150 * time.Millisecond) // 描画を待つ
	p.Stop()

	output := buf.String()
	if !strings.Contains(output, "50%") {
		t.Errorf("Expected output to contain '50%%', got: %s", output)
	}
	if !strings.Contains(output, "Processing") {
		t.Errorf("Expected output to contain 'Processing', got: %s", output)
	}
}

func TestProgress_Indeterminate(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgressWithWriter(0, "Loading", &buf)

	p.Start()
	time.Sleep(150 * time.Millisecond) // 描画を待つ
	p.Stop()

	output := buf.String()
	if !strings.Contains(output, "Loading") {
		t.Errorf("Expected output to contain 'Loading', got: %s", output)
	}
	// スピナーフレームが含まれていることを確認
	hasSpinner := false
	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	for _, frame := range spinnerFrames {
		if strings.Contains(output, frame) {
			hasSpinner = true
			break
		}
	}
	if !hasSpinner {
		t.Errorf("Expected output to contain spinner frame, got: %s", output)
	}
}

func TestProgress_Increment(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgressWithWriter(10, "Counting", &buf)

	p.Start()
	for i := 0; i < 5; i++ {
		p.Increment()
	}
	time.Sleep(150 * time.Millisecond)
	p.Stop()

	if p.GetCurrent() != 5 {
		t.Errorf("Expected current to be 5, got: %d", p.GetCurrent())
	}
}

func TestProgress_SetMessage(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgressWithWriter(100, "Initial", &buf)

	p.Start()
	time.Sleep(150 * time.Millisecond)
	p.SetMessage("Updated")
	time.Sleep(150 * time.Millisecond)
	p.Stop()

	output := buf.String()
	if !strings.Contains(output, "Updated") {
		t.Errorf("Expected output to contain 'Updated', got: %s", output)
	}
}

func TestProgress_Complete(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgressWithWriter(100, "Task", &buf)

	p.Start()
	p.Update(50)
	p.Complete()

	if p.GetCurrent() != 100 {
		t.Errorf("Expected current to be 100 after Complete, got: %d", p.GetCurrent())
	}
}

func TestProgress_DoubleStart(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgressWithWriter(100, "Test", &buf)

	p.Start()
	p.Start() // 二重呼び出し
	time.Sleep(150 * time.Millisecond)
	p.Stop()

	// パニックしないことを確認
}

func TestProgress_StopWithoutStart(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgressWithWriter(100, "Test", &buf)

	p.Stop() // Start前にStop

	// パニックしないことを確認
}

func TestNewProgressWithRuntime_UsesInjectedWriter(t *testing.T) {
	runtime := NewRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	out := runtime.Output().(*bytes.Buffer)
	p := NewProgressWithRuntime(10, "Runtime Progress", runtime)

	p.Start()
	p.Update(5)
	time.Sleep(150 * time.Millisecond)
	p.Stop()

	output := out.String()
	if !strings.Contains(output, "Runtime Progress") {
		t.Fatalf("expected injected output to contain runtime progress message, got %q", output)
	}
	if !strings.Contains(output, "50%") {
		t.Fatalf("expected injected output to contain progress percentage, got %q", output)
	}
}
