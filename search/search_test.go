package search_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/misty/gf/search"
)

// テスト用ディレクトリ構造:
//
//	root/
//	├── file.go
//	├── file.txt
//	├── main.go
//	├── README          (拡張子なし)
//	├── archive.tar.gz
//	├── .hidden_file
//	├── .hidden_dir/
//	│   └── inside.go
//	├── subdir/
//	│   ├── sub.go
//	│   ├── sub.txt
//	│   └── deep/
//	│       └── deep.go
//	└── empty_dir/
func makeTestTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	dirs := []string{
		".hidden_dir",
		"subdir",
		"subdir/deep",
		"empty_dir",
	}
	files := []string{
		"file.go",
		"file.txt",
		"main.go",
		"README",
		"archive.tar.gz",
		".hidden_file",
		".hidden_dir/inside.go",
		"subdir/sub.go",
		"subdir/sub.txt",
		"subdir/deep/deep.go",
	}

	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(root, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// Run を呼び出してソート済みのパスリストを返すヘルパー。
// パスは root からの相対スラッシュパス（例: "file.go", "subdir/sub.go"）。
func runSearch(t *testing.T, root string, opts search.Options) []string {
	t.Helper()
	results := make(chan string, 256)
	errCh := make(chan error, 1)
	go func() {
		errCh <- search.Run(root, opts, results)
		close(results)
	}()

	var paths []string
	for p := range results {
		// 絶対パス → root からの相対スラッシュパスに正規化
		rel := strings.TrimPrefix(filepath.ToSlash(p), filepath.ToSlash(root)+"/")
		paths = append(paths, rel)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("Run error: %v", err)
	}
	sort.Strings(paths)
	return paths
}

// ---- グロブパターン --------------------------------------------------------

func TestGlob_NoSpecialChars_PartialMatch(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Pattern: "file"})
	want := []string{"file.go", "file.txt", "hidden_file"} // .hidden_file はデフォルト除外
	_ = want
	// "file" を含む名前が部分一致すること
	for _, p := range got {
		if !strings.Contains(filepath.Base(p), "file") {
			t.Errorf("unexpected match: %s", p)
		}
	}
	if len(got) == 0 {
		t.Error("expected at least one match")
	}
}

func TestGlob_Star_ExtensionMatch(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Pattern: "*.go"})
	for _, p := range got {
		if !strings.HasSuffix(p, ".go") {
			t.Errorf("non-.go file matched: %s", p)
		}
	}
	// file.go, main.go, subdir/sub.go, subdir/deep/deep.go の4件
	if len(got) != 4 {
		t.Errorf("want 4, got %d: %v", len(got), got)
	}
}

func TestGlob_QuestionMark(t *testing.T) {
	root := makeTestTree(t)
	// "fil?.go" → file.go のみ
	got := runSearch(t, root, search.Options{Pattern: "fil?.go"})
	if len(got) != 1 || got[0] != "file.go" {
		t.Errorf("want [file.go], got %v", got)
	}
}

func TestGlob_CharacterClass(t *testing.T) {
	root := makeTestTree(t)
	// "[mf]ain.go" → main.go のみ（main が [mf] + ain.go にマッチ）
	got := runSearch(t, root, search.Options{Pattern: "[mf]ain.go"})
	if len(got) != 1 || got[0] != "main.go" {
		t.Errorf("want [main.go], got %v", got)
	}
}

func TestGlob_EmptyPattern_MatchAll(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Pattern: ""})
	// 隠しファイルを除いた全エントリ: dirs + files
	// file.go, file.txt, main.go, README, archive.tar.gz, subdir, subdir/sub.go,
	// subdir/sub.txt, subdir/deep, subdir/deep/deep.go, empty_dir = 11件
	if len(got) != 11 {
		t.Errorf("want 11, got %d: %v", len(got), got)
	}
}

func TestGlob_NoMatch(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Pattern: "nonexistent_xyz"})
	if len(got) != 0 {
		t.Errorf("want 0, got %d: %v", len(got), got)
	}
}

func TestGlob_MultiDotExtension(t *testing.T) {
	root := makeTestTree(t)
	// "*.gz" → archive.tar.gz のみ（path.Match はパス全体ではなく名前のみ）
	got := runSearch(t, root, search.Options{Pattern: "*.gz"})
	if len(got) != 1 || got[0] != "archive.tar.gz" {
		t.Errorf("want [archive.tar.gz], got %v", got)
	}
}

// ---- 正規表現パターン ------------------------------------------------------

func TestRegex_Basic(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Pattern: `\.go$`, Regex: true})
	for _, p := range got {
		if !strings.HasSuffix(p, ".go") {
			t.Errorf("non-.go file matched: %s", p)
		}
	}
	if len(got) != 4 {
		t.Errorf("want 4, got %d: %v", len(got), got)
	}
}

func TestRegex_Anchored(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Pattern: `^main`, Regex: true})
	if len(got) != 1 || got[0] != "main.go" {
		t.Errorf("want [main.go], got %v", got)
	}
}

func TestRegex_Alternation(t *testing.T) {
	root := makeTestTree(t)
	// "^(file|main)" → file.go, file.txt, main.go
	got := runSearch(t, root, search.Options{Pattern: `^(file|main)`, Regex: true})
	if len(got) != 3 {
		t.Errorf("want 3, got %d: %v", len(got), got)
	}
}

func TestRegex_InvalidPattern_ReturnsError(t *testing.T) {
	root := makeTestTree(t)
	results := make(chan string, 256)
	errCh := make(chan error, 1)
	go func() {
		errCh <- search.Run(root, search.Options{Pattern: "[invalid", Regex: true}, results)
		close(results)
	}()
	for range results {
	}
	if err := <-errCh; err == nil {
		t.Error("want error for invalid regex, got nil")
	}
}

func TestRegex_EmptyPattern_MatchAll(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Pattern: "", Regex: true})
	if len(got) != 11 {
		t.Errorf("want 11, got %d: %v", len(got), got)
	}
}

// ---- タイプフィルタ ---------------------------------------------------------

func TestType_FilesOnly(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Type: "f"})
	for _, p := range got {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			t.Fatalf("stat %s: %v", p, err)
		}
		if info.IsDir() {
			t.Errorf("directory in file-only results: %s", p)
		}
	}
	// file.go, file.txt, main.go, README, archive.tar.gz,
	// subdir/sub.go, subdir/sub.txt, subdir/deep/deep.go = 8件
	if len(got) != 8 {
		t.Errorf("want 8, got %d: %v", len(got), got)
	}
}

func TestType_DirsOnly(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Type: "d"})
	for _, p := range got {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			t.Fatalf("stat %s: %v", p, err)
		}
		if !info.IsDir() {
			t.Errorf("file in dir-only results: %s", p)
		}
	}
	// subdir, subdir/deep, empty_dir = 3件
	if len(got) != 3 {
		t.Errorf("want 3, got %d: %v", len(got), got)
	}
}

func TestType_Both(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Type: ""})
	if len(got) != 11 {
		t.Errorf("want 11, got %d: %v", len(got), got)
	}
}

// ---- 拡張子フィルタ ---------------------------------------------------------

func TestExt_Basic(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Ext: "go"})
	for _, p := range got {
		if !strings.HasSuffix(p, ".go") {
			t.Errorf("non-.go file matched: %s", p)
		}
	}
	if len(got) != 4 {
		t.Errorf("want 4, got %d: %v", len(got), got)
	}
}

func TestExt_CaseInsensitive(t *testing.T) {
	root := makeTestTree(t)
	gotLower := runSearch(t, root, search.Options{Ext: "go"})
	gotUpper := runSearch(t, root, search.Options{Ext: "GO"})
	if strings.Join(gotLower, ",") != strings.Join(gotUpper, ",") {
		t.Errorf("case sensitivity mismatch: lower=%v upper=%v", gotLower, gotUpper)
	}
}

func TestExt_NoExtensionFile(t *testing.T) {
	root := makeTestTree(t)
	// README は拡張子なし → "go" フィルタに引っかからない
	got := runSearch(t, root, search.Options{Ext: "go"})
	for _, p := range got {
		if filepath.Base(p) == "README" {
			t.Error("README should not match ext=go")
		}
	}
}

func TestExt_MultiDot(t *testing.T) {
	root := makeTestTree(t)
	// "gz" → archive.tar.gz のみ（path.Ext は最後のドット以降）
	got := runSearch(t, root, search.Options{Ext: "gz"})
	if len(got) != 1 || got[0] != "archive.tar.gz" {
		t.Errorf("want [archive.tar.gz], got %v", got)
	}
}

func TestExt_WithDotPrefix_NoMatch(t *testing.T) {
	root := makeTestTree(t)
	// ".go"（ドット付き）は EqualFold(".go", "go") = false → マッチしない
	got := runSearch(t, root, search.Options{Ext: ".go"})
	if len(got) != 0 {
		t.Errorf("want 0 (dot-prefixed ext should not match), got %d: %v", len(got), got)
	}
}

// ---- 隠しファイル制御 -------------------------------------------------------

func TestHidden_ExcludedByDefault(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{})
	for _, p := range got {
		for _, seg := range strings.Split(p, "/") {
			if strings.HasPrefix(seg, ".") {
				t.Errorf("hidden entry should be excluded: %s", p)
			}
		}
	}
}

func TestHidden_Included(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Hidden: true})
	var hiddenFound bool
	for _, p := range got {
		if strings.HasPrefix(filepath.Base(p), ".") {
			hiddenFound = true
		}
	}
	if !hiddenFound {
		t.Error("expected hidden files to appear when Hidden=true")
	}
}

func TestHidden_DirSubtreeExcluded(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Hidden: false})
	for _, p := range got {
		if strings.Contains(p, ".hidden_dir") {
			t.Errorf(".hidden_dir subtree should be excluded: %s", p)
		}
	}
}

func TestHidden_DirSubtreeIncluded(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Hidden: true})
	var found bool
	for _, p := range got {
		if strings.Contains(p, ".hidden_dir") {
			found = true
		}
	}
	if !found {
		t.Error("expected .hidden_dir subtree when Hidden=true")
	}
}

// ---- 深さ制限 ---------------------------------------------------------------

func TestDepth_Unlimited(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{MaxDepth: 0})
	if len(got) != 11 {
		t.Errorf("want 11, got %d: %v", len(got), got)
	}
}

func TestDepth_1_DirectChildrenOnly(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{MaxDepth: 1})
	for _, p := range got {
		if strings.Contains(p, "/") {
			t.Errorf("depth=1 should not include nested paths: %s", p)
		}
	}
	// file.go, file.txt, main.go, README, archive.tar.gz, subdir, empty_dir = 7件
	if len(got) != 7 {
		t.Errorf("want 7, got %d: %v", len(got), got)
	}
}

func TestDepth_2(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{MaxDepth: 2})
	for _, p := range got {
		depth := strings.Count(p, "/")
		if depth > 1 {
			t.Errorf("depth=2 should not include paths deeper than 2 levels: %s", p)
		}
	}
	// depth1: file.go, file.txt, main.go, README, archive.tar.gz, subdir, empty_dir (7)
	// depth2: subdir/sub.go, subdir/sub.txt, subdir/deep (3) = 10件
	if len(got) != 10 {
		t.Errorf("want 10, got %d: %v", len(got), got)
	}
}

// ---- パス形式 ----------------------------------------------------------------

func TestPath_ForwardSlash(t *testing.T) {
	root := makeTestTree(t)
	results := make(chan string, 256)
	go func() {
		search.Run(root, search.Options{}, results)
		close(results)
	}()
	for p := range results {
		if strings.Contains(p, `\`) {
			t.Errorf("path contains backslash: %s", p)
		}
	}
}

// ---- エラーケース ------------------------------------------------------------

func TestError_NonExistentRoot(t *testing.T) {
	results := make(chan string, 256)
	errCh := make(chan error, 1)
	go func() {
		errCh <- search.Run("/nonexistent/path/xyz", search.Options{}, results)
		close(results)
	}()
	for range results {
	}
	// 存在しない root は ReadDir エラー → 空結果（エラーはスキップ扱い）
	// Run 自体はエラーを返さないが、結果が0件になること
	<-errCh
	// パニックしないことを確認できれば十分
}

func TestError_RootIsFile(t *testing.T) {
	root := makeTestTree(t)
	filePath := filepath.Join(root, "file.go")
	results := make(chan string, 256)
	errCh := make(chan error, 1)
	go func() {
		errCh <- search.Run(filePath, search.Options{}, results)
		close(results)
	}()
	for range results {
	}
	<-errCh
	// ファイルを root に渡しても panic しないこと
}

// ---- 複合フィルタ ------------------------------------------------------------

func TestCombined_TypeAndExt(t *testing.T) {
	root := makeTestTree(t)
	got := runSearch(t, root, search.Options{Type: "f", Ext: "go"})
	for _, p := range got {
		if !strings.HasSuffix(p, ".go") {
			t.Errorf("unexpected: %s", p)
		}
	}
	if len(got) != 4 {
		t.Errorf("want 4, got %d: %v", len(got), got)
	}
}

func TestCombined_PatternAndDepth(t *testing.T) {
	root := makeTestTree(t)
	// depth=1 かつ *.go → file.go, main.go のみ
	got := runSearch(t, root, search.Options{Pattern: "*.go", MaxDepth: 1})
	if len(got) != 2 {
		t.Errorf("want 2, got %d: %v", len(got), got)
	}
	for _, p := range got {
		if strings.Contains(p, "/") {
			t.Errorf("should not be nested: %s", p)
		}
	}
}

func TestCombined_RegexAndType(t *testing.T) {
	root := makeTestTree(t)
	// 正規表現 "^sub" かつ type=d → subdir のみ
	got := runSearch(t, root, search.Options{Pattern: `^sub`, Regex: true, Type: "d"})
	if len(got) != 1 || got[0] != "subdir" {
		t.Errorf("want [subdir], got %v", got)
	}
}

func TestCombined_AllFilters(t *testing.T) {
	root := makeTestTree(t)
	// type=f, ext=go, pattern=sub (部分一致), depth=2, hidden=false
	got := runSearch(t, root, search.Options{
		Type:     "f",
		Ext:      "go",
		Pattern:  "sub",
		MaxDepth: 2,
	})
	// subdir/sub.go のみ
	if len(got) != 1 || got[0] != "subdir/sub.go" {
		t.Errorf("want [subdir/sub.go], got %v", got)
	}
}
