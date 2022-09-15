package tool

import (
	"github.com/stretchr/testify/require"
	"math/rand"
	"strings"
	"testing"
)

func TestRandString(t *testing.T) {
	for i := 0; i < 100; i++ {
		require.Equal(t, len(RandString(i)), i)
	}
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

func TestRandIntervalString(t *testing.T) {
	for i := 0; i < 10; i++ {
		min := rand.Intn(1024)
		max := min + rand.Intn(1024)
		str1 := RandInternalString(min, max)
		str2 := RandInternalString(max, min)
		require.True(t, len(str1) >= min && len(str1) <= max)
		require.True(t, len(str2) >= min && len(str2) <= max)
	}
}

func TestRandIP(t *testing.T) {

	for i := 0; i < 100; i++ {
		ip := RandIP()
		seg := strings.Split(ip, ".")
		require.Equal(t, 4, len(seg))
	}
}

func TestMax(t *testing.T) {
	require.Equal(t, Max(1), 1)

	// int
	maxInt := Max(1, 1010, 111)
	require.Equal(t, 1010, maxInt)

	// uint
	maxUint := Max(uint(1), uint(111), uint(100), uint(19))
	require.Equal(t, uint(111), maxUint)

	// int8
	maxInt8 := Max(int8(1), int8(100), int8(111))
	require.Equal(t, int8(111), maxInt8)

	// empty
	require.Equal(t, Max([]int{}...), 0)

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

func TestMin(t *testing.T) {

	require.Equal(t, Min(1), 1)

	// int
	minInt := Min(1, 100, 111)
	require.Equal(t, 1, minInt)

	// uint
	minUint := Min(uint(111), uint(1), uint(100), uint(111))
	require.Equal(t, uint(1), minUint)

	// int8
	minInt8 := Min(int8(122), int8(100), int8(111))
	require.Equal(t, int8(100), minInt8)
	
	// empty
	require.Equal(t, Min([]int{}...), 0)
}
