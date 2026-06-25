package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/finalcheck"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

// runFinalCheckCommands は config.yaml の final_checks.commands に定義された
// シェルコマンドを順番に実行する。いずれかのコマンドが失敗した場合、
// needsContinue=true と AI 向けのフィードバックを返す。
// すべて成功した場合は needsContinue=false を返す。
func (a *Agent) runFinalCheckCommands(ctx context.Context, changedFiles []string) finalcheck.RunResult {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := a.cfg()
	commands := cfg.FinalChecks.Commands
	out := a.output()

	// No final checks → nothing to run
	if len(commands) == 0 {
		return finalcheck.RunResult{}
	}
	a.taskTestCommand = strings.Join(commands, " && ")

	// git diff --stat: final checks 失敗時のコンテキスト用
	yellow.Fprintln(out, "📊 Running final checks with git diff --stat...")
	var diffOutput string
	diffCmd := exec.CommandContext(ctx, "git", "diff", "--stat")
	if output, err := diffCmd.CombinedOutput(); err == nil {
		diffOutput = string(output)
		if strings.TrimSpace(diffOutput) != "" {
			_, _ = fmt.Fprintln(out, diffOutput)
		}
	}
	if err := ctx.Err(); err != nil {
		return finalcheck.BuildCancelledResult(err, nil)
	}

	// untracked files（write_file で新規作成されたファイル用）
	untrackedCmd := exec.CommandContext(ctx, "git", "ls-files", "--others", "--exclude-standard")
	if untrackedOut, err := untrackedCmd.CombinedOutput(); err == nil {
		untrackedStr := strings.TrimSpace(string(untrackedOut))
		if untrackedStr != "" {
			_, _ = fmt.Fprintf(out, "New files (untracked):\n%s\n", untrackedStr)
			diffOutput += "\nNew files (untracked):\n" + untrackedStr
		}
	}
	if err := ctx.Err(); err != nil {
		return finalcheck.BuildCancelledResult(err, nil)
	}

	timeout := time.Duration(cfg.FinalChecks.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 600 * time.Second
	}

	changedFilesEnv := strings.Join(changedFiles, " ")
	checks := make([]finalcheck.CommandResult, 0, len(commands))

	for _, cmd := range commands {
		if err := ctx.Err(); err != nil {
			return finalcheck.BuildCancelledResult(err, checks)
		}
		yellow.Fprintf(out, "🧪 Running final check command: %s\n", cmd)

		commandCtx, cancel := context.WithTimeout(ctx, timeout)
		proc := exec.CommandContext(commandCtx, "bash", "-c", cmd)
		setFinalCheckProcessGroup(proc)

		if cwd, err := os.Getwd(); err == nil {
			proc.Dir = cwd
		}
		proc.Env = append(os.Environ(), "XELYON_CHANGED_FILES="+changedFilesEnv)

		output, err := runFinalCheckProcess(commandCtx, proc)
		cancel()

		if err != nil {
			outputStr := finalcheck.TruncateOutput(string(output))

			exitCode := -1
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}

			checks = append(checks, a.recordFinalCheckCommandResult(cmd, exitCode, finalcheck.CommandStatusFailed, outputStr))
			if parentErr := ctx.Err(); parentErr != nil {
				red.Fprintf(out, "  Final check cancelled: %s\n", cmd)
				return finalcheck.BuildCancelledResult(parentErr, checks)
			}
			red.Fprintf(out, "  Final check failed (exit code %d): %s\n", exitCode, cmd)

			passed := false
			a.taskTestResult = &passed
			result := finalcheck.BuildFailureResult(cmd, exitCode, string(output), diffOutput)
			result.Checks = checks
			return result
		}

		checks = append(checks, a.recordFinalCheckCommandResult(cmd, 0, finalcheck.CommandStatusPassed, string(output)))
		green.Fprintf(out, "  Final check passed: %s\n", cmd)
	}
	passed := true
	a.taskTestResult = &passed

	return finalcheck.RunResult{Checks: checks}
}

func (a *Agent) recordFinalCheckCommandResult(command string, exitCode int, status string, output string) finalcheck.CommandResult {
	a.recordFinalCheckObservation(taskstate.TestObservation{
		Command:  command,
		ExitCode: exitCode,
		Status:   status,
		Output:   output,
	})
	return finalcheck.CommandResult{
		Command:  command,
		ExitCode: exitCode,
		Status:   status,
	}
}

func (a *Agent) recordFinalCheckObservation(observation taskstate.TestObservation) {
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		return
	}
	a.Runtime.TaskLedger.Recorder().RecordTestObservation(observation)
}

func runFinalCheckProcess(ctx context.Context, proc *exec.Cmd) ([]byte, error) {
	var output bytes.Buffer
	proc.Stdout = &output
	proc.Stderr = &output

	if err := proc.Start(); err != nil {
		return nil, err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- proc.Wait()
	}()

	select {
	case err := <-errCh:
		return output.Bytes(), err
	case <-ctx.Done():
		killFinalCheckProcessGroup(proc)
		err := <-errCh
		if err == nil {
			err = ctx.Err()
		}
		return output.Bytes(), err
	}
}
