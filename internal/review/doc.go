// Package review は /review 実行の domain contract と pure policy を提供する。
//
// ReviewRunner はこの package 内の orchestration 層として、evidence
// build/render、probe plan decode/convert、probe execution、report
// decode/validate を呼び出す。TUI、provider、agent 固有の詳細はそれぞれの
// owner に残し、ここでは review domain の入力/出力境界を固定する。
package review
