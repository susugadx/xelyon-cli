package cmd

// legacy_test.go - runLegacyMode のテスト
//
// runLegacyMode は以下の理由から単体テストが困難:
// - os.Exit() を呼び出す
// - 標準出力に直接書き込む
// - 外部 API を呼び出す
// - ユーザー入力を要求する（ConfirmApply）
//
// 統合テストまたは E2E テストで検証することを推奨
