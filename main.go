package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/misty/gf/search"
)

const usage = `Usage: gf [options] [pattern] [directory]

Search for files by name under a directory tree.

Arguments:
  pattern     Glob or regex (-r) pattern matched against filename only.
              Glob without special chars (*, ?, [) is treated as substring match.
              If omitted, matches everything.
  directory   Root directory to search (default: current directory).

Options:
  -r, --regex         Treat pattern as regular expression
  -p, --path          Match pattern against full path instead of filename only
  -t, --type  f|d     Filter by type: f=file, d=directory
  -e, --ext   EXT     Filter by file extension, without dot (e.g. go, png)
      --hidden        Include hidden files and directories (dot-prefixed)
      --depth N       Maximum search depth; 0 = unlimited (default: 0)
      --sort          Sort results alphabetically

Examples:
  gf '*.go'                     # .go files (glob)
  gf main                       # files whose name contains "main"
  gf -p ドールズ                # files under paths containing "ドールズ"
  gf -r '^test.*\.go$'          # regex: test*.go
  gf -t f -e go --sort ./src    # sorted .go files under src/
  gf --hidden --depth 2 .       # include hidden, max 2 levels deep
`

func main() {
	var (
		useRegex  = flag.Bool("r", false, "")
		matchPath = flag.Bool("p", false, "")
		fileType  = flag.String("t", "", "")
		ext       = flag.String("e", "", "")
		hidden    = flag.Bool("hidden", false, "")
		maxDepth  = flag.Int("depth", 0, "")
		doSort    = flag.Bool("sort", false, "")
	)

	flag.CommandLine.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}

	// long-form aliases
	flag.BoolVar(useRegex, "regex", false, "")
	flag.BoolVar(matchPath, "path", false, "")
	flag.StringVar(fileType, "type", "", "")
	flag.StringVar(ext, "ext", "", "")

	// bool フラグ名のセットを構築する（値を取らないフラグを判別するため）。
	boolFlags := map[string]bool{}
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		if _, ok := f.Value.(interface{ IsBoolFlag() bool }); ok {
			boolFlags[f.Name] = true
		}
	})

	// フラグと位置引数を分離してから parse する。
	// 標準 flag パッケージは最初の非フラグ引数でパース停止するため前処理する。
	var positional []string
	var flagArgs []string
	rawArgs := os.Args[1:]
	for i := 0; i < len(rawArgs); i++ {
		a := rawArgs[i]
		if a == "--" {
			positional = append(positional, rawArgs[i+1:]...)
			break
		}
		if len(a) > 0 && a[0] == '-' {
			flagArgs = append(flagArgs, a)
			// 値付きフラグ（-t f 形式）は次の引数もフラグ引数として扱う
			name := strings.TrimLeft(a, "-")
			if strings.Contains(name, "=") {
				// -t=f 形式はそのまま
			} else if !boolFlags[name] && i+1 < len(rawArgs) && (len(rawArgs[i+1]) == 0 || rawArgs[i+1][0] != '-') {
				i++
				flagArgs = append(flagArgs, rawArgs[i])
			}
		} else {
			positional = append(positional, a)
		}
	}
	if err := flag.CommandLine.Parse(flagArgs); err != nil {
		os.Exit(2)
	}

	args := positional

	// バリデーション
	if *fileType != "" && *fileType != "f" && *fileType != "d" {
		fmt.Fprintln(os.Stderr, "error: --type must be 'f' or 'd'")
		os.Exit(2)
	}

	pattern := ""
	root := "."

	switch len(args) {
	case 0:
		// パターンなし、カレントディレクトリ
	case 1:
		// 引数が存在するディレクトリならルートとして扱う
		if info, err := os.Stat(args[0]); err == nil && info.IsDir() {
			root = args[0]
		} else {
			pattern = args[0]
		}
	case 2:
		pattern = args[0]
		root = args[1]
	default:
		flag.CommandLine.Usage()
		os.Exit(2)
	}

	// 検索ディレクトリの存在確認
	if _, err := os.Stat(root); err != nil {
		fmt.Fprintf(os.Stderr, "error: directory not found: %s\n", root)
		os.Exit(1)
	}

	opts := search.Options{
		Pattern:   pattern,
		Regex:     *useRegex,
		MatchPath: *matchPath,
		Type:      *fileType,
		Ext:       *ext,
		Hidden:    *hidden,
		MaxDepth:  *maxDepth,
	}

	results := make(chan string, 8192)
	errCh := make(chan error, 1)

	go func() {
		err := search.Run(root, opts, results)
		close(results)
		errCh <- err
	}()

	w := bufio.NewWriterSize(os.Stdout, 1<<20) // 1MB 出力バッファ
	defer w.Flush()

	if *doSort {
		var all []string
		for p := range results {
			all = append(all, p)
		}
		sort.Strings(all)
		for _, p := range all {
			fmt.Fprintln(w, p)
		}
	} else {
		for p := range results {
			fmt.Fprintln(w, p)
		}
	}

	if err := <-errCh; err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}
