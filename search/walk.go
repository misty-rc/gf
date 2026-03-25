package search

import (
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
)

type parallelWalker struct {
	opts      Options
	patternFn func(name string) bool // パターンマッチ専用（型・拡張子を含まない）
	extLower  string                 // 小文字化済み拡張子
	results   chan<- string
	wg        sync.WaitGroup
	sem       chan struct{}
}

func newWalker(opts Options, patternFn func(string) bool, results chan<- string) *parallelWalker {
	// I/Oバウンド処理のため NumCPU の4倍まで並行実行を許可する
	return &parallelWalker{
		opts:      opts,
		patternFn: patternFn,
		extLower:  strings.ToLower(opts.Ext),
		results:   results,
		sem:       make(chan struct{}, runtime.NumCPU()*4),
	}
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

		if !w.opts.Hidden && isHidden(name) {
			continue
		}

		childDepth := depth + 1

		if e.IsDir() {
			if w.opts.MaxDepth > 0 && childDepth > w.opts.MaxDepth {
				continue
			}
			// ディレクトリは再帰にも必要なので先にパスを構築
			childPath := dirPath + "/" + name
			if w.opts.Type != "f" && w.matches(name, childPath) {
				w.results <- childPath
			}
			w.wg.Add(1)
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
			if w.opts.Type == "d" {
				continue
			}
			// --path 時はパスが必要なので先に構築、それ以外はマッチ後に構築
			if w.opts.MatchPath {
				childPath := dirPath + "/" + name
				if w.matches(name, childPath) {
					w.results <- childPath
				}
			} else if w.matches(name, "") {
				w.results <- dirPath + "/" + name
			}
		}
	}
}

// matches は拡張子フィルタとパターンマッチを適用する。
// MatchPath が true のとき pattern は fullPath に適用し、それ以外は name に適用する。
func (w *parallelWalker) matches(name, fullPath string) bool {
	if w.extLower != "" {
		ext := strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))
		if ext != w.extLower {
			return false
		}
	}
	if w.opts.MatchPath {
		return w.patternFn(fullPath)
	}
	return w.patternFn(name)
}

func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}
