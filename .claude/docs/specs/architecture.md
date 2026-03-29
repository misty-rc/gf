# アーキテクチャ

```
main.go                  フラグ定義・引数前処理・出力
console_windows.go       SetConsoleOutputCP(65001): Windows UTF-8 対応
search/
  search.go              Options 定義・Run()
  walk.go                parallelWalker: 並行走査・フィルタ適用
  pattern.go             compilePattern(): glob fast path + regex
  birthtime_windows.go   作成日時取得（Win32FileAttributeData）
  birthtime_darwin.go    作成日時取得（Birthtimespec）
  birthtime_other.go     作成日時取得（Linux 非サポート、ゼロ値）
```
