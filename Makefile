# XELYON CLI Makefile

.PHONY: build test fmt lint gen-config gen-docs gen-registry gen-all clean

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

# 設定関連を全て自動生成
gen-all: gen-config gen-docs gen-registry

# クリーン
clean:
	rm -f xelyon
	rm -f *.bak

# 全てのチェック
check: fmt test lint

# リリース前チェック
release-check: fmt test lint build
	@echo "All checks passed!"
