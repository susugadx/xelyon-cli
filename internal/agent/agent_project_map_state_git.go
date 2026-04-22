package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"time"
)

func gitProjectMapHEAD(rootPath string) string {
	return gitProjectMapCommandDigest(rootPath, []string{"rev-parse", "HEAD"})
}

func gitProjectMapStatusDigest(rootPath string) string {
	return gitProjectMapCommandDigest(rootPath, []string{"status", "--porcelain"})
}

func gitProjectMapCommandDigest(rootPath string, args []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = rootPath

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}

	sum := sha256.Sum256(bytes.TrimSpace(stdout.Bytes()))
	return hex.EncodeToString(sum[:])
}

func isGitProjectMapAvailable(rootPath string) bool {
	return gitProjectMapHEAD(rootPath) != "" || gitProjectMapStatusDigest(rootPath) != ""
}
