package search

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// Options は検索条件を表す。
type Options struct {
	Pattern  string // グロブまたは正規表現パターン（空文字は全マッチ）
	Regex    bool   // true のとき Pattern を正規表現として扱う
	Type     string // "f"=ファイルのみ, "d"=ディレクトリのみ, ""=両方
	Ext      string // 拡張子フィルタ（ドットなし。例: "go"）
	Hidden   bool   // 隠しファイル・ディレクトリを含める
	MaxDepth int    // 最大探索深さ（0=無制限）
}

type parallelWalker struct {
	opts    Options
	glob    string // 正規化済みグロブ（部分一致変換済み）
	re      *regexp.Regexp
	results chan<- string
	wg      sync.WaitGroup
	sem     chan struct{} // goroutine 数の上限
}

// Run は root 以下を並行走査し、マッチしたパスを results チャネルに送信する。
// ディレクトリごとに goroutine を起動して os.ReadDir を並列実行する。
func Run(root string, opts Options, results chan<- string) error {
	var re *regexp.Regexp
	if opts.Regex && opts.Pattern != "" {
		var err error
		re, err = regexp.Compile(opts.Pattern)
		if err != nil {
			return err
		}
	}

	// グロブの事前正規化（ループ内で毎回変換しないよう）
	glob := opts.Pattern
	if !opts.Regex && glob != "" && !hasGlobChars(glob) {
		glob = "*" + glob + "*"
	}

	// root をスラッシュ統一しておく（以後の子パスはすべてスラッシュで構築）
	slashRoot := filepath.ToSlash(root)

	w := &parallelWalker{
		opts:    opts,
		glob:    glob,
		re:      re,
		results: results,
		sem:     make(chan struct{}, runtime.NumCPU()),
	}

	w.wg.Add(1)
	go w.walk(slashRoot, 0)
	w.wg.Wait()
	return nil
}

// walk は dirPath（スラッシュ区切り）以下を再帰的に処理する。
func (w *parallelWalker) walk(dirPath string, depth int) {
	defer w.wg.Done()

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return
	}

	for _, e := range entries {
		name := e.Name()

		// 隠しファイル・ディレクトリの除外
		if !w.opts.Hidden && isHidden(name) {
			continue
		}

		// スラッシュで子パスを構築（filepath.Join + ToSlash の代わり）
		childPath := dirPath + "/" + name
		childDepth := depth + 1

		if e.IsDir() {
			if w.opts.MaxDepth > 0 && childDepth > w.opts.MaxDepth {
				continue
			}
			// ディレクトリ自体もマッチ対象
			if w.matches(e, name) {
				w.results <- childPath
			}
			w.wg.Add(1)
			// セマフォに空きがあれば goroutine、なければインライン実行
			select {
			case w.sem <- struct{}{}:
				go func(p string, d int) {
					defer func() { <-w.sem }()
					w.walk(p, d)
				}(childPath, childDepth)
			default:
				w.walk(childPath, childDepth)
			}
		} else {
			if w.opts.MaxDepth > 0 && childDepth > w.opts.MaxDepth {
				continue
			}
			if w.matches(e, name) {
				w.results <- childPath
			}
		}
	}
}

func (w *parallelWalker) matches(e os.DirEntry, name string) bool {
	// タイプフィルタ
	switch w.opts.Type {
	case "f":
		if e.IsDir() {
			return false
		}
	case "d":
		if !e.IsDir() {
			return false
		}
	}

	// 拡張子フィルタ
	if w.opts.Ext != "" {
		ext := strings.TrimPrefix(path.Ext(name), ".")
		if !strings.EqualFold(ext, w.opts.Ext) {
			return false
		}
	}

	// パターンマッチ
	if w.opts.Pattern == "" {
		return true
	}
	if w.opts.Regex {
		return w.re.MatchString(name)
	}
	matched, err := path.Match(w.glob, name)
	return err == nil && matched
}

// hasGlobChars はパターンにグロブ特殊文字が含まれるか返す。
func hasGlobChars(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

// isHidden は名前がドットで始まるかどうかを返す（クロスプラットフォーム簡易実装）。
func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}
