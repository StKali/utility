//go:build linux

package paths

import (
	"os"
	"syscall"
	"time"
)

// GetFdCreated get the creation time of the file through the fd *os.FileInfo.
func GetFdCreated(fd os.FileInfo) time.Time {
	st := fd.Sys().(*syscall.Stat_t)
	return time.Unix(st.Ctim.Sec, st.Ctim.Nsec)
}
