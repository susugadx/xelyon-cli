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
		want [versionParts]int
	}{
		{
			name: "standard version",
			v:    "0.31.0",
			want: [versionParts]int{0, 31, 0},
		},
		{
			name: "version with dev suffix",
			v:    "0.31.0-dev",
			want: [versionParts]int{0, 31, 0},
		},
		{
			name: "version with build metadata",
			v:    "1.2.3+20130313144700",
			want: [versionParts]int{1, 2, 3},
		},
		{
			name: "two-part version",
			v:    "1.0",
			want: [versionParts]int{1, 0, 0},
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
		githubAPIURL = server.URL
		defer func() {
			githubAPIURL = oldURL
		}()

		version, err := fetchLatestVersion()
		if err != nil {
			t.Fatalf("fetchLatestVersion() error = %v", err)
		}
		if version != "v0.32.0" {
			t.Errorf("fetchLatestVersion() = %q, want %q", version, "v0.32.0")
		}
	})

	t.Run("network error", func(t *testing.T) {
		// Override with invalid URL
		oldURL := githubAPIURL
		githubAPIURL = "http://localhost:1" // Invalid port
		defer func() {
			githubAPIURL = oldURL
		}()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Error("fetchLatestVersion() expected error for invalid URL")
		}
	})

	t.Run("non-200 status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		oldURL := githubAPIURL
		githubAPIURL = server.URL
		defer func() {
			githubAPIURL = oldURL
		}()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Error("fetchLatestVersion() expected error for non-200 status")
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		oldURL := githubAPIURL
		githubAPIURL = server.URL
		defer func() {
			githubAPIURL = oldURL
		}()

		_, err := fetchLatestVersion()
		if err == nil {
			t.Error("fetchLatestVersion() expected error for invalid JSON")
		}
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
	t.Run("successful update", func(t *testing.T) {
		tmpDir := t.TempDir()
		checkFile := filepath.Join(tmpDir, versionCheckFile)

		err := updateLastCheckTime(tmpDir)
		if err != nil {
			t.Fatalf("updateLastCheckTime() error = %v", err)
		}

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
	})

	t.Run("creates parent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		nestedDir := filepath.Join(tmpDir, "nested", "dir")

		err := updateLastCheckTime(nestedDir)
		if err != nil {
			t.Fatalf("updateLastCheckTime() error = %v", err)
		}

		checkFile := filepath.Join(nestedDir, versionCheckFile)
		if _, err := os.Stat(checkFile); os.IsNotExist(err) {
			t.Error("updateLastCheckTime() did not create nested directory structure")
		}
	})
}

func TestGetUpdateCommand(t *testing.T) {
	cmd := getUpdateCommand()
	if cmd == "" {
		t.Error("getUpdateCommand() returned empty string")
	}
	// The actual value depends on runtime.GOOS, so just verify it's not empty
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

func TestCheckForUpdates(t *testing.T) {
	t.Run("update available", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			release := GitHubRelease{
				TagName: "v99.99.99", // Higher than any current version
				HTMLURL: "https://github.com/susugadx/xelyon-cli/releases/tag/v99.99.99",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(release)
		}))
		defer server.Close()

		// Temporarily override the API URL
		oldURL := githubAPIURL
		githubAPIURL = server.URL
		defer func() {
			githubAPIURL = oldURL
		}()

		tmpDir := t.TempDir()
		result, err := CheckForUpdates(tmpDir)
		if err != nil {
			t.Fatalf("CheckForUpdates() error = %v", err)
		}
		if result == nil {
			t.Fatal("CheckForUpdates() returned nil result")
		}
		if !result.HasUpdate {
			t.Error("CheckForUpdates() HasUpdate = false, want true")
		}
		if result.LatestVersion != "v99.99.99" {
			t.Errorf("CheckForUpdates() LatestVersion = %q, want %q", result.LatestVersion, "v99.99.99")
		}
	})

	t.Run("no update available", func(t *testing.T) {
		// Create mock server returning current version
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			release := GitHubRelease{
				TagName: GetVersion(), // Same as current version
				HTMLURL: "https://github.com/susugadx/xelyon-cli/releases/tag/" + GetVersion(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(release)
		}))
		defer server.Close()

		oldURL := githubAPIURL
		githubAPIURL = server.URL
		defer func() {
			githubAPIURL = oldURL
		}()

		tmpDir := t.TempDir()
		result, err := CheckForUpdates(tmpDir)
		if err != nil {
			t.Fatalf("CheckForUpdates() error = %v", err)
		}
		if result == nil {
			t.Fatal("CheckForUpdates() returned nil result")
		}
		if result.HasUpdate {
			t.Error("CheckForUpdates() HasUpdate = true, want false")
		}
	})

	t.Run("cooldown period active", func(t *testing.T) {
		tmpDir := t.TempDir()
		checkFile := filepath.Join(tmpDir, versionCheckFile)

		// Create recent check file
		_ = os.WriteFile(checkFile, []byte(time.Now().Format(time.RFC3339)), 0644)

		// Should return nil due to cooldown
		result, err := CheckForUpdates(tmpDir)
		if err != nil {
			t.Fatalf("CheckForUpdates() error = %v", err)
		}
		if result != nil {
			t.Error("CheckForUpdates() should return nil during cooldown period")
		}
	})

	t.Run("network error returns nil", func(t *testing.T) {
		oldURL := githubAPIURL
		githubAPIURL = "http://localhost:1" // Invalid port
		defer func() {
			githubAPIURL = oldURL
		}()

		tmpDir := t.TempDir()
		result, err := CheckForUpdates(tmpDir)
		if err != nil {
			t.Fatalf("CheckForUpdates() error = %v, expected nil", err)
		}
		if result != nil {
			t.Error("CheckForUpdates() should return nil on network error")
		}
	})
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
