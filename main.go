package main

import (
	"github.com/joho/godotenv"
	"github.com/susugadx/xelyon-cli/cmd"
)

func main() {
	// .envファイルを読み込み（存在しなくてもエラーにならない）
	_ = godotenv.Load()

	cmd.Execute()
}
