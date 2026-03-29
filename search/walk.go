package search

import (
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

type parallelWalker struct {
	opts       Options
	patternFn  func(name string) bool // パターンマッチ専用（型・拡張子を含まない）
	excludeFns []func(string) bool    // 除外パターン群
	extLower   string                 // 小文字化済み拡張子
	results    chan<- string
	wg         sync.WaitGroup
	sem        chan struct{}
	// limit サポート
	remaining int64      // 残り送信可能件数（Limit=0 のとき未使用）
	done      chan struct{}
	doneOnce  sync.Once
}

func newWalker(opts Options, patternFn func(string) bool, excludeFns []func(string) bool, results chan<- string) *parallelWalker {
	w := &parallelWalker{
		opts:       opts,
		patternFn:  patternFn,
		excludeFns: excludeFns,
		extLower:   strings.ToLower(opts.Ext),
		results:    results,
		sem:        make(chan struct{}, runtime.NumCPU()*4),
		done:       make(chan struct{}),
	}
	if opts.Limit > 0 {
		w.remaining = int64(opts.Limit)
	}
	return w
}

// walk は dirPath（スラッシュ区切り）以下を再帰的に処理する。
func (w *parallelWalker) walk(dirPath string, depth int) {
	defer w.wg.Done()

	// limit 到達済みなら即リターン
	if w.isDone() {
		return
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return
	}

	for _, e := range entries {
		if w.isDone() {
			return
		}

		name := e.Name()

		if !w.opts.Hidden && isHidden(name) {
			continue
		}
		if w.isExcluded(name) {
			continue // ディレクトリの場合は再帰も行わない
		}

		childDepth := depth + 1

		if e.IsDir() {
			if w.opts.MaxDepth > 0 && childDepth > w.opts.MaxDepth {
				continue
			}
			childPath := dirPath + "/" + name
			if w.opts.Type != "f" && w.matches(name, childPath) && w.matchesTime(e) {
				w.send(childPath)
			}
			if w.isDone() {
				return
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
			if w.opts.MatchPath {
				childPath := dirPath + "/" + name
				if w.matches(name, childPath) && w.matchesTime(e) {
					w.send(childPath)
				}
			} else if w.matches(name, "") && w.matchesTime(e) {
				w.send(dirPath + "/" + name)
			}
		}
	}
}

// send は results に p を送信する。limit に達したら done をクローズする。
func (w *parallelWalker) send(p string) {
	if w.opts.Limit <= 0 {
		w.results <- p
		return
	}
	n := atomic.AddInt64(&w.remaining, -1)
	if n < 0 {
		return
	}
	w.results <- p
	if n == 0 {
		w.doneOnce.Do(func() { close(w.done) })
	}
}

// isDone は limit に達しているかどうかを返す。
func (w *parallelWalker) isDone() bool {
	if w.opts.Limit <= 0 {
		return false
	}
	select {
	case <-w.done:
		return true
	default:
		return false
	}
}

// matchesTime は e が時間フィルタを通過するか返す。
// フィルタが未設定（ゼロ値）のときは常に true。
// Info() の呼び出しは時間フィルタが有効なときのみ行う（lazy 評価）。
func (w *parallelWalker) matchesTime(e os.DirEntry) bool {
	opts := &w.opts
	if opts.ModifiedAfter.IsZero() && opts.ModifiedBefore.IsZero() &&
		opts.CreatedAfter.IsZero() && opts.CreatedBefore.IsZero() {
		return true
	}
	info, err := e.Info()
	if err != nil {
		return false
	}
	mtime := info.ModTime()
	if !opts.ModifiedAfter.IsZero() && !mtime.After(opts.ModifiedAfter) {
		return false
	}
	if !opts.ModifiedBefore.IsZero() && !mtime.Before(opts.ModifiedBefore) {
		return false
	}
	if !opts.CreatedAfter.IsZero() || !opts.CreatedBefore.IsZero() {
		btime := birthTime(info)
		if !btime.IsZero() { // ゼロ値 = プラットフォーム非サポート → フィルタをスキップ
			if !opts.CreatedAfter.IsZero() && !btime.After(opts.CreatedAfter) {
				return false
			}
			if !opts.CreatedBefore.IsZero() && !btime.Before(opts.CreatedBefore) {
				return false
			}
		}
	}
	return true
}

// isExcluded は name が除外パターンにマッチするかを返す。
func (w *parallelWalker) isExcluded(name string) bool {
	for _, fn := range w.excludeFns {
		if fn(name) {
			return true
		}
	}
	return false
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
