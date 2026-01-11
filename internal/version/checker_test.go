package version

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want int
	}{
		{
			name: "equal versions",
			v1:   "0.31.0",
			v2:   "0.31.0",
			want: 0,
		},
		{
			name: "v1 less than v2",
			v1:   "0.31.0",
			v2:   "0.32.0",
			want: -1,
		},
		{
			name: "v1 greater than v2",
			v1:   "0.32.0",
			v2:   "0.31.0",
			want: 1,
		},
		{
			name: "minor version difference",
			v1:   "0.30.0",
			v2:   "0.31.0",
			want: -1,
		},
		{
			name: "patch version difference",
			v1:   "0.31.0",
			v2:   "0.31.1",
			want: -1,
		},
		{
			name: "major version difference",
			v1:   "1.0.0",
			v2:   "2.0.0",
			want: -1,
		},
		{
			name: "version with suffix",
			v1:   "0.31.0-dev",
			v2:   "0.31.0",
			want: 0, // Suffix ignored
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareVersions(tt.v1, tt.v2)
			if got != tt.want {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want [3]int
	}{
		{
			name: "standard version",
			v:    "0.31.0",
			want: [3]int{0, 31, 0},
		},
		{
			name: "version with dev suffix",
			v:    "0.31.0-dev",
			want: [3]int{0, 31, 0},
		},
		{
			name: "version with build metadata",
			v:    "1.2.3+20130313144700",
			want: [3]int{1, 2, 3},
		},
		{
			name: "two-part version",
			v:    "1.0",
			want: [3]int{1, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseVersion(tt.v)
			if got != tt.want {
				t.Errorf("parseVersion(%q) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestFetchLatestVersion(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("User-Agent") != "xelyon-cli" {
				t.Error("User-Agent header not set correctly")
			}

			release := GitHubRelease{
				TagName: "v0.32.0",
				HTMLURL: "https://github.com/susugadx/xelyon-cli/releases/tag/v0.32.0",
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(release)
		}))
		defer server.Close()

		// Temporarily override the API URL
		oldURL := githubAPIURL
		defer func() {
			// Restore (note: this won't work due to const, but demonstrates intent)
			_ = oldURL
		}()

		// For testing, we need to modify the function to accept URL as parameter
		// For now, test with the real implementation
		t.Skip("Skipping as githubAPIURL is const - need to refactor for testability")
	})

	t.Run("network error", func(t *testing.T) {
		// Override with invalid URL
		t.Skip("Skipping as githubAPIURL is const - need to refactor for testability")
	})
}

func TestShouldCheck(t *testing.T) {
	t.Run("no previous check", func(t *testing.T) {
		tmpDir := t.TempDir()

		if !shouldCheck(tmpDir) {
			t.Error("shouldCheck() = false, want true when no previous check exists")
		}
	})

	t.Run("recent check within cooldown", func(t *testing.T) {
		tmpDir := t.TempDir()
		checkFile := filepath.Join(tmpDir, versionCheckFile)

		// Create check file with current time
		_ = os.WriteFile(checkFile, []byte(time.Now().Format(time.RFC3339)), 0644)

		if shouldCheck(tmpDir) {
			t.Error("shouldCheck() = true, want false when check is within cooldown period")
		}
	})

	t.Run("old check past cooldown", func(t *testing.T) {
		tmpDir := t.TempDir()
		checkFile := filepath.Join(tmpDir, versionCheckFile)

		// Create check file with old time
		oldTime := time.Now().Add(-2 * 24 * time.Hour)
		_ = os.WriteFile(checkFile, []byte(oldTime.Format(time.RFC3339)), 0644)

		// Set file modification time to old time
		_ = os.Chtimes(checkFile, oldTime, oldTime)

		if !shouldCheck(tmpDir) {
			t.Error("shouldCheck() = false, want true when check is past cooldown period")
		}
	})
}

func TestUpdateLastCheckTime(t *testing.T) {
	tmpDir := t.TempDir()
	checkFile := filepath.Join(tmpDir, versionCheckFile)

	updateLastCheckTime(tmpDir)

	// Check if file was created
	if _, err := os.Stat(checkFile); os.IsNotExist(err) {
		t.Error("updateLastCheckTime() did not create check file")
	}

	// Check if file contains timestamp
	content, err := os.ReadFile(checkFile)
	if err != nil {
		t.Fatalf("Failed to read check file: %v", err)
	}

	if len(content) == 0 {
		t.Error("updateLastCheckTime() created empty check file")
	}

	// Verify it's a valid RFC3339 timestamp
	_, err = time.Parse(time.RFC3339, string(content))
	if err != nil {
		t.Errorf("updateLastCheckTime() wrote invalid timestamp: %v", err)
	}
}

func TestFormatUpdateNotification(t *testing.T) {
	tests := []struct {
		name   string
		result *VersionCheckResult
		want   string
	}{
		{
			name:   "nil result",
			result: nil,
			want:   "",
		},
		{
			name: "no update available",
			result: &VersionCheckResult{
				HasUpdate:      false,
				CurrentVersion: "v0.31.0",
				LatestVersion:  "v0.31.0",
			},
			want: "",
		},
		{
			name: "update available",
			result: &VersionCheckResult{
				HasUpdate:      true,
				CurrentVersion: "v0.31.0",
				LatestVersion:  "v0.32.0",
				UpdateCommand:  "brew upgrade xelyon",
			},
			want: `⚠️  新しいバージョン v0.32.0 があります
   現在: v0.31.0
   更新: brew upgrade xelyon
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatUpdateNotification(tt.result)
			if got != tt.want {
				t.Errorf("FormatUpdateNotification() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckForUpdates_Integration(t *testing.T) {
	// This is an integration test that requires network access
	t.Skip("Skipping integration test - requires network and may hit rate limits")

	tmpDir := t.TempDir()

	result, err := CheckForUpdates(tmpDir)
	if err != nil {
		t.Fatalf("CheckForUpdates() error = %v", err)
	}

	// Result may be nil if cooldown period hasn't passed
	if result != nil {
		t.Logf("Current: %s, Latest: %s, HasUpdate: %v",
			result.CurrentVersion, result.LatestVersion, result.HasUpdate)
	}
}
