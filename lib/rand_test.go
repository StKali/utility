package lib

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandString(t *testing.T) {
	t.Run("test-length", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			require.Equal(t, len(RandString(i)), i)
		}
	})
	t.Run("test-randomness", func(t *testing.T) {
		counter := make(map[string]int)
		for i := 0; i < 10000; i++ {
			str := RandString(10)
			counter[str]++
		}
		for _, count := range counter {
			require.Less(t, count, 3)
		}
	})
}

func TestRandIntervalString(t *testing.T) {
	for i := 0; i < 10; i++ {
		min := rand.Intn(1024)
		max := min + rand.Intn(1024)
		str1 := RandInternalString(min, max)
		require.True(t, len(str1) >= min && len(str1) <= max)
	}

	require.Equal(t, "", RandInternalString(1, 1))
	require.Equal(t, "", RandInternalString(-1, 2))
}

func TestRandIP(t *testing.T) {

	for i := 0; i < 100; i++ {
		ip := RandIP()
		seg := strings.Split(ip, ".")
		require.Equal(t, 4, len(seg))
	}
}

var errEmailSuffixCases = []struct {
	Name   string
	Suffix string
}{
	{
		"startswith-not-@",
		"mix.com",
	},
	{
		"no-dot",
		"@com",
	},
	{
		"empty",
		"",
	},
	{
		"contains-@-not-startswith",
		"xx@163.com",
	},
	{
		"startswith-not-@-and-no-dot",
		"com",
	},
}

func TestRandEmail(t *testing.T) {
	// not set or register email suffixes
	for i := 0; i < 100; i++ {
		email := RandEmail()
		require.Contains(t, email, "@")
	}

	// set email suffixes
	require.NoError(t, SetEmailSuffix("@test.com"))
	require.True(t, strings.HasSuffix(RandEmail(), "@test.com"))
}

func TestRegisterEmailSuffix(t *testing.T) {

	// error
	for _, c := range errEmailSuffixCases {
		t.Run(c.Name, func(t *testing.T) {
			require.ErrorIs(t, RegisterEmailSuffix(c.Suffix), InvalidEmailSuffixError)
		})
	}

	oldSuffixCount := len(emailSuffixes)
	// success
	registered := []string{"@mix.com", "@add.com"}
	err := RegisterEmailSuffix(registered...)
	require.NoError(t, err)
	require.Equal(t, oldSuffixCount+len(registered), len(emailSuffixes))
	for _, suffix := range registered {
		require.Contains(t, emailSuffixes, suffix)
	}
}

func TestSetEmailSuffix(t *testing.T) {
	// error
	for _, c := range errEmailSuffixCases {
		t.Run(c.Name, func(t *testing.T) {
			require.ErrorIs(t, SetEmailSuffix(c.Suffix), InvalidEmailSuffixError)
		})
	}

	// success
	newSuffixes := []string{"@hook.com", "@stu.edu"}
	require.NoError(t, SetEmailSuffix(newSuffixes...))
	require.Equal(t, newSuffixes, emailSuffixes)
}
