package dev

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// executeBashWithStreamingAndContext は Context 対応でストリーミング出力する。
func executeBashWithStreamingAndContext(ctx context.Context, out common.Output, cmd *exec.Cmd) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	var outputBuf strings.Builder
	var mu sync.Mutex
	var wg sync.WaitGroup
	doneCh := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		streamOutputWithContext(ctx, out, stdout, &outputBuf, &mu, false)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		streamOutputWithContext(ctx, out, stderr, &outputBuf, &mu, true)
	}()

	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-ctx.Done():
		killProcessGroup(cmd)
		return outputBuf.String(), ctx.Err()
	case <-doneCh:
		err := cmd.Wait()
		return outputBuf.String(), err
	}
}

// streamOutputWithContext は Context 対応でストリーミング出力する。
func streamOutputWithContext(ctx context.Context, out common.Output, pipe io.Reader, buf *strings.Builder, mu *sync.Mutex, isStderr bool) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()

		if isStderr {
			out.Red.Println(line)
		} else {
			out.Println(line)
		}

		mu.Lock()
		buf.WriteString(line + "\n")
		mu.Unlock()
	}
}

// executeBashWithStreaming はコマンド出力をリアルタイムでストリーミングする。
func executeBashWithStreaming(out common.Output, cmd *exec.Cmd) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	var outputBuf strings.Builder
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		streamOutput(out, stdout, &outputBuf, &mu, false)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		streamOutput(out, stderr, &outputBuf, &mu, true)
	}()

	wg.Wait()
	err = cmd.Wait()

	return outputBuf.String(), err
}

// streamOutput はパイプからの出力をリアルタイム表示し、バッファに保存する。
func streamOutput(out common.Output, pipe io.Reader, buf *strings.Builder, mu *sync.Mutex, isStderr bool) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()

		if isStderr {
			out.Red.Println(line)
		} else {
			out.Println(line)
		}

		mu.Lock()
		buf.WriteString(line + "\n")
		mu.Unlock()
	}
}
