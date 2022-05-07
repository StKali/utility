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

func TestToAbsPath(t *testing.T) {

	cases := []struct {
		Name   string
		Path   string
		Expect string
	}{
		{
			"current-file",
			"tool.go",
			filepath.Join(currentDirectory, "tool.go"),
		},
		{
			"current-directory",
			".",
			currentDirectory,
		},
		{
			"relative",
			"./../Makefile",
			relativeFile,
		},
		{
			"home",
			"~",
			homeDirectory,
		},
		{
			"relative-home",
			"~/hello.go",
			filepath.Join(homeDirectory, "hello.go"),
		},
		{
			"absolute",
			homeDirectory,
			homeDirectory,
		},
	}

	for _, c := range cases {
		require.Equal(t, c.Expect, ToAbsPath(c.Path))
	}
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

func TestUserHome(t *testing.T) {
	require.Equal(t, homeDirectory, UserHome())
}
