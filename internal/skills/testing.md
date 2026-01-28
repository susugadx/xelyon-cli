# テスト実行

## Go
```bash
# 全テスト実行
go test ./...

# 詳細出力
go test -v ./...

# 特定パッケージ
go test ./internal/agent/...

# カバレッジ
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 特定のテストだけ
go test -run TestFunctionName ./...
```

## JavaScript / TypeScript
```bash
# npm
npm test
npm run test:coverage

# Jest
npx jest
npx jest --coverage
npx jest --watch

# Vitest
npx vitest
npx vitest --coverage
```

## Python
```bash
# pytest
pytest
pytest -v
pytest --cov=src

# unittest
python -m unittest discover
```

## Rust
```bash
cargo test
cargo test --verbose
```
## Java
```bash
# Maven
mvn test
mvn test -Dtest=TestClassName

# Gradle
./gradlew test
```

## Ruby
```bash
# RSpec
bundle exec rspec
bundle exec rspec --format documentation

# Minitest
ruby -Itest test/test_*.rb
```

## PHP
```bash
# PHPUnit
./vendor/bin/phpunit
./vendor/bin/phpunit --coverage-html coverage
```

## C# / .NET
```bash
dotnet test
dotnet test --collect:"XPlat Code Coverage"
```

## Swift
```bash
swift test
swift test --verbose
```

## Elixir
```bash
mix test
mix test --cover
```