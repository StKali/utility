package paths

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stkali/utility/errors"
)

var InvalidPathError = errors.Error("invalid path error")

var (
	onceUserHome sync.Once
	userHome     string
	// for test
	makeAll = os.MkdirAll
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
	path, err := abs(path)
	errors.CheckErr(err)
	return path
}

var MustAbs = ToAbsPath

func abs(path string) (string, error) {
	switch path {
	case "":
		return "", InvalidPathError
	case "~":
		return UserHome(), nil
	case ".":
		return os.Getwd()
	}

	path = filepath.Clean(path)
	if strings.HasPrefix(path, "~/") {
		path = UserHome() + path[1:]
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	path = os.ExpandEnv(path)
	return filepath.Abs(path)
}

// Abs(path) returns the absolute path of the given path.
// If failed to convert path to absolutely, it returns an error.
func Abs(path string) (string, error) {
	return abs(path)
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

// OpenFile attempts to create or open a file with the specified name, flags, and permissions.
// If the file's directory does not exist, it attempts to create the directory with 0755 permissions.
func OpenFile(file string, flag int, perm os.FileMode) (fd *os.File, err error) {
	fd, err = os.OpenFile(file, flag, perm)
	if err != nil {
		if os.IsNotExist(err) {
			directory := filepath.Dir(file)
			err = makeAll(directory, os.ModePerm)
			if err != nil {
				return nil, errors.Newf("failed to create directory: %q, err: %s", directory, err)
			}
			return os.OpenFile(file, flag, perm)
		}
	}
	return fd, err
}
