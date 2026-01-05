package file

import (
    "fmt"
    "os"
    "path/filepath"
)

func ReadFile(path string) (string, error) {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return "", err
    }

    content, err := os.ReadFile(absPath)
    if err != nil {
        return "", fmt.Errorf("ファイルを読めません: %s", err)
    }

    return string(content), nil
}

func ReadFiles(paths []string) (string, error) {
    var result string
    for _, path := range paths {
        content, err := ReadFile(path)
        if err != nil {
            return "", err
        }
        result += fmt.Sprintf("## ファイル: %s\n```\n%s\n```\n\n", path, content)
    }
    return result, nil
}
