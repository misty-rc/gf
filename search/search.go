package search

import (
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strings"
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

// Run は root 以下を走査し、マッチしたパスを results チャネルに送信する。
// マッチング処理は WalkDir コールバック内でインライン実行する。
func Run(root string, opts Options, results chan<- string) error {
	var re *regexp.Regexp
	if opts.Regex && opts.Pattern != "" {
		var err error
		re, err = regexp.Compile(opts.Pattern)
		if err != nil {
			return err
		}
	}

	// グロブパターンの事前正規化（WalkDir内で毎回やらないよう）
	glob := opts.Pattern
	if !opts.Regex && glob != "" && !hasGlobChars(glob) {
		glob = "*" + glob + "*"
	}

	rootDepth := 0
	if opts.MaxDepth > 0 {
		rootDepth = strings.Count(filepath.ToSlash(root), "/")
	}

	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// アクセス権限エラーなどはスキップ
			return nil
		}
		if p == root {
			return nil
		}

		// 深さチェック
		if opts.MaxDepth > 0 {
			depth := strings.Count(filepath.ToSlash(p), "/") - rootDepth
			if depth > opts.MaxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// 隠しファイル・ディレクトリの除外
		if !opts.Hidden && isHidden(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if matches(d, opts, glob, re) {
			results <- filepath.ToSlash(p)
		}
		return nil
	})
}

// hasGlobChars はパターンにグロブ特殊文字が含まれるか返す。
func hasGlobChars(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

func matches(d fs.DirEntry, opts Options, glob string, re *regexp.Regexp) bool {
	name := d.Name()

	// タイプフィルタ
	switch opts.Type {
	case "f":
		if d.IsDir() {
			return false
		}
	case "d":
		if !d.IsDir() {
			return false
		}
	}

	// 拡張子フィルタ
	if opts.Ext != "" {
		ext := strings.TrimPrefix(path.Ext(name), ".")
		if !strings.EqualFold(ext, opts.Ext) {
			return false
		}
	}

	// パターンマッチ
	if opts.Pattern == "" {
		return true
	}
	if opts.Regex {
		return re.MatchString(name)
	}
	matched, err := path.Match(glob, name)
	return err == nil && matched
}

// isHidden は名前がドットで始まるかどうかを返す（クロスプラットフォーム簡易実装）。
func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}
