package configgen

import (
	"errors"
	"fmt"
	"os"
)

// OutputPathFromArgs は script の引数から出力先を解決する。
func OutputPathFromArgs(args []string, defaultPath string) string {
	if len(args) > 2 && args[1] == "--" && args[2] != "" {
		return args[2]
	}
	if len(args) > 1 && args[1] != "" {
		return args[1]
	}
	return defaultPath
}

// ReadFileIfExists はファイルが存在するときだけ読み込み、存在有無を返す。
func ReadFileIfExists(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return nil, false, err
}

// ExitWithError は標準エラーへ出力して非0終了する。
func ExitWithError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
