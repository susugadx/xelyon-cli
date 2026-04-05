package ui

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewSpinner(t *testing.T) {
	s := NewSpinner()

	if s == nil {
		t.Fatal("NewSpinner() returned nil")
	}

	if s.writer == nil {
		t.Error("NewSpinner() writer is nil")
	}

	if len(s.frames) != 10 {
		t.Errorf("NewSpinner() frames length = %d, want 10", len(s.frames))
	}

	if s.active {
		t.Error("NewSpinner() active should be false initially")
	}

	if s.stopChan != nil {
		t.Error("NewSpinner() stopChan should be nil initially")
	}
}

func TestNewSpinnerWithRuntime_UsesInjectedWriter(t *testing.T) {
	runtime := NewRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	out := runtime.Output()

	s := NewSpinnerWithRuntime(runtime)

	if s.writer != out {
		t.Fatalf("spinner writer = %#v, want runtime output %#v", s.writer, out)
	}
}

func TestSpinner_StartStop(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinnerWithWriter(&buf)

	// Start
	s.Start("Testing")

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if !active {
		t.Error("Start() should set active to true")
	}

	// Stop
	s.Stop()

	s.mu.Lock()
	active = s.active
	stopChan := s.stopChan
	s.mu.Unlock()

	if active {
		t.Error("Stop() should set active to false")
	}

	if stopChan != nil {
		t.Error("Stop() should set stopChan to nil")
	}

	// 出力が生成されたことを確認
	output := buf.String()
	if len(output) == 0 {
		t.Error("Spinner should produce output")
	}

	if !strings.Contains(output, "Testing") {
		t.Fatalf("Spinner output should contain start message, got %q", output)
	}

	if strings.Contains(output, "\r") {
		t.Errorf("non-terminal spinner should not animate with carriage returns, got %q", output)
	}
}

func TestSpinner_StartTwice(t *testing.T) {
	s := NewSpinner()
	var buf bytes.Buffer
	s.writer = &buf

	s.Start("First")

	// 2回目のStartは無視されるべき
	s.Start("Second")

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if !active {
		t.Error("Spinner should still be active")
	}

	s.Stop()
}

func TestSpinner_StopTwice(t *testing.T) {
	s := NewSpinner()
	var buf bytes.Buffer
	s.writer = &buf

	s.Start("Testing")
	time.Sleep(50 * time.Millisecond)
	s.Stop()

	// 2回目のStopはpanicしないはず
	s.Stop()

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if active {
		t.Error("Spinner should not be active after Stop")
	}
}

func TestSpinner_StopWithoutStart(t *testing.T) {
	s := NewSpinner()
	var buf bytes.Buffer
	s.writer = &buf

	// Startせずにstop（panicしないことを確認）
	s.Stop()

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if active {
		t.Error("Spinner should not be active")
	}
}

func TestSpinner_SetStatus_StaticWriterLogsDistinctUpdates(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinnerWithWriter(&buf)

	s.Start("Waiting for Gemini...")
	s.SetStatus("retrying in 1s")
	s.SetStatus("retrying in 1s")
	s.SetStatus("retrying in 2s")
	s.Stop()

	output := buf.String()
	if strings.Count(output, "Waiting for Gemini...") != 3 {
		t.Fatalf("expected one start line and two distinct status lines, got %q", output)
	}
	if strings.Contains(output, "\033[38;5;") {
		t.Fatalf("non-terminal spinner should not emit ANSI color escapes, got %q", output)
	}
}

func TestSpinner_Animation(t *testing.T) {
	// Note: This test has a race condition when using bytes.Buffer
	// TODO: Use a thread-safe writer or sync the buffer access
	t.Skip("Skipping due to race condition with bytes.Buffer - needs thread-safe writer")

	s := NewSpinner()
	var buf bytes.Buffer
	s.writer = &buf

	s.Start("Loading")

	// アニメーションフレームが変化するのを待つ
	time.Sleep(200 * time.Millisecond)

	s.Stop()

	output := buf.String()

	// 複数のフレームが出力されているはず
	frameCount := strings.Count(output, "\r")
	if frameCount < 2 {
		t.Errorf("Expected multiple animation frames, got %d", frameCount)
	}

	// メッセージが含まれていることを確認
	if !strings.Contains(output, "Loading") {
		t.Error("Spinner output should contain the message")
	}
}

func TestSpinner_ConcurrentStartStop(t *testing.T) {
	s := NewSpinner()
	var buf bytes.Buffer
	s.writer = &buf

	var wg sync.WaitGroup
	iterations := 10

	// 並行してStart/Stopを繰り返す
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Start("Concurrent")
			time.Sleep(10 * time.Millisecond)
			s.Stop()
		}()
	}

	wg.Wait()

	// panicせずに完了することを確認
	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if active {
		t.Error("Spinner should not be active after all goroutines complete")
	}
}

func TestSpinner_MultipleMessages(t *testing.T) {
	// Note: This test has a race condition when using bytes.Buffer
	// TODO: Use a thread-safe writer or sync the buffer access
	t.Skip("Skipping due to race condition with bytes.Buffer - needs thread-safe writer")

	messages := []string{"Loading", "Processing", "Finishing"}

	for _, msg := range messages {
		s := NewSpinner()
		var buf bytes.Buffer
		s.writer = &buf

		s.Start(msg)
		time.Sleep(100 * time.Millisecond)
		s.Stop()

		// Stop完了まで少し待機（race condition回避）
		time.Sleep(10 * time.Millisecond)

		output := buf.String()

		if !strings.Contains(output, msg) {
			t.Errorf("Expected output to contain %q, got %q", msg, output)
		}
	}
}

func TestSpinner_FrameRotation(t *testing.T) {
	s := NewSpinner()
	var buf bytes.Buffer
	s.writer = &buf

	// フレーム数を確認
	expectedFrames := 10
	if len(s.frames) != expectedFrames {
		t.Errorf("Expected %d frames, got %d", expectedFrames, len(s.frames))
	}

	// 各フレームが空でないことを確認
	for i, frame := range s.frames {
		if frame == "" {
			t.Errorf("Frame %d is empty", i)
		}
	}
}

func TestSpinner_RapidStartStop(t *testing.T) {
	s := NewSpinner()
	var buf bytes.Buffer
	s.writer = &buf

	// 非常に短い間隔でStart/Stop
	for i := 0; i < 5; i++ {
		s.Start("Rapid")
		time.Sleep(5 * time.Millisecond)
		s.Stop()
		time.Sleep(5 * time.Millisecond)
	}

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if active {
		t.Error("Spinner should not be active after rapid start/stop cycles")
	}
}

func TestSpinner_WriterOutput(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinnerWithWriter(&buf)

	s.Start("Test message")
	s.Stop()

	output := buf.String()

	if strings.Contains(output, "\033[K") {
		t.Errorf("non-terminal spinner should not emit ANSI clear sequence, got %q", output)
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "less than 1 second",
			duration: 500 * time.Millisecond,
			want:     "",
		},
		{
			name:     "exactly 0",
			duration: 0,
			want:     "",
		},
		{
			name:     "1 second",
			duration: 1 * time.Second,
			want:     "(1s)",
		},
		{
			name:     "5 seconds",
			duration: 5 * time.Second,
			want:     "(5s)",
		},
		{
			name:     "10 seconds",
			duration: 10 * time.Second,
			want:     "(10s)",
		},
		{
			name:     "60 seconds",
			duration: 60 * time.Second,
			want:     "(60s)",
		},
		{
			name:     "90 seconds",
			duration: 90 * time.Second,
			want:     "(90s)",
		},
		{
			name:     "1.5 seconds (rounds down)",
			duration: 1500 * time.Millisecond,
			want:     "(1s)",
		},
		{
			name:     "999 milliseconds",
			duration: 999 * time.Millisecond,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatElapsed(tt.duration)
			if got != tt.want {
				t.Errorf("formatElapsed(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestSpinnerMessageForTool(t *testing.T) {
	tests := []struct {
		toolName string
		want     string
	}{
		{"write_file", "Writing file..."},
		{"str_replace", "Editing file..."},
		{"unknown_tool", "Preparing..."},
		{"", "Preparing..."},
	}

	for _, tt := range tests {
		got := SpinnerMessageForTool(tt.toolName)
		if got != tt.want {
			t.Errorf("SpinnerMessageForTool(%q) = %q; want %q", tt.toolName, got, tt.want)
		}
	}
}
