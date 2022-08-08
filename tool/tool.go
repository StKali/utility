package tool

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"unsafe"
)

// CheckError prints the message with the prefix and exits with error code 1
// if the message is nil, it does nothing.
func CheckError(text string, err any) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "occurred error: %s, err:%+s\n", text, err)
	os.Exit(1)
}

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

// ToString allows zero-copy conversion of a bytes to a string.
func ToString(b []byte) string {
	var s string
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&s))
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	stringHeader.Data = sliceHeader.Data
	stringHeader.Len = sliceHeader.Len
	return s
}

// ToBytes allows zero-copy conversion of a string to a byte array,
// but it is unsafe as the resulting bytes must not be modified or unpredictable issues may occur.
func ToBytes(s string) []byte {
	var b []byte
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&s))
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sliceHeader.Data = stringHeader.Data
	sliceHeader.Cap = stringHeader.Len
	sliceHeader.Len = stringHeader.Len
	return b
}
