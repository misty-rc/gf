//go:build darwin

package search

import (
	"os"
	"syscall"
	"time"
)

// birthTime は macOS の Birthtimespec（st_birthtime）を返す。
func birthTime(info os.FileInfo) time.Time {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}
	}
	return time.Unix(st.Birthtimespec.Sec, int64(st.Birthtimespec.Nsec))
}
