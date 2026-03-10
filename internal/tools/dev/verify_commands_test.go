package dev

import "testing"

func TestIsVerifyCommand(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		// 単体コマンド — verify（プレフィックスマッチ）
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

		// 非verify（nonVerifyPrefixesで除外）
		{"git diff", false},
		{"cat file.go", false},
		{"ls -la", false},
		{"git log", false},
		{"echo hello", false},
		{"curl http://example.com", false},

		// コマンドチェーン — 全verify
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

func TestIsVerifyCommand_NonVerify(t *testing.T) {
	nonVerifyCmds := []string{
		"cat file.go",
		"grep -rn 'pattern' .",
		"curl http://example.com",
		"psql -c 'SELECT 1'",
		"kubectl get pods",
		"kubectl describe pod my-pod",
		"docker inspect container",
		"aws describe-instances",
		"terraform state list",
		"git diff HEAD~3",
		"openssl x509 -text",
		"git status",
		"hyperfine ./bench",
	}
	for _, cmd := range nonVerifyCmds {
		t.Run(cmd, func(t *testing.T) {
			if IsVerifyCommand(cmd) {
				t.Errorf("IsVerifyCommand(%q) = true, want false (non-verify)", cmd)
			}
		})
	}
}

func TestIsVerifyCommand_KeywordFallback(t *testing.T) {
	keywordCmds := []string{
		"some-new-tool build",
		"custom-ci deploy staging",
		"my-tool install --global",
		"foo-lint check src/",
		"internal-tool migrate up",
		"my-cli generate models",
		"unknown-tool test --verbose",
	}
	for _, cmd := range keywordCmds {
		t.Run(cmd, func(t *testing.T) {
			if !IsVerifyCommand(cmd) {
				t.Errorf("IsVerifyCommand(%q) = false, want true (keyword fallback)", cmd)
			}
		})
	}
}

func TestIsVerifyCommand_NonVerifyOverridesKeyword(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"echo test", false},       // "echo" が非verify、"test" キーワードより優先
		{"cat install.log", false}, // "cat" が非verify
		{"grep build .", false},    // "grep" が非verify
		{"psql -c 'SELECT 1'", false},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			if got := IsVerifyCommand(tt.command); got != tt.want {
				t.Errorf("IsVerifyCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestIsVerifyCommand_InfraCommands(t *testing.T) {
	verifyCmds := []string{
		"kubectl apply -f deployment.yaml",
		"helm install my-release chart",
		"aws s3 cp file.txt s3://bucket/",
		"terraform apply -auto-approve",
		"terraform destroy",
		"ansible-playbook site.yml",
		"docker push myimage:latest",
		"docker compose up -d",
		"gcloud app deploy",
		"az webapp deploy",
		"npm install",
		"pip3 install requests",
		"brew install go",
		"pg_dump mydb",
		"mongorestore --dir dump/",
		"trivy image myimage:latest",
		"hadolint Dockerfile",
		"gh pr create --fill",
		"go generate ./...",
	}
	for _, cmd := range verifyCmds {
		t.Run(cmd, func(t *testing.T) {
			if !IsVerifyCommand(cmd) {
				t.Errorf("IsVerifyCommand(%q) = false, want true", cmd)
			}
		})
	}
}

func TestMatchesNonVerifyPrefix(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"echo hello", true},
		{"cat file.go", true},
		{"kubectl get pods", true},
		{"docker inspect foo", true},
		{"go test ./...", false},
		{"npm test", false},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			if got := matchesNonVerifyPrefix(tt.command); got != tt.want {
				t.Errorf("matchesNonVerifyPrefix(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestIsVerifyByKeyword(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"mytool build", true},
		{"custom deploy staging", true},
		{"tool install pkg", true},
		{"random-cmd unknown-arg", false},
		{"cat install.log", false}, // "cat" は1語目、"install.log" は "install" と完全一致しない
		{"tool test:unit", false},  // "test:unit" != "test"
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			if got := isVerifyByKeyword(tt.command); got != tt.want {
				t.Errorf("isVerifyByKeyword(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}
