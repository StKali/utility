//go:build windows
package paths

import (
	"os"
	"syscall"
	"time"
)

// GetFdCreated get the creation time of the file through the fd *os.FileInfo.
func GetFdCreated(fd os.FileInfo) time.Time {
	st := fd.Sys().(*syscall.Win32FileAttributeData)
	return time.Unix(st.CreationTime.Nanoseconds()/1e9, 0)
}
