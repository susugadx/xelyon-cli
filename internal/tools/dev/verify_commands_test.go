package dev

import "testing"

func TestIsVerifyCommand(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		// 単体コマンド — verify
		{"go test ./...", true},
		{"go build ./...", true},
		{"go fmt ./...", true},
		{"go vet ./...", true},
		{"npm test", true},
		{"pytest", true},
		{"cargo test", true},
		{"make", true},
		{"make ci-check", true},
		{"sqlfluff lint .", true},
		{"prisma migrate deploy", true},
		{"golangci-lint run", true},
		{"docker build .", true},

		// 非verify
		{"git diff", false},
		{"cat file.go", false},
		{"ls -la", false},
		{"git log", false},
		{"echo hello", false},
		{"curl http://example.com", false},

		// コマンドチェーン — 全verify（splitChainCommand で分割される）
		{"go fmt ./... && go build ./... && go test ./...", true},
		{"go fmt ./... && go build ./...", true},

		// 非verifyを含むチェーン
		{"go test ./... && git diff", false},
		{"go build ./... || echo 'failed'", false},

		// エッジケース
		{"", false},
		{"   go test ./...   ", true}, // 前後空白

		// プレフィックスマッチの境界
		{"gotest", false},    // "go test" のプレフィックスでない
		{"makefiles", false}, // "make" の後に空白/タブなし
		{"make ci-check", true},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := IsVerifyCommand(tt.command)
			if got != tt.want {
				t.Errorf("IsVerifyCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}
