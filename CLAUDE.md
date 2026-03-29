# gf — プロジェクト概要

Go 製の高速ファイル検索 CLI。ファイル名・パス・属性で探すツール。
メタデータ検索（EXIF等）はスコープ外。パフォーマンス優先（find より速いことが目標）。

## ビルド・テスト

```bash
go build -o gf.exe .   # ビルド（スキル: .claude/skills/build/SKILL.md）
go test ./search/       # テスト
```

## 詳細仕様

- [アーキテクチャ](.claude/docs/specs/architecture.md)
- [フラグ・パターン・出力仕様](.claude/docs/specs/flags.md)
- [パフォーマンス設計](.claude/docs/specs/performance.md)

## Compact Instructions

コンパクティング時は以下を保持すること：
- 各ファイルの役割と実装済み機能
- アーキテクチャ上の決定とその理由
- 未解決の課題・次のステップ
