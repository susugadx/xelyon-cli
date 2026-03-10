package dev

import "strings"

// nonVerifyPrefixes は出力を読む用途のコマンド。verify判定から明示的に除外する。
// verifyプレフィックス・キーワードより優先される。
var nonVerifyPrefixes = []string{
	// シェル基本
	"echo", "cat", "head", "tail", "less", "more",
	"grep", "rg", "awk", "sed", "sort", "uniq", "cut", "tr",
	"wc", "du", "df",

	// ファイル操作（見る/移動系）
	"ls", "find", "tree",
	"mv", "cp", "rm", "mkdir", "chmod", "chown", "ln",
	"tar", "zip", "unzip", "gzip",

	// ネットワーク/API
	"curl", "wget", "ssh", "scp", "rsync",
	"nc", "nslookup", "dig", "ping", "traceroute", "whois",

	// git（出力を読む系 — Phase 2で別途閾値圧縮済み）
	"git diff", "git log", "git show", "git blame",
	"git status", "git stash",

	// エディタ
	"vi", "vim", "nano", "code",

	// プロセス
	"ps", "top", "kill", "htop",

	// 環境/設定
	"cd", "pwd", "env", "export", "set", "source",
	"which", "whereis", "type",
	"printenv",

	// DB クライアント（クエリ結果を読む用途）
	"psql", "mysql", "sqlite3", "mongosh", "redis-cli",
	"pgcli", "mycli", "litecli",

	// Kubernetes（一覧/詳細を読む用途）
	"kubectl get", "kubectl describe",
	"kubectl exec", "kubectl port-forward",

	// Docker（状態を見る用途）
	"docker inspect", "docker ps", "docker images",
	"docker exec",

	// AWS/GCP/Azure（情報取得系）
	"aws describe", "aws get", "aws list",
	"aws s3 ls",
	"gcloud compute instances list",
	"gcloud describe",
	"az vm list", "az show",

	// Terraform（状態を見る用途）
	"terraform state", "terraform output", "terraform show",

	// プロファイラ/ベンチマーク（結果を見る用途）
	"hyperfine", "perf", "strace", "ltrace",

	// 暗号系（出力を読む用途）
	"openssl",

	// man / help
	"man", "info",
}

// verifyPrefixes はビルド/テスト/lint等の検証系コマンドのプレフィックス一覧。
// 確実にverify対象と判定できるもの。
var verifyPrefixes = []string{
	// Go
	"go build", "go test", "go fmt", "go vet",
	"go run", "go generate", "go mod download", "go mod tidy", "go get",
	"golangci-lint", "gofumpt",

	// JavaScript / TypeScript
	"npm test", "npm run test", "npm run build", "npm run lint",
	"npm run check", "npm run typecheck", "npm run format",
	"npm install", "npm ci", "npm i", "npm publish",
	"npx jest", "npx vitest", "npx tsc", "npx eslint", "npx prettier",
	"npx mocha", "npx cypress run", "npx playwright test",
	"yarn test", "yarn build", "yarn lint", "yarn check", "yarn typecheck",
	"yarn install", "yarn add",
	"pnpm test", "pnpm build", "pnpm lint", "pnpm check",
	"pnpm install", "pnpm add",
	"bun test", "bun build", "bun run test", "bun run build",
	"bun install", "bun add",
	"deno test", "deno check", "deno lint", "deno fmt", "deno install",
	"node --check",
	"tsc --noEmit", "tsc -b",
	"eslint", "prettier --check", "biome check", "biome lint",

	// Python
	"pytest", "python -m pytest", "python -m unittest",
	"python -m doctest", "python -m py_compile",
	"ruff check", "ruff format --check",
	"mypy", "pyright", "pytype",
	"flake8", "pylint", "pydocstyle",
	"black --check", "autopep8 --diff",
	"isort --check", "isort --diff",
	"bandit", "safety check",
	"tox", "nox",
	"python setup.py test", "python -m build",
	"pip install", "pip3 install", "uv run", "uv build",
	"uv pip install", "uv sync",
	"pipenv install", "pipenv sync",
	"conda install", "conda env create",
	"poetry build", "poetry check",
	"twine upload",

	// Rust
	"cargo build", "cargo test", "cargo check", "cargo clippy",
	"cargo fmt", "cargo bench", "cargo doc",
	"cargo deny check", "cargo audit",
	"cargo fetch", "cargo install", "cargo publish",
	"rustfmt --check",

	// Java / Kotlin
	"mvn compile", "mvn test", "mvn verify", "mvn package",
	"mvn checkstyle:check", "mvn spotbugs:check",
	"gradle build", "gradle test", "gradle check",
	"gradle compileJava", "gradle compileKotlin",
	"gradle assemble",
	"./gradlew build", "./gradlew test", "./gradlew check",
	"./gradlew assemble",
	"./mvnw compile", "./mvnw test", "./mvnw verify",
	"javac", "java -jar",
	"ktlint", "detekt",

	// Ruby
	"bundle exec rspec", "bundle exec rake test",
	"bundle exec rake spec", "bundle exec minitest",
	"rubocop", "rubocop --autocorrect",
	"ruby -c",
	"rails test", "rails test:system",
	"rails generate", "rails g",
	"bundle install", "gem build", "gem install", "gem push",

	// PHP
	"phpunit", "composer test", "composer check",
	"composer install", "composer update", "composer validate",
	"php artisan test", "php artisan dusk",
	"phpstan analyse", "phpstan analyze",
	"phpcs", "php-cs-fixer fix --dry-run",
	"pest", "psalm",
	"php -l",

	// Swift / Objective-C
	"swift build", "swift test", "swift package resolve",
	"xcodebuild test", "xcodebuild build",
	"xcodebuild -scheme", "xcodebuild analyze", "xcodebuild archive",
	"swiftlint", "swiftformat --lint",
	"pod install", "pod update",

	// C / C++
	"make", "make test", "make check", "make all",
	"cmake --build", "cmake --install",
	"ctest", "ctest --test-dir",
	"gcc", "g++", "clang", "clang++",
	"clang-tidy", "clang-format --dry-run",
	"cppcheck", "cpplint",
	"meson compile", "meson test",
	"ninja", "ninja test",
	"bazel build", "bazel test",

	// .NET / C# / F#
	"dotnet build", "dotnet test", "dotnet run",
	"dotnet publish", "dotnet restore",
	"dotnet format --check",
	"nuget restore",

	// Dart / Flutter
	"dart analyze", "dart test", "dart compile",
	"dart format", "dart fix --apply",
	"flutter test", "flutter build", "flutter analyze",
	"flutter pub get",

	// Elixir
	"mix compile", "mix test", "mix format --check",
	"mix credo", "mix dialyzer",
	"mix deps.get",

	// Haskell
	"cabal build", "cabal test",
	"stack build", "stack test",
	"hlint",

	// Scala
	"sbt compile", "sbt test", "sbt package",
	"scalafmt --check", "scalafix",

	// Zig
	"zig build", "zig test",
	"zig fmt --check",

	// Nim
	"nim compile", "nim c", "nim check",
	"nimble build", "nimble test",

	// Perl
	"prove", "perl -c",

	// Lua
	"luacheck", "busted",

	// R
	"Rscript -e", "R CMD check", "R CMD build",

	// Clojure
	"lein test", "lein check", "lein compile",
	"clj -M:test",

	// OCaml
	"dune build", "dune test", "dune runtest",
	"opam build",

	// Julia
	"julia --project -e",

	// Erlang
	"rebar3 compile", "rebar3 eunit", "rebar3 ct",
	"rebar3 dialyzer",

	// Shell
	"shellcheck", "shfmt -d",
	"bats",

	// Database migrations / validation
	"prisma migrate", "prisma db push", "prisma validate",
	"prisma generate", "prisma format",
	"drizzle-kit push", "drizzle-kit generate",
	"typeorm migration:run", "typeorm schema:sync",
	"sequelize db:migrate",
	"knex migrate:latest",
	"diesel migration run", "diesel migration redo",
	"alembic upgrade", "alembic check",
	"flask db upgrade", "flask db migrate",
	"rails db:migrate", "rails db:schema:load",
	"rake db:migrate", "rake db:test:prepare",
	"goose up", "goose status",
	"migrate -database", "golang-migrate",
	"flyway migrate", "flyway validate", "flyway info",
	"liquibase update", "liquibase validate",
	"dbmate up", "dbmate migrate",
	"sqitch deploy", "sqitch verify",
	"atlas migrate apply", "atlas schema apply",

	// DB管理
	"createdb", "dropdb",
	"pg_dump", "pg_restore", "pg_basebackup",
	"mysqldump", "mysqlimport",
	"mongodump", "mongorestore", "mongoimport", "mongoexport",
	"pgloader",

	// SQL linters
	"sqlfluff lint", "sqlfluff fix",
	"squawk",
	"pg_format --check",

	// Docker / CI
	"docker build", "docker compose build",
	"docker compose up", "docker compose down", "docker compose restart",
	"docker push", "docker tag", "docker pull", "docker compose pull",

	// Terraform / IaC
	"terraform validate", "terraform plan", "terraform fmt -check",
	"terraform apply", "terraform destroy", "terraform init",
	"tofu validate", "tofu plan",
	"pulumi preview",

	// Ansible
	"ansible-playbook", "ansible-pull",

	// Kubernetes
	"kubectl apply", "kubectl delete",
	"kubectl rollout", "kubectl scale",
	"kubectl patch", "kubectl create", "kubectl set",
	"helm install", "helm upgrade", "helm uninstall",
	"kustomize build",

	// AWS
	"aws s3 cp", "aws s3 mv", "aws s3 rm",
	"aws deploy",
	"aws cloudformation deploy", "aws cloudformation create",
	"aws cloudformation update", "aws cloudformation delete",
	"aws lambda update",
	"aws ecr get-login",
	"aws ecs update-service",
	"aws eks update-kubeconfig",
	"sam build", "sam deploy",
	"cdk deploy", "cdk synth",

	// GCP
	"gcloud app deploy", "gcloud run deploy",
	"gcloud builds submit", "gcloud functions deploy",
	"gcloud compute instances create",
	"gcloud container clusters create",

	// Azure
	"az webapp deploy", "az acr build",
	"az deployment group create",
	"az aks create", "az functionapp deploy",

	// セキュリティスキャン
	"trivy", "grype", "semgrep",
	"gitleaks", "trufflehog",
	"osv-scanner", "govulncheck",
	"bearer scan",

	// モバイル
	"react-native run",
	"fastlane",

	// ドキュメント生成
	"sphinx-build", "typedoc",

	// Protocol Buffers
	"protoc", "buf generate",

	// 証明書
	"certbot certonly", "certbot renew",

	// Git操作
	"git rebase", "git cherry-pick",

	// GitHub CLI
	"gh workflow run",
	"gh pr create", "gh pr merge",
	"gh release create",

	// Linter追加
	"hadolint", "yamllint", "jsonlint",
	"markdownlint", "mdl",
	"editorconfig-checker",
	"actionlint", "commitlint",

	// パッケージマネージャ（システム）
	"apt-get install", "apt-get update",
	"apt install", "apt update",
	"brew install", "brew update", "brew upgrade",
	"pacman -S", "pacman -Syu",
	"dnf install", "yum install",
	"apk add",
	"mc cp", "mc mirror",

	// パッケージ公開
	"goreleaser",

	// 汎用
	"make ci-check", "make ci", "make lint", "make format",
	"make verify", "make compile",
	"pre-commit run",
}

// verifyKeywords はverify系コマンドのサブコマンドに共通するキーワード。
// コマンドの先頭2語のいずれかがこのリストに一致すればverify判定する。
// nonVerifyPrefixes → verifyPrefixes の後にフォールバックとして適用。
var verifyKeywords = []string{
	// ビルド/テスト/lint
	"install", "build", "test", "lint", "check",
	"compile", "format", "fmt", "vet", "audit",
	"validate", "verify",

	// マイグレーション/データ管理
	"migrate", "restore", "dump", "backup",
	"seed", "reset", "import", "export", "load",

	// パッケージ管理
	"update", "upgrade", "fetch", "download",
	"publish", "release",

	// デプロイ/インフラ操作
	"deploy", "apply", "create", "delete",
	"push", "pull", "rollout", "restart",
	"scale", "patch", "sync", "init",
	"provision", "destroy", "launch",
	"up", "down", "start", "stop",
	"login", "logout",

	// コード生成/変換
	"generate", "clean", "archive", "assemble",
	"package", "scan", "detect",

	// Git操作
	"merge", "tag", "rebase", "cherry-pick",
}

// IsVerifyCommand はビルド/テスト/lint等の検証系コマンドかを判定する。
// verify系の場合、bash成功時の出力を1行サマリーに圧縮する対象となる。
// && や || や ; でつながったコマンドチェーンは、全パートがverify系の場合のみtrueを返す。
//
// 3層判定:
//  1. 非verifyプレフィックス → false（出力を読む用途を明示除外）
//  2. verifyプレフィックス → true（確実なもの）
//  3. verifyキーワード → true（先頭2語にキーワードが含まれるかのフォールバック）
func IsVerifyCommand(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}

	parts := splitChainCommand(command)
	for _, part := range parts {
		if !isSingleVerifyCommand(strings.TrimSpace(part)) {
			return false
		}
	}
	return true
}

// isSingleVerifyCommand は単一コマンドを3層判定する。
func isSingleVerifyCommand(command string) bool {
	// 1. 非verifyリスト → false（出力を読む用途を明示除外）
	if matchesNonVerifyPrefix(command) {
		return false
	}
	// 2. verifyプレフィックス → true（確実なもの）
	if matchesVerifyPrefix(command) {
		return true
	}
	// 3. verifyキーワード → true（フォールバック）
	return isVerifyByKeyword(command)
}

// matchesNonVerifyPrefix はコマンドが非verifyプレフィックスに一致するか判定する。
func matchesNonVerifyPrefix(command string) bool {
	for _, prefix := range nonVerifyPrefixes {
		if matchCommandWord(command, prefix) {
			return true
		}
	}
	return false
}

// matchesVerifyPrefix はコマンドがverifyプレフィックスに一致するか判定する。
func matchesVerifyPrefix(command string) bool {
	for _, prefix := range verifyPrefixes {
		if matchCommandWord(command, prefix) {
			return true
		}
	}
	return false
}

// matchCommandWord はコマンドがプレフィックスに完全一致、またはプレフィックス+空白で始まるか判定する。
func matchCommandWord(command, prefix string) bool {
	if command == prefix {
		return true
	}
	if strings.HasPrefix(command, prefix) && len(command) > len(prefix) {
		next := command[len(prefix)]
		return next == ' ' || next == '\t'
	}
	return false
}

// isVerifyByKeyword はコマンドの先頭2語にverifyキーワードが含まれるか判定する。
// 先頭2語に限定することで、引数部分の偶然の一致を防ぐ。
func isVerifyByKeyword(command string) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}
	checkParts := parts
	if len(checkParts) > 2 {
		checkParts = checkParts[:2]
	}
	for _, part := range checkParts {
		lower := strings.ToLower(part)
		for _, kw := range verifyKeywords {
			if lower == kw {
				return true
			}
		}
	}
	return false
}
