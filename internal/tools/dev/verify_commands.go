package dev

import "strings"

// verifyPrefixes はビルド/テスト/lint等の検証系コマンドのプレフィックス一覧。
// IsVerifyCommand はこのリストとのプレフィックスマッチで判定する。
var verifyPrefixes = []string{
	// Go
	"go build", "go test", "go fmt", "go vet",
	"go run", "golangci-lint",

	// JavaScript / TypeScript
	"npm test", "npm run test", "npm run build", "npm run lint",
	"npm run check", "npm run typecheck", "npm run format",
	"npx jest", "npx vitest", "npx tsc", "npx eslint", "npx prettier",
	"npx mocha", "npx cypress run", "npx playwright test",
	"yarn test", "yarn build", "yarn lint", "yarn check", "yarn typecheck",
	"pnpm test", "pnpm build", "pnpm lint", "pnpm check",
	"bun test", "bun build", "bun run test", "bun run build",
	"deno test", "deno check", "deno lint", "deno fmt",
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
	"pip install", "uv run", "uv build",
	"poetry build", "poetry check",

	// Rust
	"cargo build", "cargo test", "cargo check", "cargo clippy",
	"cargo fmt", "cargo bench", "cargo doc",
	"cargo deny check", "cargo audit",
	"rustfmt --check",

	// Java / Kotlin
	"mvn compile", "mvn test", "mvn verify", "mvn package",
	"mvn checkstyle:check", "mvn spotbugs:check",
	"gradle build", "gradle test", "gradle check",
	"gradle compileJava", "gradle compileKotlin",
	"./gradlew build", "./gradlew test", "./gradlew check",
	"./mvnw compile", "./mvnw test", "./mvnw verify",
	"javac", "java -jar",
	"ktlint", "detekt",

	// Ruby
	"bundle exec rspec", "bundle exec rake test",
	"bundle exec rake spec", "bundle exec minitest",
	"rubocop", "rubocop --autocorrect",
	"ruby -c",
	"rails test", "rails test:system",
	"bundle install", "gem build",
	"steep check", "sorbet",

	// PHP
	"phpunit", "composer test", "composer check",
	"php artisan test", "php artisan dusk",
	"phpstan analyse", "phpstan analyze",
	"phpcs", "php-cs-fixer fix --dry-run",
	"pest", "psalm",
	"php -l",
	"composer install", "composer validate",

	// Swift / Objective-C
	"swift build", "swift test", "swift package resolve",
	"xcodebuild test", "xcodebuild build",
	"xcodebuild -scheme", "xcodebuild analyze",
	"swiftlint", "swiftformat --lint",

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

	// SQL linters
	"sqlfluff lint", "sqlfluff fix",
	"squawk",
	"pg_format --check",

	// Docker / CI
	"docker build", "docker compose build",
	"docker compose up --build",

	// Terraform / IaC
	"terraform validate", "terraform plan", "terraform fmt -check",
	"tofu validate", "tofu plan",
	"pulumi preview",

	// 汎用
	"make ci-check", "make ci", "make lint", "make format",
	"make verify", "make compile",
	"pre-commit run",
}

// IsVerifyCommand はビルド/テスト/lint等の検証系コマンドかを判定する。
// verify系の場合、bash成功時の出力を1行サマリーに圧縮する対象となる。
// && や || や ; でつながったコマンドチェーンは、全パートがverify系の場合のみtrueを返す。
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

// isSingleVerifyCommand は単一コマンドがverifyプレフィックスに一致するか判定する。
func isSingleVerifyCommand(command string) bool {
	for _, prefix := range verifyPrefixes {
		if command == prefix {
			return true
		}
		if strings.HasPrefix(command, prefix) && len(command) > len(prefix) {
			next := command[len(prefix)]
			if next == ' ' || next == '\t' {
				return true
			}
		}
	}
	return false
}
