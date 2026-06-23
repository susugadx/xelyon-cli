# XELYON CLI Makefile

.PHONY: build test fmt lint gen-config gen-docs gen-registry gen-help gen-all clean check verify-fast ci-check ci-check-full e2e azure-smoke azure-doctor-smoke openai-doctor-smoke deepseek-doctor-smoke gemini-doctor-smoke claude-doctor-smoke groq-doctor-smoke openrouter-doctor-smoke kimi-doctor-smoke ollama-doctor-smoke doctor-smoke-matrix kimi-smoke kimi-tool-smoke kimi-image-smoke kimi-web-search-smoke bedrock-smoke bedrock-doctor-smoke bedrock-smoke-matrix bedrock-smoke-probe release-check ci-verify-deps ci-check-fmt ci-check-tidy ci-build ci-check-binary-size ci-lint ci-test ci-check-coverage release-test

CI_BINARY := xelyon
CI_COVERAGE_FILE := coverage.txt
CI_COVERAGE_THRESHOLD := 60
CI_TEST_CMD := go test -p=2 -v -race -coverprofile=$(CI_COVERAGE_FILE) -covermode=atomic -tags grammar_set_core ./...
RELEASE_TEST_CMD := go test -p=2 -v -race -tags grammar_set_core ./...
BEDROCK_SMOKE_CLAUDE_MODEL ?= global.anthropic.claude-sonnet-4-6
BEDROCK_SMOKE_CONVERSE_MODELS ?= amazon.nova-pro-v1:0 moonshotai.kimi-k2.5
BEDROCK_PROBE_CONVERSE_MODELS ?= us.meta.llama4-scout-17b-instruct-v1:0 us.deepseek.r1-v1:0 google.gemma-3-4b-it
OPENAI_DOCTOR_SMOKE_MODEL ?= gpt-5.4
DEEPSEEK_DOCTOR_SMOKE_MODEL ?= deepseek-v4-flash
GEMINI_DOCTOR_SMOKE_MODEL ?= gemini-3.1-pro-preview-customtools
GEMINI_DOCTOR_SMOKE_TIMEOUT ?= 180s
CLAUDE_DOCTOR_SMOKE_MODEL ?= claude-sonnet-4-6
CLAUDE_DOCTOR_SMOKE_TIMEOUT ?= 180s
GROQ_DOCTOR_SMOKE_MODEL ?= meta-llama/llama-4-scout-17b-16e-instruct
OPENROUTER_DOCTOR_SMOKE_MODEL ?= openai/gpt-5.4-mini
KIMI_DOCTOR_SMOKE_MODEL ?= kimi-k2.6
OLLAMA_DOCTOR_SMOKE_MODEL ?= qwen2.5-coder:7b
DOCTOR_SMOKE_PROVIDERS ?=

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

# 実装中の軽量共通チェック（テストは touched package ごとに別途実行）
verify-fast:
	@$(MAKE) --no-print-directory ci-check-fmt
	@$(MAKE) --no-print-directory ci-check-tidy
	@$(MAKE) --no-print-directory ci-build
	@$(MAKE) --no-print-directory ci-lint
	@rm -f $(CI_BINARY)
	@echo "✅ Fast verification checks passed!"

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

# OpenAI doctor 診断経路を実環境で確認（OPENAI_API_KEY 必須）
openai-doctor-smoke:
	@test -n "$(OPENAI_API_KEY)" || { echo "OPENAI_API_KEY is required for make openai-doctor-smoke"; exit 1; }
	go run . doctor openai --model "$(OPENAI_DOCTOR_SMOKE_MODEL)" --smoke --tool-smoke --retention-smoke

# DeepSeek doctor 診断経路を実環境で確認（DEEPSEEK_API_KEY 必須）
deepseek-doctor-smoke:
	@test -n "$(DEEPSEEK_API_KEY)" || { echo "DEEPSEEK_API_KEY is required for make deepseek-doctor-smoke"; exit 1; }
	go run . doctor deepseek --model "$(DEEPSEEK_DOCTOR_SMOKE_MODEL)" --smoke --tool-smoke

# Gemini doctor 診断経路を実環境で確認（GEMINI_API_KEY 必須）
gemini-doctor-smoke:
	@test -n "$(GEMINI_API_KEY)" || { echo "GEMINI_API_KEY is required for make gemini-doctor-smoke"; exit 1; }
	go run . doctor gemini --model "$(GEMINI_DOCTOR_SMOKE_MODEL)" --timeout "$(GEMINI_DOCTOR_SMOKE_TIMEOUT)" --smoke --tool-smoke --image-smoke --web-search-smoke

# Claude doctor 診断経路を実環境で確認（ANTHROPIC_API_KEY 必須）
claude-doctor-smoke:
	@test -n "$(ANTHROPIC_API_KEY)" || { echo "ANTHROPIC_API_KEY is required for make claude-doctor-smoke"; exit 1; }
	go run . doctor claude --model "$(CLAUDE_DOCTOR_SMOKE_MODEL)" --timeout "$(CLAUDE_DOCTOR_SMOKE_TIMEOUT)" --smoke --tool-smoke --image-smoke --thinking-smoke --web-search-smoke

# Groq doctor 診断経路を実環境で確認（GROQ_API_KEY 必須）
groq-doctor-smoke:
	@test -n "$(GROQ_API_KEY)" || { echo "GROQ_API_KEY is required for make groq-doctor-smoke"; exit 1; }
	go run . doctor groq --model "$(GROQ_DOCTOR_SMOKE_MODEL)" --smoke --tool-smoke

# OpenRouter doctor 診断経路を実環境で確認（OPENROUTER_API_KEY 必須）
openrouter-doctor-smoke:
	@test -n "$(OPENROUTER_API_KEY)" || { echo "OPENROUTER_API_KEY is required for make openrouter-doctor-smoke"; exit 1; }
	go run . doctor openrouter --model "$(OPENROUTER_DOCTOR_SMOKE_MODEL)" --smoke --tool-smoke

# Kimi doctor 診断経路を実環境で確認（MOONSHOT_API_KEY 必須）
kimi-doctor-smoke:
	@test -n "$(MOONSHOT_API_KEY)" || { echo "MOONSHOT_API_KEY is required for make kimi-doctor-smoke"; exit 1; }
	go run . doctor kimi --model "$(KIMI_DOCTOR_SMOKE_MODEL)" --catalog-model "$(KIMI_DOCTOR_SMOKE_MODEL)" --smoke --tool-smoke --image-smoke --web-search-smoke

# Ollama doctor 診断経路をローカル Ollama で確認（ollama serve と pull 済みモデルが必要）
ollama-doctor-smoke:
	go run . doctor ollama --model "$(OLLAMA_DOCTOR_SMOKE_MODEL)" --catalog-model "$(OLLAMA_DOCTOR_SMOKE_MODEL)" --smoke --tool-smoke

# 任意 provider の doctor smoke を横断実行（DOCTOR_SMOKE_PROVIDERS で明示 opt-in）
doctor-smoke-matrix:
	@providers="$(DOCTOR_SMOKE_PROVIDERS)"; \
	if [ -z "$$providers" ]; then \
		echo "Set DOCTOR_SMOKE_PROVIDERS=\"openai deepseek gemini claude groq openrouter kimi ollama bedrock azure\" to opt in."; \
		exit 0; \
	fi; \
	matrix_status=0; \
	for provider in $$providers; do \
		case "$$provider" in \
			azure) target="azure-doctor-smoke" ;; \
			openai) target="openai-doctor-smoke" ;; \
			deepseek) target="deepseek-doctor-smoke" ;; \
			gemini) target="gemini-doctor-smoke" ;; \
			claude) target="claude-doctor-smoke" ;; \
			groq) target="groq-doctor-smoke" ;; \
			openrouter) target="openrouter-doctor-smoke" ;; \
			kimi) target="kimi-doctor-smoke" ;; \
			ollama) target="ollama-doctor-smoke" ;; \
			bedrock) target="bedrock-doctor-smoke" ;; \
			*) echo "Unknown DOCTOR_SMOKE_PROVIDERS entry: $$provider"; exit 2 ;; \
		esac; \
		echo "=== doctor smoke: $$provider ($$target) ==="; \
		if ! $(MAKE) --no-print-directory "$$target"; then \
			matrix_status=1; \
		fi; \
	done; \
	exit $$matrix_status

# Kimi native provider の実 API smoke test（MOONSHOT_API_KEY 必須）
kimi-smoke:
	@test -n "$(MOONSHOT_API_KEY)" || { echo "MOONSHOT_API_KEY is required for make kimi-smoke"; exit 1; }
	go test -tags live ./internal/api/providers/kimi -run KimiLive -v -count=1 -timeout 300s

# Kimi tool calling を含めた実 API smoke test（MOONSHOT_API_KEY 必須）
kimi-tool-smoke:
	@test -n "$(MOONSHOT_API_KEY)" || { echo "MOONSHOT_API_KEY is required for make kimi-tool-smoke"; exit 1; }
	XELYON_KIMI_TOOL_SMOKE=1 go test -tags live ./internal/api/providers/kimi -run KimiLive -v -count=1 -timeout 300s

# Kimi image input の実 API smoke test（MOONSHOT_API_KEY 必須）
kimi-image-smoke:
	@test -n "$(MOONSHOT_API_KEY)" || { echo "MOONSHOT_API_KEY is required for make kimi-image-smoke"; exit 1; }
	go run . doctor kimi --image-smoke

# Kimi built-in web_search の実 API smoke test（MOONSHOT_API_KEY 必須）
kimi-web-search-smoke:
	@test -n "$(MOONSHOT_API_KEY)" || { echo "MOONSHOT_API_KEY is required for make kimi-web-search-smoke"; exit 1; }
	go run . doctor kimi --web-search-smoke

# Bedrock 実 API smoke test（AWS 認証チェーン必須）
bedrock-smoke:
	XELYON_BEDROCK_SMOKE=1 go test ./internal/api/providers/bedrock -run TestBedrockLiveSmoke -count=1 -v -timeout 10m

# Bedrock doctor 診断経路を実環境で確認
bedrock-doctor-smoke:
	go run . doctor bedrock --model "$(BEDROCK_SMOKE_CLAUDE_MODEL)" --smoke --tool-smoke --image-smoke --thinking-smoke

# Bedrock runtime supported モデルの実 API smoke matrix
bedrock-smoke-matrix:
	XELYON_BEDROCK_SMOKE=1 XELYON_BEDROCK_SMOKE_CLAUDE_MODEL="$(BEDROCK_SMOKE_CLAUDE_MODEL)" go test ./internal/api/providers/bedrock -run TestBedrockLiveSmoke_ClaudeMessagesRoute -count=1 -v -timeout 10m
	@for model in $(BEDROCK_SMOKE_CONVERSE_MODELS); do \
		echo "=== Bedrock Converse smoke: $$model ==="; \
		XELYON_BEDROCK_SMOKE=1 XELYON_BEDROCK_SMOKE_CONVERSE_MODEL="$$model" go test ./internal/api/providers/bedrock -run TestBedrockLiveSmoke_ConverseRoute -count=1 -v -timeout 10m || exit $$?; \
	done

# Bedrock streaming tool-use unsupported/unverified モデルを検証
bedrock-smoke-probe:
	XELYON_BEDROCK_SMOKE=1 XELYON_BEDROCK_PROBE_CONVERSE_MODELS="$(BEDROCK_PROBE_CONVERSE_MODELS)" go test ./internal/api/providers/bedrock -run TestBedrockLiveProbe_UnsupportedConverseModels -count=1 -v -timeout 10m

# リリース前チェック
release-check: fmt test lint build
	@echo "All checks passed!"
