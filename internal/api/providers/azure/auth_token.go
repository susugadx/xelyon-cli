package azure

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const defaultAuthTokenCommandTimeout = 10 * time.Second

var azureAuthJWTPattern = regexp.MustCompile(`[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)

func (p *Provider) currentAuthToken(ctx context.Context) (string, error) {
	if token := strings.TrimSpace(p.authToken); token != "" {
		return token, nil
	}
	if strings.TrimSpace(p.authTokenCommand) == "" {
		return "", fmt.Errorf("%s, %s, or %s not set", apiKeyEnv, authTokenEnv, authTokenCommandEnv)
	}
	if err := p.refreshAuthToken(ctx); err != nil {
		return "", err
	}
	return p.authToken, nil
}

func (p *Provider) refreshAuthToken(ctx context.Context) error {
	token, err := runAzureAuthTokenCommand(ctx, p.authTokenCommand, p.authTokenCommandTimeout)
	if err != nil {
		return err
	}
	p.authToken = token
	return nil
}

func (p *Provider) canRefreshAuthToken() bool {
	return strings.TrimSpace(p.APIKey) == "" && strings.TrimSpace(p.authTokenCommand) != ""
}

func runAzureAuthTokenCommand(ctx context.Context, command string, timeout time.Duration) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("%s is empty", authTokenCommandEnv)
	}
	if timeout <= 0 {
		timeout = defaultAuthTokenCommandTimeout
	}

	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := azureAuthShellCommand(commandCtx, command)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if commandCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("%s timed out after %s", authTokenCommandEnv, timeout)
		}
		if commandCtx.Err() != nil {
			return "", fmt.Errorf("%s canceled: %w", authTokenCommandEnv, commandCtx.Err())
		}
		detail := sanitizeAzureAuthCommandOutput(stderr.String())
		if detail == "" {
			return "", fmt.Errorf("%s failed: %w", authTokenCommandEnv, err)
		}
		return "", fmt.Errorf("%s failed: %w: %s", authTokenCommandEnv, err, detail)
	}

	token := firstNonEmptyLine(stdout.String())
	if token == "" {
		return "", fmt.Errorf("%s produced empty stdout", authTokenCommandEnv)
	}
	return token, nil
}

func azureAuthTokenCommandTimeout() time.Duration {
	timeout, _ := parseAzureAuthTokenCommandTimeout()
	return timeout
}

func parseAzureAuthTokenCommandTimeout() (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(authTokenCommandTimeoutEnv))
	if raw == "" {
		return defaultAuthTokenCommandTimeout, nil
	}
	timeout, err := time.ParseDuration(raw)
	if err != nil || timeout <= 0 {
		if err == nil {
			err = fmt.Errorf("must be positive")
		}
		return defaultAuthTokenCommandTimeout, fmt.Errorf("%s=%q is invalid: %w", authTokenCommandTimeoutEnv, raw, err)
	}
	return timeout, nil
}

func firstNonEmptyLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func sanitizeAzureAuthCommandOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}
	output = azureAuthJWTPattern.ReplaceAllString(output, "[REDACTED]")
	message := api.SanitizeErrorMessage([]byte(output), 0).Error()
	message = strings.TrimPrefix(message, "API error (0): ")
	const maxLen = 400
	if len(message) > maxLen {
		message = message[:maxLen] + "... (truncated)"
	}
	return message
}
