# XELYON CLI Makefile

.PHONY: build test fmt lint gen-config gen-docs gen-registry gen-help gen-all clean check ci-check ci-check-full e2e azure-smoke azure-doctor-smoke bedrock-smoke release-check ci-verify-deps ci-check-fmt ci-check-tidy ci-build ci-check-binary-size ci-lint ci-test ci-check-coverage release-test

CI_BINARY := xelyon
CI_COVERAGE_FILE := coverage.txt
CI_COVERAGE_THRESHOLD := 60
CI_TEST_CMD := go test -p=2 -v -race -coverprofile=$(CI_COVERAGE_FILE) -covermode=atomic -tags grammar_set_core ./...
RELEASE_TEST_CMD := go test -p=2 -v -race -tags grammar_set_core ./...

# ビルド
build:
	go build -tags grammar_set_core -o xelyon

# テスト
test:
	go test -tags grammar_set_core ./...

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
	go run scripts/gen-help.go

# コマンドドキュメントの骨格を追加（不足分のみ）
gen-commands-docs:
	go run scripts/gen-commands-docs.go

# 設定関連を全て自動生成（生成後に go fmt で整形）
gen-all: gen-config gen-docs gen-registry gen-help gen-commands-docs
	go fmt ./internal/config/registry_generated.go

# クリーン
clean:
	rm -f xelyon
	rm -f *.bak

# 全てのチェック
check: fmt test lint

# CI の依存関係整合性チェック
ci-verify-deps:
	@echo "=== Verifying dependencies ==="
	@go mod verify
	@echo "✓ Dependency verification passed"
	@echo ""

# CI の go fmt チェック
ci-check-fmt:
	@echo "=== Checking go fmt ==="
	@if [ -n "$$(go fmt ./...)" ]; then \
		echo "Error: Files need formatting. Run 'make fmt' to fix."; \
		exit 1; \
	fi
	@echo "✓ go fmt check passed"
	@echo ""

# CI の go mod tidy チェック
ci-check-tidy:
	@echo "=== Checking go mod tidy ==="
	@go mod tidy
	@if ! git diff --exit-code go.mod go.sum >/dev/null; then \
		echo "Error: go.mod/go.sum are not tidy. Run 'go mod tidy' and commit the changes."; \
		exit 1; \
	fi
	@echo "✓ go mod tidy check passed"
	@echo ""

# CI のビルドチェック
ci-build:
	@echo "=== Building binary ==="
	@go build -tags grammar_set_core -v -o $(CI_BINARY)
	@echo "✓ Build check passed"
	@echo ""

# CI のバイナリサイズ確認
ci-check-binary-size:
	@echo "=== Checking binary size ==="
	@size=$$(stat -c%s $(CI_BINARY)); \
	echo "Binary size: $$((size / 1024 / 1024))MB ($$size bytes)"; \
	if [ $$size -gt 52428800 ]; then \
		echo "Warning: Binary size exceeds 50MB!"; \
	fi
	@echo "✓ Binary size check passed"
	@echo ""

# CI の lint チェック
ci-lint:
	@echo "=== Running golangci-lint ==="
	@golangci-lint run
	@echo "✓ Lint check passed"
	@echo ""

# CI と同じテストコマンド
ci-test:
	@echo "=== Running tests ==="
	@$(CI_TEST_CMD)
	@echo "✓ Tests passed"
	@echo ""

# CI のカバレッジ閾値チェック
ci-check-coverage:
	@echo "=== Checking coverage threshold ==="
	@coverage=$$(go tool cover -func=$(CI_COVERAGE_FILE) | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}'); \
	if [ -z "$$coverage" ]; then \
		echo "Error: Failed to read total coverage from $(CI_COVERAGE_FILE)."; \
		exit 1; \
	fi; \
	echo "Total coverage: $$coverage%"; \
	awk "BEGIN { exit !($$coverage >= $(CI_COVERAGE_THRESHOLD)) }" >/dev/null 2>&1 || { \
		echo "Error: Coverage $$coverage% is below $(CI_COVERAGE_THRESHOLD)% threshold"; \
		exit 1; \
	}
	@echo "✓ Coverage check passed"
	@echo ""

# CI と同じ主要チェック（ローカル実行向け）
ci-check:
	@$(MAKE) --no-print-directory ci-check-fmt
	@$(MAKE) --no-print-directory ci-verify-deps
	@$(MAKE) --no-print-directory ci-check-tidy
	@$(MAKE) --no-print-directory ci-build
	@$(MAKE) --no-print-directory ci-check-binary-size
	@$(MAKE) --no-print-directory ci-lint
	@$(MAKE) --no-print-directory ci-test
	@$(MAKE) --no-print-directory ci-check-coverage
	@rm -f $(CI_BINARY) $(CI_COVERAGE_FILE)
	@echo "✅ All CI checks passed!"

# インテグレーションテスト含む全テスト
ci-check-full:
	@echo "=== Running all tests (including integration) ==="
	@go test -tags "grammar_set_core integration" -race -timeout 600s ./...
	@echo "✓ All tests passed (including integration)"

# リリース検証と同じテスト
release-test:
	@$(RELEASE_TEST_CMD)

# E2Eテスト（実際のLLM APIを使用、OPENAI_API_KEY必須）
e2e:
	XELYON_E2E=1 go test ./e2e/ -v -timeout 300s

# Azure OpenAI Responses API の実環境 smoke test
azure-smoke:
	XELYON_AZURE_SMOKE=1 go test ./internal/api/providers/azure -run 'TestAzure(Responses|Doctor)Smoke' -v -count=1 -timeout 300s

# Azure doctor 診断経路だけを実環境で確認
azure-doctor-smoke:
	XELYON_AZURE_SMOKE=1 go test ./internal/api/providers/azure -run TestAzureDoctorSmoke -v -count=1 -timeout 300s

# Bedrock 実 API smoke test（AWS 認証チェーン必須）
bedrock-smoke:
	XELYON_BEDROCK_SMOKE=1 go test ./internal/api/providers/bedrock -run TestBedrockLiveSmoke -count=1 -v -timeout 10m

# リリース前チェック
release-check: fmt test lint build
	@echo "All checks passed!"
