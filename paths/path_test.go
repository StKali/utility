package paths

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSplitWithExt(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		folder   string
		filename string
		extend   string
	}{
		{
			"empty",
			"",
			"",
			"",
			"",
		},
		{
			"no extend",
			"hello",
			"",
			"hello",
			"",
		},
		{
			"with extend",
			"file.txt",
			"",
			"file",
			".txt",
		},
		{
			"existed point but no extend",
			"file.",
			"",
			"file",
			".",
		},
		{
			"full filepath",
			"/users/home/project.log",
			"/users/home/",
			"project",
			".log",
		},
		{
			"only extend",
			".log",
			"",
			"",
			".log",
		},
		{
			"multi point",
			"file/test.tar.gz",
			"file/",
			"test.tar",
			".gz",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			folder, filename, etx := SplitWithExt(c.path)
			require.Equal(t, c.folder, folder)
			require.Equal(t, c.filename, filename)
			require.Equal(t, c.extend, etx)
		})
	}
}

func TestGetFileCreated(t *testing.T) {

	testFile := filepath.Join(t.TempDir(), "testfile")
	// get not existed file created time
	_, err := GetFileCreated(testFile)
	require.ErrorIs(t, err, os.ErrNotExist)

	preTime := time.Now().Add(-100 * time.Millisecond)
	f, err := os.Create(testFile)
	require.NoError(t, err)
	defer f.Close()
	postTime := time.Now().Add(+100 * time.Millisecond)

	created, err := GetFileCreated(testFile)
	require.NoError(t, err)
	require.True(t, preTime.Before(created))
	require.True(t, created.Before(postTime))
}

func TestIsExisted(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "testfile")
	// get not existed file created time
	require.False(t, IsExisted(testFile))
	f, err := os.Create(testFile)
	require.NoError(t, err)
	defer f.Close()
	require.True(t, IsExisted(testFile))
}

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
			"home",
			"~",
			homeDirectory,
		},
		{
			"relative-home",
			"~/hello.go",
			filepath.Join(homeDirectory, "hello.go"),
		},
	}

	for _, c := range cases {
		require.True(t, c.Expect == ToAbsPath(c.Path))
	}
}
