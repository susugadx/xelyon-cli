package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	versionCheckFile  = "version_check"
	checkCooldownDays = 1
	versionParts      = 3 // major, minor, patch
)

// githubAPIURL is a var to allow overriding in tests
var githubAPIURL = "https://api.github.com/repos/susugadx/xelyon-cli/releases/latest"

// GitHubRelease represents the GitHub API response for a release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// VersionCheckResult contains the result of a version check
type VersionCheckResult struct {
	HasUpdate      bool
	LatestVersion  string
	CurrentVersion string
	UpdateCommand  string
}

// CheckForUpdates checks if a new version is available
func CheckForUpdates(configDir string) (*VersionCheckResult, error) {
	// Check if cooldown period has passed
	if !shouldCheck(configDir) {
		return nil, nil
	}

	// Fetch latest version from GitHub
	latestVersion, err := fetchLatestVersion()
	if err != nil {
		// Network errors are silently ignored
		return nil, nil
	}

	// Update last check time (error is intentionally ignored to maintain existing behavior)
	_ = updateLastCheckTime(configDir)

	// Compare versions
	currentVersion := GetVersion()
	result := &VersionCheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		UpdateCommand:  getUpdateCommand(),
	}

	// Remove 'v' prefix for comparison if present
	current := strings.TrimPrefix(currentVersion, "v")
	latest := strings.TrimPrefix(latestVersion, "v")

	// Simple version comparison (assumes semantic versioning)
	if compareVersions(current, latest) < 0 {
		result.HasUpdate = true
	}

	return result, nil
}

// shouldCheck determines if enough time has passed since last check
func shouldCheck(configDir string) bool {
	checkFile := filepath.Join(configDir, versionCheckFile)

	stat, err := os.Stat(checkFile)
	if err != nil {
		// File doesn't exist or error reading - should check
		return true
	}

	// Check if cooldown period has passed
	timeSinceCheck := time.Since(stat.ModTime())
	return timeSinceCheck > time.Duration(checkCooldownDays)*24*time.Hour
}

// updateLastCheckTime updates the version check file timestamp
func updateLastCheckTime(configDir string) error {
	checkFile := filepath.Join(configDir, versionCheckFile)

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Touch the file to update timestamp
	if err := os.WriteFile(checkFile, []byte(time.Now().Format(time.RFC3339)), 0644); err != nil {
		return fmt.Errorf("failed to write version check file: %w", err)
	}

	return nil
}

// fetchLatestVersion fetches the latest version from GitHub Releases API
func fetchLatestVersion() (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return "", err
	}

	// Set User-Agent to avoid rate limiting
	req.Header.Set("User-Agent", "xelyon-cli")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

// getUpdateCommand returns the appropriate update command for the current OS
func getUpdateCommand() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew upgrade xelyon"
	default:
		return "https://github.com/susugadx/xelyon-cli/releases からダウンロード"
	}
}

// compareVersions compares two semantic version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	// Parse version strings (assumes format: X.Y.Z or X.Y.Z-suffix)
	parts1 := parseVersion(v1)
	parts2 := parseVersion(v2)

	// Compare each part
	for i := 0; i < versionParts; i++ {
		if parts1[i] < parts2[i] {
			return -1
		}
		if parts1[i] > parts2[i] {
			return 1
		}
	}

	return 0
}

// parseVersion extracts major, minor, patch from version string
func parseVersion(v string) [versionParts]int {
	var parts [versionParts]int

	// Remove any suffix (e.g., "-dev", "-beta")
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}

	// Split by dots
	components := strings.Split(v, ".")
	for i := 0; i < len(components) && i < versionParts; i++ {
		_, _ = fmt.Sscanf(components[i], "%d", &parts[i])
	}

	return parts
}

// FormatUpdateNotification formats the update notification message
func FormatUpdateNotification(result *VersionCheckResult) string {
	if result == nil || !result.HasUpdate {
		return ""
	}

	return fmt.Sprintf(`⚠️  新しいバージョン %s があります
   現在: %s
   更新: %s
`,
		result.LatestVersion,
		result.CurrentVersion,
		result.UpdateCommand,
	)
}
