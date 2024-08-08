package lib

import (
	"unsafe"
)

// ToString ...
func ToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// ToBytes ...
func ToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
