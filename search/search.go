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
	opts      Options
	patternFn func(name string) bool // パターンマッチ専用（型・拡張子を含まない）
	extLower  string                 // 小文字化済み拡張子（EqualFold の代わりに ToLower で比較）
	results   chan<- string
	wg        sync.WaitGroup
	sem       chan struct{}
}

// Run は root 以下を並行走査し、マッチしたパスを results チャネルに送信する。
func Run(root string, opts Options, results chan<- string) error {
	patternFn, err := compilePattern(opts)
	if err != nil {
		return err
	}

	w := &parallelWalker{
		opts:      opts,
		patternFn: patternFn,
		extLower:  strings.ToLower(opts.Ext),
		results:   results,
		sem:       make(chan struct{}, runtime.NumCPU()),
	}

	w.wg.Add(1)
	go w.walk(filepath.ToSlash(root), 0)
	w.wg.Wait()
	return nil
}

// compilePattern はパターン部分のマッチ関数を構築する。
// 単純な glob（*.ext, prefix*, *contains*）は strings の fast path を使い、
// path.Match の呼び出しコストを回避する。
func compilePattern(opts Options) (func(string) bool, error) {
	if opts.Pattern == "" {
		return func(string) bool { return true }, nil
	}
	if opts.Regex {
		re, err := regexp.Compile(opts.Pattern)
		if err != nil {
			return nil, err
		}
		return re.MatchString, nil
	}

	glob := opts.Pattern
	if !hasGlobChars(glob) {
		glob = "*" + glob + "*" // グロブ記号なし → 部分一致に変換
	}

	// *.suffix — HasSuffix で代替
	if strings.HasPrefix(glob, "*") && !strings.ContainsAny(glob[1:], "*?[") {
		suffix := glob[1:]
		return func(name string) bool { return strings.HasSuffix(name, suffix) }, nil
	}
	// prefix.* — HasPrefix で代替
	if strings.HasSuffix(glob, "*") && !strings.ContainsAny(glob[:len(glob)-1], "*?[") {
		prefix := glob[:len(glob)-1]
		return func(name string) bool { return strings.HasPrefix(name, prefix) }, nil
	}
	// *contains* — Contains で代替
	if strings.HasPrefix(glob, "*") && strings.HasSuffix(glob, "*") {
		inner := glob[1 : len(glob)-1]
		if !strings.ContainsAny(inner, "*?[") {
			return func(name string) bool { return strings.Contains(name, inner) }, nil
		}
	}

	// 汎用グロブ（path.Match）
	return func(name string) bool {
		matched, err := path.Match(glob, name)
		return err == nil && matched
	}, nil
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
			if w.opts.Type != "f" && w.matchesName(name) {
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
			// ファイルはマッチした場合のみパスを構築（非マッチ分のアロケーションを削減）
			if w.opts.Type != "d" && w.matchesName(name) {
				w.results <- dirPath + "/" + name
			}
		}
	}
}

// matchesName は拡張子フィルタとパターンマッチを適用する。
func (w *parallelWalker) matchesName(name string) bool {
	if w.extLower != "" {
		ext := strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))
		if ext != w.extLower {
			return false
		}
	}
	return w.patternFn(name)
}

func hasGlobChars(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}
