//go:build windows

package search

import (
	"os"
	"syscall"
	"time"
)

// birthTime は Windows の CreationTime を返す。
// os.FileInfo.Sys() は *syscall.Win32FileAttributeData を返すため追加 syscall 不要。
func birthTime(info os.FileInfo) time.Time {
	d, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return time.Time{}
	}
	return time.Unix(0, d.CreationTime.Nanoseconds())
}
