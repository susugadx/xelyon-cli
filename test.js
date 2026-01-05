try {
    console.log("hello");
} catch (error) {
    console.error("エラーが発生しました:", error.message);
    // 必要に応じて追加のエラーハンドリング
    // 例: エラーログの送信、ユーザーへの通知など
}