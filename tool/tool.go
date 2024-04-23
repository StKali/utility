package tool

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

const (
	_ = 1 << (iota * 10)
	KB
	MB
	GB
	TB
	PB
	EB
	// ZB
	// YB
)

var InvalidMemorySizeError = errors.New("invalid memory size string")
var errPrefix = "occurred error"

// SetErrorPrefix set prefix of CheckError output string
func SetErrorPrefix(prefix string) {
	errPrefix = prefix
}

// CheckError prints the message with the prefix and exits with error code 1
// if the message is nil, it does nothing.
func CheckError(text string, err any) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s: %s, err:%+s\n", errPrefix, text, err)
	os.Exit(1)
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

func SizeString2Number(s string) (int64, error) {

	length := len(s)
	if length == 0 {
		return 0, nil
	}

	if s[0] == '-' {
		return -1, InvalidMemorySizeError
	}

	index := strings.LastIndexFunc(s, func(r rune) bool {
		switch r {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return true
		}
		return false
	})
	if index == -1 {
		return -1, InvalidMemorySizeError
	}
	sLow := strings.ToLower(s)
	return toSize(sLow[:index+1], sLow[index+1:])
}

func toSize(s string, unit string) (int64, error) {
	base, err := strconv.ParseFloat(s, 0)
	if err != nil {
		return -1, InvalidMemorySizeError
	}
	switch unit {
	case "", "b", "byte":
		return int64(base), nil
	case "k", "kb", "kib":
		return int64(base * KB), nil
	case "m", "mb", "mib":
		return int64(base * MB), nil
	case "g", "gb", "gib":
		return int64(base * GB), nil
	case "t", "tb", "tib":
		return int64(base * TB), nil
	case "p", "pb", "pib":
		return int64(base * PB), nil
	case "e", "eb", "eib":
		return int64(base * EB), nil
	}
	return -1, InvalidMemorySizeError
}
