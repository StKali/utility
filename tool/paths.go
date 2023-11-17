// This file is only compatible with historical versions. 
// All path-related operations will be implemented in the paths package.

package tool
import (
	"sync"
	"os"
	"path/filepath"
)

var (
	onceUserHome sync.Once
	userHome     string
)

// UserHome return current user home path string
func UserHome() string {
	onceUserHome.Do(func() {
		var err error
		userHome, err = os.UserHomeDir()
		CheckError("failed to get user home path", err)
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
		CheckError("failed to convert path to absolutely", err)
	}
	return os.ExpandEnv(path)
}
