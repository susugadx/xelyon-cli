package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// ToolCall はAIからのツール呼び出し
type ToolCall struct {
	ID               string           `json:"id,omitempty"` // Function Calling 用の tool_call_id
	Tool             string           `json:"tool"`
	RawArgs          map[string]any   `json:"args"`
	ThoughtSignature string           `json:"thought_signature,omitempty"` // Gemini 3: 暗号化された思考プロセス署名
	ThoughtParts     []map[string]any `json:"thought_parts,omitempty"`     // Gemini 3: thought パート情報
	Args             map[string]string
}

// NormalizeArgs はRawArgsをArgsに変換（数値→文字列）
// 末尾の制御文字・ゼロ幅文字をトリミングし、path引数のサニタイズも実施
func (tc *ToolCall) NormalizeArgs() {
	tc.Args = make(map[string]string)
	for k, v := range tc.RawArgs {
		switch val := v.(type) {
		case string:
			tc.Args[k] = sanitizeArgValue(k, val)
		case float64:
			// JSONの数値はfloat64としてパースされる
			if val == float64(int64(val)) {
				tc.Args[k] = fmt.Sprintf("%d", int64(val))
			} else {
				tc.Args[k] = fmt.Sprintf("%g", val)
			}
		case int64:
			tc.Args[k] = fmt.Sprintf("%d", val)
		case bool:
			tc.Args[k] = fmt.Sprintf("%t", val)
		default:
			// 配列・オブジェクト等の複合型は json.Marshal で正しい JSON 文字列に変換
			// fmt.Sprintf("%v", v) は Go 構文（map[k:v]）になるため NG
			if jsonBytes, err := json.Marshal(v); err == nil {
				tc.Args[k] = string(jsonBytes)
			} else {
				tc.Args[k] = fmt.Sprintf("%v", v)
			}
		}
	}
}

// sanitizeArgValue はツール引数の文字列値をサニタイズ
// - 末尾の制御文字・ゼロ幅文字をトリミング
// - path 引数の末尾に付いた不正なサフィックスを除去
func sanitizeArgValue(key, value string) string {
	// 末尾の制御文字・ゼロ幅文字をトリミング
	value = strings.TrimRightFunc(value, func(r rune) bool {
		return unicode.IsControl(r) || r == '\uFEFF' || r == '\u200B'
	})

	// path 引数: ファイルパスに不正なサフィックス（スペース+ランダムID）が付くパターンを除去
	// 例: "internal/agent/agent.go lodge-187541604" → "internal/agent/agent.go"
	if key == "path" {
		value = cleanPathArg(value)
	}

	return value
}

// cleanPathArg はファイルパス引数から不正なサフィックスを除去
// Geminiが出力するランダムID付きパスを修正
func cleanPathArg(path string) string {
	// パスにスペースが含まれる場合、スペース以降が不正なサフィックスの可能性
	// 正当なパスにスペースが含まれることもあるが、ツール引数のpathでは稀
	if idx := strings.IndexByte(path, ' '); idx != -1 {
		candidate := path[:idx]
		// スペース前がファイルパスとして妥当なら（拡張子があるか、/で終わるか）
		if looksLikeFilePath(candidate) {
			return candidate
		}
	}
	return path
}

// looksLikeFilePath は文字列がファイルパスらしいかを簡易判定
func looksLikeFilePath(s string) bool {
	// 拡張子を持つ（.go, .js, .py, etc.）
	if idx := strings.LastIndexByte(s, '.'); idx != -1 && idx < len(s)-1 {
		ext := s[idx:]
		if !strings.ContainsAny(ext, " \t\n") {
			return true
		}
	}
	// ディレクトリパス（/で終わる）
	if strings.HasSuffix(s, "/") {
		return true
	}
	return false
}

// FileChangeDetail は1件のツール実行に含まれる個別ファイル変更を表す。
type FileChangeDetail struct {
	FilePath     string
	Action       string
	LinesAdded   int
	LinesRemoved int
}

// FileChange はファイル変更履歴
type FileChange struct {
	FilePath     string
	Details      []FileChangeDetail
	Timestamp    time.Time
	Tool         string
	Description  string
	LinesAdded   int // 追加行数
	LinesRemoved int // 削除行数
}
