package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

type finalCheckRunResult struct {
	needsContinue      bool
	feedback           string
	failureFingerprint string
}

// runFinalCheckCommands は config.yaml の final_checks.commands に定義された
// シェルコマンドを順番に実行する。いずれかのコマンドが失敗した場合、
// needsContinue=true と AI 向けのフィードバックを返す。
// すべて成功した場合は needsContinue=false を返す。
func (a *Agent) runFinalCheckCommands(changedFiles []string) finalCheckRunResult {
	cfg := a.cfg()
	commands := cfg.FinalChecks.Commands
	out := a.output()

	// No final checks → nothing to run
	if len(commands) == 0 {
		return finalCheckRunResult{}
	}
	a.taskTestCommand = strings.Join(commands, " && ")

	// git diff --stat: final checks 失敗時のコンテキスト用
	yellow.Fprintln(out, "📊 Running final checks with git diff --stat...")
	var diffOutput string
	diffCmd := exec.Command("git", "diff", "--stat")
	if output, err := diffCmd.CombinedOutput(); err == nil {
		diffOutput = string(output)
		if strings.TrimSpace(diffOutput) != "" {
			_, _ = fmt.Fprintln(out, diffOutput)
		}
	}

	// untracked files（write_file で新規作成されたファイル用）
	untrackedCmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	if untrackedOut, err := untrackedCmd.CombinedOutput(); err == nil {
		untrackedStr := strings.TrimSpace(string(untrackedOut))
		if untrackedStr != "" {
			_, _ = fmt.Fprintf(out, "New files (untracked):\n%s\n", untrackedStr)
			diffOutput += "\nNew files (untracked):\n" + untrackedStr
		}
	}

	timeout := time.Duration(cfg.FinalChecks.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 600 * time.Second
	}

	changedFilesEnv := strings.Join(changedFiles, " ")

	for _, cmd := range commands {
		yellow.Fprintf(out, "🧪 Running final check command: %s\n", cmd)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		proc := exec.CommandContext(ctx, "bash", "-c", cmd)
		setFinalCheckProcessGroup(proc)

		if cwd, err := os.Getwd(); err == nil {
			proc.Dir = cwd
		}
		proc.Env = append(os.Environ(), "XELYON_CHANGED_FILES="+changedFilesEnv)

		output, err := runFinalCheckProcess(ctx, proc)
		cancel()

		if err != nil {
			outputStr := string(output)
			if len(outputStr) > 2000 {
				outputStr = outputStr[:2000] + "\n... (truncated)"
			}

			exitCode := -1
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}

			a.recordFinalCheckObservation(taskstate.TestObservation{
				Command:  cmd,
				ExitCode: exitCode,
				Status:   "failed",
				Output:   outputStr,
			})
			red.Fprintf(out, "  Final check failed (exit code %d): %s\n", exitCode, cmd)

			feedback := fmt.Sprintf(`[SYSTEM] Final check failed. Command %q failed (exit code %d):

%s

[Context] git diff --stat:
%s

Please fix these errors before declaring completion. Do NOT skip these issues.`, cmd, exitCode, outputStr, diffOutput)
			passed := false
			a.taskTestResult = &passed
			return finalCheckRunResult{
				needsContinue:      true,
				feedback:           feedback,
				failureFingerprint: finalCheckFailureFingerprint(cmd, exitCode, outputStr),
			}
		}

		a.recordFinalCheckObservation(taskstate.TestObservation{
			Command:  cmd,
			ExitCode: 0,
			Status:   "passed",
			Output:   string(output),
		})
		green.Fprintf(out, "  Final check passed: %s\n", cmd)
	}
	passed := true
	a.taskTestResult = &passed

	return finalCheckRunResult{}
}

func (a *Agent) recordFinalCheckObservation(observation taskstate.TestObservation) {
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		return
	}
	a.Runtime.TaskLedger.Recorder().RecordTestObservation(observation)
}

func finalCheckFailureFingerprint(command string, exitCode int, output string) string {
	return errorFingerprint(fmt.Sprintf("%s\nexit=%d\n%s", command, exitCode, output))
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
