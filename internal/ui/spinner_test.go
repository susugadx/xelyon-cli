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

func TestSpinner_StartStop(t *testing.T) {
	s := NewSpinner()
	var buf bytes.Buffer
	s.writer = &buf

	// Start
	s.Start("Testing")

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if !active {
		t.Error("Start() should set active to true")
	}

	// 少し待機してアニメーションを確認
	time.Sleep(100 * time.Millisecond)

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

	// クリア文字が含まれていることを確認
	if !strings.Contains(output, "\r") {
		t.Error("Spinner output should contain carriage return")
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

func TestSpinner_Animation(t *testing.T) {
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
	messages := []string{"Loading", "Processing", "Finishing"}

	for _, msg := range messages {
		s := NewSpinner()
		var buf bytes.Buffer
		s.writer = &buf

		s.Start(msg)
		time.Sleep(100 * time.Millisecond)

		// Stop()前にバッファの内容を取得（Stop()がクリアシーケンスを書くため）
		output := buf.String()
		s.Stop()

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
	s := NewSpinner()
	var buf bytes.Buffer
	s.writer = &buf

	s.Start("Test message")
	time.Sleep(100 * time.Millisecond)
	s.Stop()

	output := buf.String()

	// ANSI エスケープシーケンスを確認
	if !strings.Contains(output, "\033[K") {
		t.Error("Stop() should output ANSI clear sequence")
	}
}
