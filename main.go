package main

import (
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
  pattern     Glob pattern (default) or regex with -r. Matches filename only.
              If omitted, matches everything.
  directory   Root directory to search (default: current directory).

Options:
`

func main() {
	var (
		useRegex = flag.Bool("r", false, "Treat pattern as regular expression")
		fileType = flag.String("t", "", "Filter by type: f=file, d=directory")
		ext      = flag.String("e", "", "Filter by file extension (without dot, e.g. go)")
		hidden   = flag.Bool("hidden", false, "Include hidden files and directories")
		maxDepth = flag.Int("depth", 0, "Maximum search depth (0=unlimited)")
		doSort   = flag.Bool("sort", false, "Sort results alphabetically")
	)

	flag.CommandLine.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		flag.PrintDefaults()
	}

	// long-form aliases
	flag.BoolVar(useRegex, "regex", false, "Treat pattern as regular expression")
	flag.StringVar(fileType, "type", "", "Filter by type: f=file, d=directory")
	flag.StringVar(ext, "ext", "", "Filter by file extension (without dot)")

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
			if idx := strings.IndexByte(name, '='); idx >= 0 {
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
		Pattern:  pattern,
		Regex:    *useRegex,
		Type:     *fileType,
		Ext:      *ext,
		Hidden:   *hidden,
		MaxDepth: *maxDepth,
	}

	results := make(chan string, 256)
	errCh := make(chan error, 1)

	go func() {
		err := search.Run(root, opts, results)
		close(results)
		errCh <- err
	}()

	var all []string
	for p := range results {
		all = append(all, p)
	}
	if *doSort {
		sort.Strings(all)
	}
	for _, p := range all {
		fmt.Println(p)
	}

	if err := <-errCh; err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}
