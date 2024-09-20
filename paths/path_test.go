package paths

import (
	"github.com/stkali/utility/lib"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
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
		{
			"abs-path",
			"/",
			"/",
		},
		{
			"env-path",
			"$HOME/hello.go",
			filepath.Join(homeDirectory, "hello.go"),
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			require.True(t, c.Expect == ToAbsPath(c.Path))
		})
	}
}

func TestOpenFile(t *testing.T) {
	testDir := t.TempDir()
	defer os.RemoveAll(testDir)

	// not existed file
	file := filepath.Join(testDir, "not-existed-file")
	fd, err := OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o777)
	require.NoError(t, err)
	defer fd.Close()

	// not existed dir
	file = filepath.Join(testDir, "not-exited-dir", "not-existed-file")
	fd, err = OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o777)
	require.NoError(t, err)
	defer fd.Close()

	// failed to create directory
	file = filepath.Join(testDir, "not-exited-dir2", "not-existed-file")
	originMakeAll := osMakeAll
	defer func() {
		osMakeAll = originMakeAll
	}()
	osMakeAll = func(path string, perm os.FileMode) error {
		return InvalidPathError
	}
	fd, err = OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o777)
	require.ErrorIs(t, err, InvalidPathError)
}

func TestAbs(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, err := Abs("")
		require.ErrorIs(t, err, InvalidPathError)
	})
}

// /var/folders/k0/nf8vpwfj4_b1y0b_mc4h35fc0000gn/T/TestClear3793610792/001/CWJRYYJmBBLd
// /var/folders/k0/nf8vpwfj4_b1y0b_mc4h35fc0000gn/T/TestClear3793610792/001/CWJRYYJmBBLd
func TestClear(t *testing.T) {

	err := Clear(lib.RandInternalString(32, 64))
	require.ErrorIs(t, err, os.ErrNotExist)

	testDir := t.TempDir()
	defer os.RemoveAll(testDir)
	err = Clear(testDir)
	require.NoError(t, err)

	// create 10 files
	for i := 0; i < 10; i++ {
		file := filepath.Join(testDir, lib.RandInternalString(10, 20))
		f, err := os.Create(file)
		require.NoError(t, err)
		text := lib.RandInternalString(10, 30)
		n, err := f.WriteString(text)
		require.NoError(t, err)
		require.Equal(t, len(text), n)
		err = f.Close()
		require.NoError(t, err)
	}
	files, err := os.ReadDir(testDir)
	require.NoError(t, err)
	require.Equal(t, 10, len(files))

	// create sub dir
	subDir := filepath.Join(testDir, "sub")
	err = os.Mkdir(subDir, 0o777)
	require.NoError(t, err)
	// create 5 files in sub dir
	for i := 0; i < 5; i++ {
		file := filepath.Join(subDir, lib.RandInternalString(10, 20))
		f, err := os.Create(file)
		require.NoError(t, err)
		text := lib.RandInternalString(10, 30)
		n, err := f.WriteString(text)
		require.NoError(t, err)
		require.Equal(t, len(text), n)
		err = f.Close()
		require.NoError(t, err)
	}
	files, err = os.ReadDir(subDir)
	require.NoError(t, err)
	require.Equal(t, 5, len(files))

	err = Clear(testDir)
	require.NoError(t, err)
	files, err = os.ReadDir(testDir)
	require.NoError(t, err)
	require.Equal(t, 0, len(files))
}
