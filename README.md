# gf — Fast File Finder

[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go&logoColor=white)](https://golang.org)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey?style=flat)](https://github.com/misty/gf)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat)](LICENSE)

`gf` is a fast command-line file finder written in Go.
It walks directory trees concurrently using goroutines and supports both glob and regex pattern matching.

## Why gf?

| | `gf` | `find` |
|---|---|---|
| Full scan (336K files) | **185 ms** | 446 ms |
| Glob pattern (`*.png`) | **90 ms** | 1,249 ms |

> Benchmarked on a local SSD with 336,000 files.

## Installation

```bash
go install github.com/misty/gf@latest
```

Or build from source:

```bash
git clone https://github.com/misty/gf
cd gf
go build -o gf .
```

## Usage

```
gf [options] [pattern] [directory]
```

| Argument | Description |
|---|---|
| `pattern` | Glob or regex (`-r`) pattern matched against the filename. Omit to match everything. A pattern without glob characters (`*`, `?`, `[`) is treated as a substring match. |
| `directory` | Root directory to search (default: current directory). |

## Options

| Flag | Long form | Description |
|---|---|---|
| `-r` | `--regex` | Treat pattern as a regular expression |
| `-p` | `--path` | Match pattern against the full path instead of filename only |
| `-t f\|d` | `--type f\|d` | Filter by type: `f` = files, `d` = directories |
| `-e EXT` | `--ext EXT` | Filter by file extension, without dot (e.g. `go`, `png`) |
| | `--hidden` | Include hidden files and directories (dot-prefixed) |
| | `--depth N` | Maximum search depth; `0` = unlimited (default: `0`) |
| | `--sort` | Sort results alphabetically |
| `-n N` | `--limit N` | Stop after N results; `0` = unlimited (default: `0`) |
| | `--exclude PAT` | Exclude entries matching glob PAT; repeatable |

## Examples

```bash
# Glob patterns
gf '*.go'                        # files ending in .go
gf 'main'                        # files whose name contains "main"
gf 'test*' -e go                 # .go files starting with "test"

# Regex
gf -r '^test.*\.go$'             # regex: test*.go
gf -r '\.(png|jpg)$'             # images

# Type & extension filters
gf -t f -e go --sort ./src       # sorted .go files under src/
gf -t d 'cache'                  # directories whose name contains "cache"

# Full path matching
gf -p 'Scripts'                  # files under any path containing "Scripts"
gf -p -r 'src/.*_test\.go$'      # test files under src/ (regex on full path)

# Hidden files & depth
gf --hidden '.env'               # find .env files
gf --depth 2 '*.md' ./docs       # .md files up to 2 levels deep

# Limit
gf '*.log' -n 5                  # first 5 .log files
gf -n 1 'main.go'                # stop at first match

# Exclude
gf '*.go' --exclude '*_test*'              # skip test files
gf --exclude node_modules --exclude dist   # skip build dirs
gf '*.ts' --exclude '*.d.ts' ./src        # TypeScript, no declaration files
```

## Pattern Matching

### Glob (default)

| Pattern | Behavior |
|---|---|
| `*.go` | Files ending in `.go` |
| `main*` | Files starting with `main` |
| `test` | Files whose name **contains** `test` (auto-wrapped as `*test*`) |
| `[mf]oo.go` | Matches `moo.go` or `foo.go` |

### Regex (`-r`)

Uses Go's `regexp` package (RE2 syntax).

```bash
gf -r '^main\.go$'       # exact filename match
gf -r '\.(png|jpg|gif)$' # multiple extensions
gf -r '^[0-9]+'          # starts with a digit
```

### Full path matching (`-p`)

By default, patterns are matched against the **filename only**.
With `-p`, the pattern is applied to the **full slash-separated path**.

```bash
gf -p 'Assets/Textures'          # files anywhere under Assets/Textures/
gf -p -r 'Assets/.*\.png$'       # PNG files under Assets/
```

## Output

- One path per line, printed to stdout
- Paths always use forward slashes (`/`), even on Windows
- Order is non-deterministic by default (concurrent walk)
- Use `--sort` for alphabetical order

## Performance Design

- Concurrent directory walk using goroutines and a semaphore (`NumCPU × 4`)
- Glob fast paths: `*.ext` → `HasSuffix`, `prefix*` → `HasPrefix`, `*x*` → `Contains`
- Lazy path string construction (skips allocation for non-matching entries)
- 1 MB buffered stdout writer

## Project Structure

```
gf/
├── main.go               # Entry point, flag parsing, output
├── console_windows.go    # Windows UTF-8 console setup
├── search/
│   ├── search.go         # Options & Run()
│   ├── walk.go           # Parallel directory walker
│   └── pattern.go        # Pattern compilation (glob / regex)
├── go.mod
└── SPEC.md
```

## License

[MIT License](LICENSE) — Copyright (c) 2026 misty
