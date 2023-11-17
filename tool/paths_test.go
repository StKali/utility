package tool

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserHome(t *testing.T) {
	require.Equal(t, homeDirectory, UserHome())
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