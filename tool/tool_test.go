package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	currentDirectory string
	relativeFile     string
	homeDirectory    string
)

func TestMain(m *testing.M) {
	var err error
	currentDirectory, err = os.Getwd()
	if err != nil {
		CheckError("failed to get current directory", err)
	}
	relativeFile, err = filepath.Abs("./../Makefile")
	if err != nil {
		CheckError("failed to get relative file", err)
	}
	homeDirectory, err = os.UserHomeDir()
	if err != nil {
		CheckError("failed to get home directory", err)
	}
	os.Exit(m.Run())
}

func Test2String(t *testing.T) {
	cases := []struct {
		Name   string
		Bytes  []byte
		Expect string
	}{
		{
			"empty",
			[]byte{},
			"",
		},
		{
			"integer",
			[]byte("1"),
			"1",
		},

		{
			"return",
			[]byte("\r"),
			"\r",
		},

		{
			"newline",
			[]byte("\n"),
			"\n",
		},
		{
			"other",
			[]byte("\r\n928176\tasljh\tt"),
			"\r\n928176\tasljh\tt",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			require.Equal(t, c.Expect, ToString(c.Bytes))
		})
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			require.Equal(t, c.Bytes, ToBytes(c.Expect))
		})
	}
}

func TestSetErrorPrefix(t *testing.T) {

	cases := []struct {
		Name   string
		Prefix string
	}{
		{
			"empty-string",
			"",
		},
		{
			"general-string",
			"Error",
		},
		{
			"contain-space-string",
			"meet error",
		},
		{
			"contain-line-string",
			"meet_error",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			old := errPrefix
			SetErrorPrefix(c.Prefix)
			require.NotEqual(t, c.Prefix, old)
		})
	}
}

func TestSizeString2Byte(t *testing.T) {
	cases := []struct {
		name   string
		size   string
		expect int64
		err    error
	}{
		{
			"empty",
			"",
			0,
			nil,
		},
		{
			"0b",
			"0b",
			0,
			nil,
		},
		{
			"0k",
			"0k",
			0,
			nil,
		},
		{
			"0G",
			"0G",
			0,
			nil,
		},
		{
			"1k",
			"1k",
			1024,
			nil,
		},
		{
			"1.2k",
			"1.2k",
			1228,
			nil,
		},
		{
			"5.5Kib",
			"5.5Kib",
			5632,
			nil,
		},
		{
			"100.01mb",
			"100.01mb",
			104868085,
			nil,
		},
		{
			"1tb",
			"1tb",
			1 << 40,
			nil,
		},
		{
			"1pb",
			"1pb",
			1 << 50,
			nil,
		},
		{
			"1eb",
			"1eb",
			1 << 60,
			nil,
		},
		{
			"xxx",
			"xxx",
			-1,
			InvalidMemorySizeError,
		},
		{
			"11111KKK",
			"111kkk",
			-1,
			InvalidMemorySizeError,
		},
		{
			"-1k",
			"-1k",
			-1,
			InvalidMemorySizeError,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := SizeString2Number(c.size)
			if c.err == nil {
				require.NoError(t, err)
				require.Equal(t, c.expect, actual)
			} else {
				require.Equal(t, c.expect, actual)
				require.Equal(t, err, c.err)
			}
		})
	}
}
