package lib

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unsafe"
)

const (
	// Byte is the size of a byte in bytes.
	_        = iota
	KB int64 = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB

	// Time constants.
	Day   = 24 * time.Hour
	Month = 30 * Day
	Year  = 12 * Month
)

// ToString converts a byte slice to a string.
// The string is not copied, but the underlying memory is shared.
func ToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// ToBytes converts a string to a byte slice.
// The string is not copied, but the underlying memory is shared.
func ToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// Size2String converts a size in bytes to a string in the format of "1024" or "1024 KB" or "1024 MB" or "1024 GB" or
// "1024 TB" or "1024 PB" or "1024 EB".
// The unit is chosen automatically based on the size.
// If the size is too large to be represented in the largest unit, it is rounded to the nearest multiple of the largest unit.
// If the size is negative or zero, it is returned as "0 B".
func Size2String(size int64) (string, error) {
	switch {
	case size < 0:
		return "", fmt.Errorf("size is negative: %d", size)
	case size < KB:
		return fmt.Sprintf("%d B", size), nil
	case size < MB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB)), nil
	case size < GB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB)), nil
	case size < TB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB)), nil
	case size < PB:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(TB)), nil
	case size < EB:
		return fmt.Sprintf("%.2f PB", float64(size)/float64(PB)), nil
	default:
		return fmt.Sprintf("%.2f EB", float64(size)/float64(EB)), nil
	}
}

// String2Size converts a string to a size in bytes.
// The string should be in the format of "1024" or "1024 KB" or "1024 MB" or "1024 GB" or "1024 TB" or "1024 PB"
// or "1024 EB".
// The unit can be "KB", "MB", "GB", "TB", "PB", "EB", "K", "M", "G", "T", "P", "E", "KiB", "MiB", "GiB", "TiB",
// "PiB", "EiB", or empty.
// If the unit is empty, it is assumed to be "B".
// If the string is invalid, an error is returned.
func String2Size(size string) (ret int64, err error) {
	if size == "" {
		return 0, nil
	}
	if size[0] == '-' {
		return 0, fmt.Errorf("size cannot be negative: %s", size)
	}
	index := strings.IndexFunc(size, func(r rune) bool {
		return !unicode.IsNumber(r) && r != '.'
	})
	if index == -1 {
		index = len(size)
	}
	unit := strings.TrimSpace(size[index:])
	value := strings.TrimSpace(size[:index])
	power := int64(1)
	switch strings.ToLower(unit) {
	case "", "byte":
	case "kb", "k", "kib":
		power = KB
	case "mb", "m", "mib":
		power = MB
	case "gb", "g", "gib":
		power = GB
	case "tb", "t", "tib":
		power = TB
	case "pb", "p", "pib":
		power = PB
	case "eb", "e", "eib":
		power = EB
	default:
		return 0, fmt.Errorf("invalid size: %s", size)
	}
	fret, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size: %s", size)
	}
	return int64(fret * float64(power)), nil
}
