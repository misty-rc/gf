//go:build !windows && !darwin

package search

import (
	"os"
	"time"
)

// birthTime は Linux 等では作成日時を標準 syscall で取得できないためゼロ値を返す。
// ゼロ値が返った場合、呼び出し側は作成日時フィルタをスキップする。
func birthTime(_ os.FileInfo) time.Time {
	return time.Time{}
}
