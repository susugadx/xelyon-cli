package version

import (
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "returns Version variable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := GetVersion()
			if version == "" {
				t.Error("GetVersion() returned empty string")
			}

			// デフォルト値または GoReleaser によってセットされた値を確認
			if version != Version {
				t.Errorf("GetVersion() = %v, want %v", version, Version)
			}
		})
	}
}

func TestGetFullVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		wantFmt string
	}{
		{
			name:    "default values",
			version: "0.31.0-dev",
			commit:  "unknown",
			date:    "unknown",
			wantFmt: "0.31.0-dev (commit: unknown, built: unknown)",
		},
		{
			name:    "custom values",
			version: "1.0.0",
			commit:  "abc123",
			date:    "2024-01-01",
			wantFmt: "1.0.0 (commit: abc123, built: 2024-01-01)",
		},
		{
			name:    "with hyphen in version",
			version: "0.15.0-beta",
			commit:  "def456",
			date:    "2024-06-15",
			wantFmt: "0.15.0-beta (commit: def456, built: 2024-06-15)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// バックアップして復元
			oldVersion := Version
			oldCommit := Commit
			oldDate := Date
			defer func() {
				Version = oldVersion
				Commit = oldCommit
				Date = oldDate
			}()

			// テスト用の値をセット
			Version = tt.version
			Commit = tt.commit
			Date = tt.date

			got := GetFullVersion()
			if got != tt.wantFmt {
				t.Errorf("GetFullVersion() = %v, want %v", got, tt.wantFmt)
			}

			// フォーマット検証
			if !strings.Contains(got, tt.version) {
				t.Errorf("GetFullVersion() should contain version %v, got %v", tt.version, got)
			}
			if !strings.Contains(got, tt.commit) {
				t.Errorf("GetFullVersion() should contain commit %v, got %v", tt.commit, got)
			}
			if !strings.Contains(got, tt.date) {
				t.Errorf("GetFullVersion() should contain date %v, got %v", tt.date, got)
			}
		})
	}
}

func TestVersionConstants(t *testing.T) {
	tests := []struct {
		name     string
		variable *string
		varName  string
	}{
		{name: "Version is not empty", variable: &Version, varName: "Version"},
		{name: "Commit is not empty", variable: &Commit, varName: "Commit"},
		{name: "Date is not empty", variable: &Date, varName: "Date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if *tt.variable == "" {
				t.Errorf("%s is empty string", tt.varName)
			}
		})
	}
}

func TestGetVersion_ConsistencyWithVariable(t *testing.T) {
	// GetVersion() が直接 Version 変数を返すことを確認
	if GetVersion() != Version {
		t.Error("GetVersion() should return Version variable directly")
	}
}

func TestGetFullVersion_Format(t *testing.T) {
	// 実際のパッケージ変数を使ったフォーマット検証
	fullVersion := GetFullVersion()

	// 形式: "{version} (commit: {commit}, built: {date})"
	if !strings.Contains(fullVersion, "(commit:") {
		t.Error("GetFullVersion() should contain '(commit:' substring")
	}
	if !strings.Contains(fullVersion, "built:") {
		t.Error("GetFullVersion() should contain 'built:' substring")
	}
	if !strings.HasPrefix(fullVersion, Version) {
		t.Errorf("GetFullVersion() should start with version %v, got %v", Version, fullVersion)
	}
}
