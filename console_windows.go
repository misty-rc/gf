package main

import "syscall"

func init() {
	// Windows のコンソール出力コードページを UTF-8 (65001) に設定する。
	// デフォルトの CP932 (Shift-JIS) 環境では日本語ファイル名が文字化けするため。
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	kernel32.NewProc("SetConsoleOutputCP").Call(65001)
}
