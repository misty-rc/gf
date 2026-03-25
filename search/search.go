package search

import (
	"path/filepath"
)

// Options は検索条件を表す。
type Options struct {
	Pattern   string // グロブまたは正規表現パターン（空文字は全マッチ）
	Regex     bool   // true のとき Pattern を正規表現として扱う
	MatchPath bool   // true のとき Pattern をフルパスに適用する（デフォルト: ファイル名のみ）
	Type      string // "f"=ファイルのみ, "d"=ディレクトリのみ, ""=両方
	Ext       string // 拡張子フィルタ（ドットなし。例: "go"）
	Hidden    bool   // 隠しファイル・ディレクトリを含める
	MaxDepth  int    // 最大探索深さ（0=無制限）
}

// Run は root 以下を並行走査し、マッチしたパスを results チャネルに送信する。
// 呼び出し側は Run 完了後に results を close すること。
// エラーは正規表現コンパイル失敗時のみ返す。ReadDir 失敗はサイレントスキップする。
func Run(root string, opts Options, results chan<- string) error {
	patternFn, err := compilePattern(opts)
	if err != nil {
		return err
	}

	w := newWalker(opts, patternFn, results)
	w.wg.Add(1)
	go w.walk(filepath.ToSlash(root), 0)
	w.wg.Wait()
	return nil
}
