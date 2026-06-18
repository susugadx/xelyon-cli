package tools

import "testing"

func TestIsParallelSafe_StaticTools(t *testing.T) {
	tests := []struct {
		tool     string
		expected bool
	}{
		{"read_file", true},
		{"read_files", true},
		{"list_dir", true},
		{"search_code", true},
		{"web_search", true},
		{"git_status", true},
		{"git_log", true},
		{"git_diff", true},
		{"spawn_agent", true},
		{"wait_agent", true},
		{"apply_patch", false},
		{"write_file", false},
		{"str_replace", false},
		{"delete_file", false},
		{"ask_user_question", false},
		{"unknown_tool", false},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			tc := &ToolCall{Tool: tt.tool, Args: map[string]string{}}
			got := IsParallelSafe(tc)
			if got != tt.expected {
				t.Errorf("IsParallelSafe(%q) = %v, want %v", tt.tool, got, tt.expected)
			}
		})
	}
}

func TestIsParallelSafe_BashReadOnly(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		// parallel-safe bash commands
		{"pwd", true},
		{"ls", true},
		{"ls -la", true},
		{"find . -name '*.go'", true},
		{"rg 'pattern' src/", true},
		{"grep -r 'TODO' .", true},
		{"cat README.md", true},
		{"head -20 file.go", true},
		{"tail -10 file.go", true},
		{"wc -l file.go", true},
		{"git status", true},
		{"git diff", true},
		{"git diff --stat", true},
		{"git log", true},
		{"git log --oneline -10", true},
		{"git branch", true},
		{"git ls-files", true},
		{"git show HEAD", true},
		{"git remote -v", true},

		// NOT parallel-safe bash commands
		{"rm -rf /tmp/foo", false},
		{"echo hello", false},
		{"go build ./...", false},
		{"npm install", false},
		{"make", false},
		{"", false},

		// Pipes/redirects/chains are never parallel-safe
		{"cat file.go | grep TODO", false},
		{"ls > output.txt", false},
		{"ls >> output.txt", false},
		{"ls && pwd", false},
		{"ls; pwd", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			tc := &ToolCall{
				Tool: "bash",
				Args: map[string]string{"command": tt.command},
			}
			got := IsParallelSafe(tc)
			if got != tt.expected {
				t.Errorf("IsParallelSafe(bash %q) = %v, want %v", tt.command, got, tt.expected)
			}
		})
	}
}

func TestIsParallelSafe_BashFromRawArgs(t *testing.T) {
	// Args が未設定で RawArgs のみの場合
	tc := &ToolCall{
		Tool:    "bash",
		RawArgs: map[string]any{"command": "git status"},
		Args:    map[string]string{}, // 空
	}
	if !IsParallelSafe(tc) {
		t.Error("should be parallel-safe when command is in RawArgs")
	}
}

func TestClassifyToolCalls(t *testing.T) {
	toolCalls := []*ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "a.go"}},                              // 0: parallel
		{Tool: "write_file", Args: map[string]string{"path": "b.go"}},                             // 1: sequential
		{Tool: "search_code", Args: map[string]string{"pattern": "foo"}},                          // 2: parallel
		{Tool: "bash", Args: map[string]string{"command": "git status"}},                          // 3: parallel (read-only bash)
		{Tool: "bash", Args: map[string]string{"command": "go build ./..."}},                      // 4: sequential (non-read-only bash)
		{Tool: "list_dir", Args: map[string]string{"path": "."}},                                  // 5: parallel
		{Tool: "str_replace", Args: map[string]string{"path": "c.go", "old_str": "foo"}},          // 6: sequential
		{Tool: "apply_patch", Args: map[string]string{"patch": "*** Begin Patch\n*** End Patch"}}, // 7: sequential
	}

	parallelIdx, seqIdx := ClassifyToolCalls(toolCalls)

	expectedParallel := []int{0, 2, 3, 5}
	expectedSeq := []int{1, 4, 6, 7}

	if len(parallelIdx) != len(expectedParallel) {
		t.Fatalf("parallel count = %d, want %d", len(parallelIdx), len(expectedParallel))
	}
	for i, idx := range parallelIdx {
		if idx != expectedParallel[i] {
			t.Errorf("parallelIdx[%d] = %d, want %d", i, idx, expectedParallel[i])
		}
	}

	if len(seqIdx) != len(expectedSeq) {
		t.Fatalf("sequential count = %d, want %d", len(seqIdx), len(expectedSeq))
	}
	for i, idx := range seqIdx {
		if idx != expectedSeq[i] {
			t.Errorf("seqIdx[%d] = %d, want %d", i, idx, expectedSeq[i])
		}
	}
}

func TestIsBashParallelSafe(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		{"pwd", true},
		{"ls", true},
		{"ls -la /tmp", true},
		{"find . -name '*.go'", true},
		{"rg pattern src/", true},
		{"grep -rn TODO .", true},
		{"cat file.txt", true},
		{"head -20 main.go", true},
		{"tail -5 log.txt", true},
		{"wc -l file.go", true},
		{"git status", true},
		{"git diff", true},
		{"git diff --stat", true},
		{"git log", true},
		{"git log --oneline -10", true},
		{"git branch", true},
		{"git ls-files", true},
		{"git show HEAD:main.go", true},
		{"git remote -v", true},

		// NOT parallel-safe
		{"echo hello", false},
		{"go build ./...", false},
		{"npm install", false},
		{"rm file.txt", false},
		{"mkdir -p dir", false},
		{"touch file.txt", false},
		{"", false},

		// Pipes/redirects/chains
		{"cat file | grep foo", false},
		{"ls > out.txt", false},
		{"ls >> out.txt", false},
		{"pwd && ls", false},
		{"pwd; ls", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := IsBashParallelSafe(tt.command)
			if got != tt.expected {
				t.Errorf("IsBashParallelSafe(%q) = %v, want %v", tt.command, got, tt.expected)
			}
		})
	}
}
