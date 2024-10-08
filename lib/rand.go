package lib

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// RandString returns a random string of length n
func RandString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
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

// RandInternalString returns a random string of length between min and max, consisting
// of visible ASCII characters only
func RandInternalString(min, max int) string {
	if min < 0 || min >= max {
		return ""
	}
	n := min + rand.Intn(max-min)
	return RandString(n)
}

var InvalidEmailSuffixError = errors.New("invalid email suffix, must be startswith '@' and contains '.'")

// RegisterEmailSuffix add an item to the list of email suffixes for generating email addresses
func RegisterEmailSuffix(suffix ...string) error {
	if err := validateEmailAddressSuffix(suffix...); err != nil {
		return err
	}
	emailSuffixes = append(emailSuffixes, suffix...)
	return nil
}

// SetEmailSuffix set up a list of email suffixes for generating email addresses
func SetEmailSuffix(suffix ...string) error {
	if err := validateEmailAddressSuffix(suffix...); err != nil {
		return err
	}
	emailSuffixes = suffix
	return nil
}

// validateEmailAddress validate whether the email suffix is valid
func validateEmailAddressSuffix(suffix ...string) error {
	for _, suf := range suffix {
		if suf == "" || suf[0] != '@' || !strings.Contains(suf, ".") {
			return InvalidEmailSuffixError
		}
	}
	return nil
}

var emailSuffixes []string
var defaultEmailSuffixes = []string{
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
	if emailSuffixes == nil {
		return prefix + defaultEmailSuffixes[len(prefix)%len(defaultEmailSuffixes)]
	} else {
		return prefix + emailSuffixes[len(prefix)%len(emailSuffixes)]
	}
}

// RandIP returns a random IPv4 address, which may be either private or public
func RandIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Int31n(255), rand.Int31n(255), rand.Int31n(255), rand.Int31n(255))
}
