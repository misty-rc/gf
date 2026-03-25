# gf — 高速ファイル検索コマンド

[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go&logoColor=white)](https://golang.org)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey?style=flat)](https://github.com/misty/gf)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat)](LICENSE)

`gf` は Go 製の高速ファイル検索 CLI ツールです。
goroutine による並行ディレクトリ走査を採用し、グロブ・正規表現の両方によるパターンマッチをサポートします。

## なぜ gf？

| | `gf` | `find` |
|---|---|---|
| 全件走査（33万6千ファイル） | **185 ms** | 446 ms |
| グロブ検索（`*.png`） | **90 ms** | 1,249 ms |

> SSD 上のローカルディレクトリ（336,000 ファイル）でのベンチマーク結果。

## インストール

```bash
go install github.com/misty/gf@latest
```

ソースからビルドする場合:

```bash
git clone https://github.com/misty/gf
cd gf
go build -o gf .
```

## 使い方

```
gf [オプション] [パターン] [検索ディレクトリ]
```

| 引数 | 説明 |
|---|---|
| `パターン` | ファイル名に対するグロブまたは正規表現（`-r`）パターン。省略すると全件マッチ。グロブ記号（`*`、`?`、`[`）を含まないパターンは部分一致として扱われる。 |
| `検索ディレクトリ` | 検索起点ディレクトリ（デフォルト: カレントディレクトリ）。 |

## オプション

| 短形式 | 長形式 | 説明 |
|---|---|---|
| `-r` | `--regex` | パターンを正規表現として扱う |
| `-p` | `--path` | パターンをファイル名でなくフルパスに適用する |
| `-t f\|d` | `--type f\|d` | 種別フィルタ: `f` = ファイルのみ, `d` = ディレクトリのみ |
| `-e EXT` | `--ext EXT` | 拡張子フィルタ（ドットなし。例: `go`, `png`） |
| | `--hidden` | 隠しファイル・ディレクトリを含める（デフォルト: 除外） |
| | `--depth N` | 最大探索深さ（`0` = 無制限、デフォルト: `0`） |
| | `--sort` | 結果をアルファベット順にソートして出力 |

## 使用例

```bash
# グロブパターン
gf '*.go'                        # .go ファイル
gf 'main'                        # 名前に "main" を含むファイル
gf 'test*' -e go                 # "test" で始まる .go ファイル

# 正規表現
gf -r '^test.*\.go$'             # 正規表現: test*.go
gf -r '\.(png|jpg)$'             # 画像ファイル

# 種別・拡張子フィルタ
gf -t f -e go --sort ./src       # src/ 以下の .go ファイルをソートして出力
gf -t d 'cache'                  # "cache" を含むディレクトリ

# フルパスマッチ
gf -p 'Scripts'                  # パスに "Scripts" を含むファイル
gf -p -r 'src/.*_test\.go$'      # src/ 以下のテストファイル（正規表現）

# 隠しファイル・深さ制限
gf --hidden '.env'               # .env ファイルを検索
gf --depth 2 '*.md' ./docs       # docs/ 以下 2階層までの .md ファイル
```

## パターンマッチの詳細

### グロブ（デフォルト）

| パターン | 動作 |
|---|---|
| `*.go` | `.go` で終わるファイル |
| `main*` | `main` で始まるファイル |
| `test` | 名前に `test` を**含む**ファイル（自動的に `*test*` に展開） |
| `[mf]oo.go` | `moo.go` または `foo.go` にマッチ |

### 正規表現（`-r`）

Go の `regexp` パッケージ（RE2 構文）を使用します。

```bash
gf -r '^main\.go$'       # ファイル名の完全一致
gf -r '\.(png|jpg|gif)$' # 複数の拡張子
gf -r '^[0-9]+'          # 数字で始まるファイル
```

### フルパスマッチ（`-p`）

デフォルトではパターンは**ファイル名のみ**に適用されます。
`-p` を指定すると、パターンが**フルパス（スラッシュ区切り）**に適用されます。

```bash
gf -p 'Assets/Textures'          # Assets/Textures/ 以下のファイル
gf -p -r 'Assets/.*\.png$'       # Assets/ 以下の PNG ファイル
gf -p 'ドールズフロントライン'   # パスに日本語を含むファイル
```

## 出力形式

- 1行につき1パスを標準出力に出力
- パスは常にスラッシュ区切り（Windows でも `/`）
- デフォルトの出力順は不定（並行処理のため）
- `--sort` でアルファベット順にソート

## パフォーマンス設計

- goroutine + セマフォ（`NumCPU × 4`）による並行ディレクトリ走査
- グロブ fast path: `*.ext` → `HasSuffix`、`prefix*` → `HasPrefix`、`*x*` → `Contains`
- 非マッチエントリのパス文字列アロケーションを遅延構築で削減
- 1 MB バッファ付き stdout ライター

## ファイル構成

```
gf/
├── main.go               # エントリーポイント・フラグ定義・出力
├── console_windows.go    # Windows UTF-8 コンソール設定
├── search/
│   ├── search.go         # Options 定義・Run()
│   ├── walk.go           # 並行ディレクトリウォーカー
│   └── pattern.go        # パターンコンパイル（グロブ・正規表現）
├── go.mod
└── SPEC.md
```

## ライセンス

MIT
