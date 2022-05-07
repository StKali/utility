package tool

import (
	"fmt"
	"math/rand"
	"time"
)

type Number interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64
}

// Max returns the max value in values
// panic when no parameters are given
func Max[T Number](values ...T) T {
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// Min returns the min value in values
// panic when no parameters are given
func Min[T Number](values ...T) T {
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

// RandString returns a random string of length n
func RandString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}

// RandInternalString returns a random string of length between min and max, consisting of visible ASCII characters only
func RandInternalString(min, max int) string {
	if min > max {
		max, min = min, max
	}
	n := min + rand.Intn(max-min)
	return RandString(n)
}

var emailSuffixes = []string{
	"@mock_google.com",
	"@mock_outlook.com",
	"@mock_yahoo.com",
	"@mock_apple.com",
	"@mock_163.com",
	"@mock_qq.com",
	"@mock_babiq.com",
}

// RandEmail returns a random email address
func RandEmail() string {
	prefix := RandInternalString(4, 32)
	return prefix + emailSuffixes[len(prefix)%len(emailSuffixes)]
}

// RandIP returns a random IPv4 address, which may be either private or public
func RandIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Int31n(255), rand.Int31n(255), rand.Int31n(255), rand.Int31n(255))
}
