package agent

import "testing"

func TestIsCompletionTriggerResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
	}{
		{"japanese_kanryo", "すべての変更が完了しました。", true},
		{"japanese_shuhsei_shimashita", "ファイルを修正しました。", true},
		{"japanese_ijou_desu", "変更は以上です。", true},
		{"japanese_jisso_shimashita", "新機能を実装しました。", true},
		{"japanese_taiou_shimashita", "バグに対応しました。", true},
		{"japanese_shuhsei_kanryo", "修正完了です。", true},
		{"japanese_sagyo_ha_ijou", "作業は以上になります。", true},
		{"japanese_henkou_ha_ijou", "変更は以上となります。", true},
		{"english_done", "I'm done with the changes.", true},
		{"english_completed", "The task is completed.", true},
		{"english_requested_changes_done", "The requested changes are done.", true},
		{"english_changes_have_been_completed", "The changes have been completed.", true},
		{"english_finished", "I've finished implementing the feature.", true},
		{"english_all_done", "That's all done now.", true},
		{"english_all_set", "Everything is all set.", true},
		{"english_thats_it", "That's it for the changes.", true},
		{"english_task_complete", "The task complete.", true},
		{"english_changes_complete", "The changes are complete.", true},
		{"english_impl_complete", "The implementation is complete.", true},
		{"english_DONE", "DONE", true},
		{"english_Completed", "Completed successfully.", true},
		{"english_FINISHED", "FINISHED!", true},
		{"no_match_english_progress_then_next_step", "Completed updating foo.go. Next I'll update tests.", false},
		{"no_match_english_progress_with_remaining_work", "Finished fixing foo.go, but I still need to update the tests.", false},
		{"no_match_japanese_progress_then_next_step", "foo.go を修正しました。次にテストを更新します。", false},
		{"no_match_japanese_progress_with_remaining_work", "対応しましたが、残りのテスト追加があります。", false},
		{"no_match_question", "Would you like me to make more changes?", false},
		{"no_match_explanation", "Here's how the function works...", false},
		{"no_match_tool_discussion", "I need to read the file first.", false},
		{"no_match_continuing", "I'll continue working on the remaining changes.", false},
		{"no_match_not_done_yet", "I haven't done that yet.", false},
		{"no_match_changes_not_completed_yet", "The changes have not been completed yet.", false},
		{"no_match_needs_to_be_done", "The remaining step still needs to be done.", false},
		{"no_match_japanese_not_completed", "まだ完了していません。", false},
		{"empty_string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCompletionTriggerResponse(tt.response)
			if got != tt.want {
				t.Errorf("isCompletionTriggerResponse(%q) = %v, want %v", tt.response, got, tt.want)
			}
		})
	}
}
