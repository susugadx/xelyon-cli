package api

import (
	"bufio"
	"context"
	"io"
	"time"
)

const defaultStreamScannerBufferSize = 10 * 1024 * 1024

// StreamLoopEventType は StreamLoopController が返すイベント種別。
type StreamLoopEventType int

const (
	// StreamLoopEventLine は 1 行読み取り完了イベント。
	StreamLoopEventLine StreamLoopEventType = iota
	// StreamLoopEventScannerDone は scanner 終了イベント。
	StreamLoopEventScannerDone
	// StreamLoopEventContextDone は context キャンセルイベント。
	StreamLoopEventContextDone
	// StreamLoopEventIdleTimeout は idle timeout イベント。
	StreamLoopEventIdleTimeout
	// StreamLoopEventExternal は外部タイマーなどの追加イベント。
	StreamLoopEventExternal
)

// StreamLoopEvent は StreamLoopController の単一イベント。
type StreamLoopEvent struct {
	Type        StreamLoopEventType
	Line        string
	Err         error
	IdleTimeout time.Duration
}

// StreamLoopOptions は StreamLoopController の設定。
type StreamLoopOptions struct {
	IdleTimeout         time.Duration
	AutoResetIdleOnLine bool
	ScannerBufferSize   int
}

type streamScanResult struct {
	line string
	err  error
	done bool
}

// StreamLoopController はストリーミング読取の loop/timer/cancel を管理する。
type StreamLoopController struct {
	lineCh              <-chan streamScanResult
	idleTimer           *time.Timer
	idleTimeout         time.Duration
	autoResetIdleOnLine bool
	finished            bool
}

// NewStreamLoopController はストリーミング読取制御を初期化する。
func NewStreamLoopController(body io.Reader, opts StreamLoopOptions) *StreamLoopController {
	bufferSize := opts.ScannerBufferSize
	if bufferSize <= 0 {
		bufferSize = defaultStreamScannerBufferSize
	}

	controller := &StreamLoopController{
		lineCh:              startStreamScanner(body, bufferSize),
		idleTimeout:         opts.IdleTimeout,
		autoResetIdleOnLine: opts.AutoResetIdleOnLine,
	}
	if opts.IdleTimeout > 0 {
		controller.idleTimer = time.NewTimer(opts.IdleTimeout)
	}

	return controller
}

// Next は次のイベントを 1 つ返す。
// externalCh には thinking timeout など追加で待ちたいチャネルを渡す。
func (c *StreamLoopController) Next(ctx context.Context, externalCh <-chan time.Time) StreamLoopEvent {
	if c.finished {
		return StreamLoopEvent{Type: StreamLoopEventScannerDone}
	}

	var idleTimerCh <-chan time.Time
	if c.idleTimer != nil {
		idleTimerCh = c.idleTimer.C
	}

	select {
	case <-ctx.Done():
		c.finish()
		return StreamLoopEvent{
			Type: StreamLoopEventContextDone,
			Err:  ctx.Err(),
		}

	case <-idleTimerCh:
		timeout := c.idleTimeout
		c.finish()
		return StreamLoopEvent{
			Type:        StreamLoopEventIdleTimeout,
			IdleTimeout: timeout,
		}

	case <-externalCh:
		return StreamLoopEvent{Type: StreamLoopEventExternal}

	case result, ok := <-c.lineCh:
		if !ok {
			c.finish()
			return StreamLoopEvent{Type: StreamLoopEventScannerDone}
		}

		if result.done {
			c.finish()
			return StreamLoopEvent{
				Type: StreamLoopEventScannerDone,
				Err:  result.err,
			}
		}

		if c.autoResetIdleOnLine {
			c.ResetIdleTimer()
		}

		return StreamLoopEvent{
			Type: StreamLoopEventLine,
			Line: result.line,
		}
	}
}

// ResetIdleTimer は idle timer を初期状態に戻す。
func (c *StreamLoopController) ResetIdleTimer() {
	if c.idleTimer == nil {
		return
	}
	if !c.idleTimer.Stop() {
		select {
		case <-c.idleTimer.C:
		default:
		}
	}
	c.idleTimer.Reset(c.idleTimeout)
}

// Stop は controller が保持している内部タイマーを停止する。
func (c *StreamLoopController) Stop() {
	c.finish()
}

func (c *StreamLoopController) finish() {
	if c.finished {
		return
	}
	c.finished = true
	c.stopIdleTimer()
}

func (c *StreamLoopController) stopIdleTimer() {
	if c.idleTimer == nil {
		return
	}
	if !c.idleTimer.Stop() {
		select {
		case <-c.idleTimer.C:
		default:
		}
	}
}

func startStreamScanner(body io.Reader, bufferSize int) <-chan streamScanResult {
	lineCh := make(chan streamScanResult)
	go func() {
		defer close(lineCh)
		scanner := bufio.NewScanner(body)
		buf := make([]byte, 0, bufferSize)
		scanner.Buffer(buf, bufferSize)
		for scanner.Scan() {
			lineCh <- streamScanResult{line: scanner.Text()}
		}
		if err := scanner.Err(); err != nil {
			lineCh <- streamScanResult{err: err, done: true}
			return
		}
		lineCh <- streamScanResult{done: true}
	}()
	return lineCh
}
