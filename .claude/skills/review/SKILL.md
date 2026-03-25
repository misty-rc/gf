---
name: review
description: gf のコードをレビューする。「レビュー」「review」「コードを確認して」などのキーワードが出たら必ずこのスキルを使う。
---

# コードレビュー手順

## 対象の特定

```bash
git diff --name-only HEAD
```

## 実施

1. 対象ファイルを Read ツールで読み込む
2. `C:\Users\misty\.claude\standards\review.md` を参照する
3. `.claude/skills/review/rules.md` を参照する（プロジェクト固有ルール）
4. 以下フォーマットで報告する

## レポートフォーマット

```
## コードレビュー結果

### :x: 要修正
- `ファイル名:行番号` — 説明と修正案

### :warning: 要検討
- `ファイル名:行番号` — 説明と改善案

### :white_check_mark: 良好
- 良かった点

### サマリー
1〜2文でまとめる
```
