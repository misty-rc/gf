package search

import (
	"path"
	"regexp"
	"strings"
)

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
	// prefix* — HasPrefix で代替
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

func hasGlobChars(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

// compileExcludes は除外パターン群をマッチ関数のスライスにコンパイルする。
// パターン形式はグロブ（部分一致含む）のみ。正規表現は不可。
func compileExcludes(patterns []string) ([]func(string) bool, error) {
	fns := make([]func(string) bool, 0, len(patterns))
	for _, p := range patterns {
		fn, err := compilePattern(Options{Pattern: p, Regex: false})
		if err != nil {
			return nil, err
		}
		fns = append(fns, fn)
	}
	return fns, nil
}
