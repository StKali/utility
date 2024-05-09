package paths

import (
	"os"
	"path/filepath"
	"time"

	"github.com/stkali/utility/errors"
	"github.com/stkali/utility/tool"
)

var (
	ToAbsPath = tool.ToAbsPath
	UserHome  = tool.UserHome
)

// GetFileCreated get the creation time of the file through the file name.
func GetFileCreated(file string) (t time.Time, err error) {
	info, err := os.Stat(file)
	if err != nil {
		return t, errors.Newf("failed to open file: %s, err: %s", file, err)
	}
	return GetFdCreated(info), nil
}

// // GetFdCreated get the creation time of the file through the fd *os.FileInfo.
// func GetFdCreated(fd os.FileInfo) (time.Time, error) {
// 	st := fd.Sys().(*syscall.Stat_t)
// 	stValue := reflect.ValueOf(st).Elem()

// 	// linux
// 	cTimeValue := stValue.FieldByName("Ctim")
// 	if cTimeValue.Kind() != reflect.Invalid {
// 		timeSpec := cTimeValue.Interface().(syscall.Timespec)
// 		return time.Unix(timeSpec.Sec, timeSpec.Nsec), nil
// 	}
// 	// mac
// 	cTimeValue = stValue.FieldByName("Ctimespec")
// 	if cTimeValue.Kind() != reflect.Invalid {
// 		timeSpec := cTimeValue.Interface().(syscall.Timespec)
// 		return time.Unix(timeSpec.Sec, timeSpec.Nsec), nil
// 	}
// 	return time.Time{}, errors.Newf("failed to get file created time")
// }

func SplitWithExt(path string) (string, string, string) {
	vol := filepath.VolumeName(path)
	i := len(path) - 1
	etxIndex := -1
	for i >= len(vol) && !os.IsPathSeparator(path[i]) {
		if etxIndex == -1 && path[i] == '.' {
			etxIndex = i
		}
		i--
	}
	if etxIndex == -1 {
		return path[:i+1], path[i+1:], ""
	}
	return path[:i+1], path[i+1 : etxIndex], path[etxIndex:]
}

func IsExisted(file string) bool {
	_, err := os.Stat(file)
	return err == nil || os.IsExist(err)
}
