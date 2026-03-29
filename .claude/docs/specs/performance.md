# パフォーマンス設計

- `os.ReadDir` + goroutine + セマフォ（`NumCPU × 4`）で並行走査
- セマフォ満杯時はインラインフォールバック（デッドロック回避）
- glob fast path: `*.ext` → `HasSuffix`、`prefix*` → `HasPrefix`、`*x*` → `Contains`
- 非マッチエントリのパス文字列は lazy 構築（アロケーション削減）
- フィルタ評価順: hidden → exclude → depth → type → pattern/ext → **time（最後・lazy）**
- `bufio.NewWriterSize(1MB)` で stdout バッファリング
- `--limit` は `atomic.Int64` + `done` チャネルで goroutine を即停止

## ベンチマーク（336K ファイル、SSD）

| | gf | find |
|---|---|---|
| 全件走査 | **185ms** | 446ms |
| glob `*.png` | **90ms** | 1,249ms |
