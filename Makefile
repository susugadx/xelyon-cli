# XELYON CLI Makefile

.PHONY: build test fmt lint gen-config gen-docs gen-registry gen-help gen-all clean check ci-check ci-check-full release-check

# ビルド
build:
	go build -o xelyon

# テスト
test:
	go test ./...

# フォーマット
fmt:
	go fmt ./...

# Lint
lint:
	golangci-lint run

# 設定例を自動生成
gen-config:
	go run scripts/config_sections.go scripts/gen-config-example.go

# 設定ドキュメントを自動生成
gen-docs:
	go run scripts/config_sections.go scripts/gen-config-docs.go

# 設定レジストリを自動生成
gen-registry:
	go run scripts/config_sections.go scripts/gen-config-registry.go

# ヘルプテキストを自動生成
gen-help:
	go run scripts/commands.go scripts/gen-help.go

# コマンドドキュメントの骨格を追加（不足分のみ）
gen-commands-docs:
	go run scripts/commands.go scripts/gen-commands-docs.go

# 設定関連を全て自動生成
gen-all: gen-config gen-docs gen-registry gen-help gen-commands-docs

# クリーン
clean:
	rm -f xelyon
	rm -f *.bak

# 全てのチェック
check: fmt test lint

# CIと同じチェック（未フォーマットがあればエラー）
ci-check:
	@echo "=== Checking go fmt ==="
	@if [ -n "$$(go fmt ./...)" ]; then \
		echo "Error: Files need formatting. Run 'make fmt' to fix."; \
		exit 1; \
	fi
	@echo "✓ go fmt check passed"
	@echo ""
	@echo "=== Building all packages ==="
	@go build ./...
	@echo "✓ Build check passed"
	@echo ""
	@echo "=== Running golangci-lint ==="
	@golangci-lint run
	@echo "✓ Lint check passed"
	@echo ""
	@echo "=== Running tests ==="
	@go test -timeout 120s ./...
	@echo "✓ Tests passed"
	@echo ""
	@echo "✅ All CI checks passed!"

# インテグレーションテスト含む全テスト
ci-check-full:
	@echo "=== Running all tests (including integration) ==="
	@go test -tags integration -race -timeout 600s ./...
	@echo "✓ All tests passed (including integration)"

# リリース前チェック
release-check: fmt test lint build
	@echo "All checks passed!"
