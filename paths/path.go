package paths

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/stkali/utility/errors"
)

var (
	onceUserHome sync.Once
	userHome string
	
)

// UserHome return current user home path string
func UserHome() string {
	onceUserHome.Do(func() {
		var err error
		userHome, err = os.UserHomeDir()
		errors.CheckErr(err)
	})
	return userHome
}

// ToAbsPath convert any style path to posix  absolutely path
func ToAbsPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	if path[0] == '~' {
		path = UserHome() + path[1:]
	}
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		errors.Exitf(1, "failed to convert path to absolutely, err: %s", err)
	}
	return os.ExpandEnv(path)
}

// GetFileCreated get the creation time of the file through the file name.
func GetFileCreated(file string) (t time.Time, err error) {
	info, err := os.Stat(file)
	if err != nil {
		return t, errors.Newf("failed to open file: %s, err: %s", file, err)
	}
	return GetFdCreated(info), nil
}

// SplitWithExt splits a file path into three parts: the volume name (if any), the directory and filename without extension,
// and the file extension. It handles paths with and without extensions gracefully.
// It returns the volume name, the directory and filename without extension, and the file extension respectively.
// If the path does not contain an extension, the extension part will be an empty string.
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

// IsExisted checks if a file or directory exists at the given path.
// It returns true if the path exists, false otherwise.
func IsExisted(file string) bool {
	_, err := os.Stat(file)
	return err == nil || os.IsExist(err)
}
