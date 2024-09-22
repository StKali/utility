package rotate

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/stkali/utility/errors"
	"github.com/stkali/utility/lib"
	"github.com/stkali/utility/paths"
	"github.com/stretchr/testify/require"
)

//go:generate mockgen -package rotate -destination mock_WriteCloser_test.go io WriteCloser
//go:generate mockgen -package rotate -destination mock_DirEntry_test.go os DirEntry

// -·-·-·-·-·-·--·-·-·-·-
//
//	UNIT TEST
//
// -·-·-·-·-·-·--·-·-·-·-
func TestDeleteBackupFiles(t *testing.T) {

	folder := t.TempDir()
	defer os.RemoveAll(folder)

	t.Run("delete existed file", func(t *testing.T) {
		absFile := filepath.Join(folder, lib.RandString(6))
		file, err := os.Create(absFile)
		require.NoError(t, err)
		err = file.Close()
		require.True(t, paths.IsExisted(absFile))
		require.NoError(t, err)
		buf := &bytes.Buffer{}
		deleteBackupFiles([]backupFile{{file: absFile}})
		errors.SetWarningOutput(buf)
		warningText := buf.String()
		require.True(t, len(warningText) == 0)
	})

	t.Run("delete not existed file", func(t *testing.T) {
		buf := &bytes.Buffer{}
		errors.SetWarningOutput(buf)
		deleteBackupFiles([]backupFile{{file: lib.RandString(8)}, {file: lib.RandString(8)}})
		require.Contains(t, buf.String(), "failed to remove")
	})
}

func TestCompressFile(t *testing.T) {
	folder := t.TempDir()
	defer os.RemoveAll(folder)
	srcFile := filepath.Join(folder, lib.RandString(6))
	f, err := os.Create(srcFile)
	content := lib.RandInternalString(128, 1024)
	n, err := f.WriteString(content)
	require.NoError(t, err)
	require.Equal(t, len(content), int(n))
	err = f.Close()
	require.NoError(t, err)

	t.Run("successfully compress file", func(t *testing.T) {
		dstFile := srcFile + ".gz"
		require.NoError(t, err)
		err = compressFile(srcFile, dstFile, 6)
		require.NoError(t, err)
		require.False(t, paths.IsExisted(srcFile))
		fd, err := os.Open(dstFile)
		require.NoError(t, err)
		defer fd.Close()
		reader, err := gzip.NewReader(fd)
		require.NoError(t, err)
		defer reader.Close()
		data, err := io.ReadAll(reader)
		require.Equal(t, content, lib.ToString(data))
	})

	t.Run("failed to compress file", func(t *testing.T) {

		// not exist src file
		buf := &bytes.Buffer{}
		errors.SetWarningOutput(buf)
		defer errors.SetWarningOutput(os.Stderr)
		err := compressFile("not-existed-file", "not-existed-file.gz", 6)
		require.NoError(t, err)
		require.Contains(t, buf.String(), "no such file or directory")

		// cannot get file stat
		osOpen = func(name string) (*os.File, error) {
			return nil, nil
		}
		err = compressFile(srcFile, srcFile+".gz", 6)
		require.ErrorIs(t, err, os.ErrInvalid)
		osOpen = os.Open

		// cannot create dst file
		srcFile := filepath.Join(folder, lib.RandString(6))
		f, err := os.Create(srcFile)
		f.WriteString(lib.RandString(10))
		require.NoError(t, err)
		err = f.Close()
		require.NoError(t, err)
		dstDir := filepath.Join(folder, lib.RandString(6))
		err = os.Mkdir(dstDir, 0o000)
		require.NoError(t, err)
		err = compressFile(srcFile, filepath.Join(dstDir, "not-existed-file.gz"), 6)
		require.ErrorIs(t, err, os.ErrPermission)

		// invalid compression level
		err = compressFile(srcFile, filepath.Join(folder, "not-existed-file.gz"), 10)
		require.Errorf(t, err, "invalid compression level:")

		// copy error
		ioCopy = func(dst io.Writer, src io.Reader) (written int64, err error) {
			return 0, io.ErrUnexpectedEOF
		}
		err = compressFile(srcFile, filepath.Join(folder, "not-existed-file.gz"), 6)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
		ioCopy = io.Copy
	})
}

func TestBackupFileString(t *testing.T) {
	file := "/user/home/stkali/test.log"
	bf := backupFile{
		file:    file,
		modTime: time.Time{},
	}
	require.Contains(t, bf.String(), fmt.Sprintf("backupFile(%s created at ", file))
}

func TestRotatingFileString(t *testing.T) {
	testDir := t.TempDir()
	defer os.RemoveAll(testDir)
	testFile := filepath.Join(testDir, lib.RandString(6))
	f, err := NewRotatingFile(testFile, nil)
	require.NoError(t, err)
	defer f.Close()
	require.Equal(t, fmt.Sprintf("RotatingFile(%s)", f.filename), f.String())
}

func TestRotatingWriteString(t *testing.T) {

	t.Run("successfully call `WriteString` and `Write`", func(t *testing.T) {
		testDir := t.TempDir()
		defer os.RemoveAll(testDir)
		testFile := filepath.Join(testDir, lib.RandString(6))
		f, err := NewRotatingFile(testFile)
		require.NoError(t, err)
		defer f.Close()
		n, err := f.WriteString("hello")
		require.Equal(t, 5, n)
		require.NoError(t, err)
		n, err = f.Write(nil)
		require.Equal(t, 0, n)
		require.NoError(t, err)
		n, err = f.Write([]byte("world"))
		require.Equal(t, 5, n)
		require.NoError(t, err)
	})

	t.Run("write string failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		w := NewMockWriteCloser(ctrl)
		retErr := errors.Error("write string failed")
		w.EXPECT().Write(gomock.Any()).Return(0, retErr)
		f, err := NewRotatingFile("test", nil)
		require.NoError(t, err)
		f.writer = w
		n, err := f.WriteString("hello")
		require.Equal(t, 0, n)
		require.ErrorIs(t, err, retErr)
	})

	t.Run("write failed", func(t *testing.T) {
		testDir := t.TempDir()
		defer os.RemoveAll(testDir)
		testFile := filepath.Join(testDir, lib.RandString(6))
		f, err := NewRotatingFile(testFile, WithMaxSize(lib.GB), WithDuration(lib.Day))
		require.NoError(t, err)
		defer f.Close()

		// failed to create file
		osOpenFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
			return nil, os.ErrInvalid
		}
		n, err := f.Write([]byte(lib.RandString(10)))
		require.Equal(t, 0, n)
		require.ErrorIs(t, err, os.ErrInvalid)
		osOpenFile = os.OpenFile

		// failed to make directory
		testFile = filepath.Join(testDir, lib.RandString(6), lib.RandString(6))
		f, err = NewRotatingFile(testFile)
		require.NoError(t, err)
		osMkdirAll = func(path string, perm os.FileMode) error {
			return os.ErrPermission
		}
		n, err = f.WriteString(lib.RandString(10))
		require.Equal(t, 0, n)
		require.ErrorIs(t, err, os.ErrPermission)
		osMkdirAll = os.MkdirAll

		// failed to get file stat
		osOpenFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
			return nil, nil
		}
		n, err = f.WriteString(lib.RandString(10))
		require.Equal(t, 0, n)
		require.ErrorIs(t, err, os.ErrInvalid)
		osOpenFile = os.OpenFile

		// failed to rotate file
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		w := NewMockWriteCloser(ctrl)
		w.EXPECT().Write(gomock.Any()).Return(15, nil)
		w.EXPECT().Close().Return(os.ErrClosed)

		f.writer = w
		f.option.MaxSize = 10
		n, err = f.WriteString(lib.RandString(15))
		require.Equal(t, 0, n)
		require.ErrorIs(t, err, os.ErrClosed)

	})
}

func TestRotatingFileModePerm(t *testing.T) {
	testDir := t.TempDir()
	defer os.RemoveAll(testDir)
	testFile := filepath.Join(testDir, lib.RandString(6))
	f, err := NewRotatingFile(testFile, WithModePerm(0o777))
	require.NoError(t, err)
	defer f.Close()
}

func TestClose(t *testing.T) {

	t.Run("size", func(t *testing.T) {
		testDir := t.TempDir()
		defer os.RemoveAll(testDir)
		testFile := filepath.Join(testDir, lib.RandString(6))
		f, err := NewRotatingFile(testFile, WithMaxSize(10), WithDuration(-1))
		require.NoError(t, err)
		require.Nil(t, f.timer)
		require.Equal(t, int64(0), f.used)
		err = f.Close()
		require.NoError(t, err)
		require.Nil(t, f.timer)
		require.Nil(t, f.writer)
	})
	t.Run("duration", func(t *testing.T) {
		testDir := t.TempDir()
		defer os.RemoveAll(testDir)
		testFile := filepath.Join(testDir, lib.RandString(6))
		f, err := NewRotatingFile(testFile, WithMaxSize(-1), WithDuration(lib.Day))
		require.NoError(t, err)
		require.NotNil(t, f.timer)
		require.Equal(t, int64(0), f.used)
		err = f.Close()
		require.NoError(t, err)
		require.Nil(t, f.writer)
	})
	t.Run("failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		recorder := NewMockWriteCloser(ctrl)
		err := fmt.Errorf("close error")
		recorder.EXPECT().Close().Return(err)
		file := RotatingFile{
			writer: recorder,
			option: defaultOption.clone(),
		}
		wrapperErr := file.Close()
		require.Error(t, err)
		require.ErrorIs(t, wrapperErr, err)
	})
}

func TestRotatingFileCleanBackups(t *testing.T) {
	testDir := t.TempDir()
	defer os.RemoveAll(testDir)
	testFile := filepath.Join(testDir, lib.RandString(6))
	f, err := NewRotatingFile(testFile, WithMaxSize(10), WithDuration(-1))
	require.NoError(t, err)
	defer f.Close()

	t.Run("cannot read directory", func(t *testing.T) {
		osReadDir = func(name string) ([]os.DirEntry, error) {
			return nil, os.ErrInvalid
		}
		defer func() {
			osReadDir = os.ReadDir
		}()
		_, err = f.cleanBackups()
		require.ErrorIs(t, err, os.ErrInvalid)
	})

	t.Run("cannot get file stat", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		entry := NewMockDirEntry(ctrl)
		bkFilename := f.nextBackupFilename()
		entry.EXPECT().Name().Return(bkFilename)
		entry.EXPECT().IsDir().Return(false)
		entry.EXPECT().Info().Return(nil, os.ErrInvalid)

		osReadDir = func(name string) ([]os.DirEntry, error) {
			return []os.DirEntry{entry}, nil
		}
		defer func() {
			osReadDir = os.ReadDir
		}()
		_, err = f.cleanBackups()
		require.ErrorIs(t, err, os.ErrInvalid)
	})

	t.Run("clean by max age", func(t *testing.T) {
		// delete all backups by max age
		err = paths.Clear(f.folder)
		require.NoError(t, err)
		for i := 0; i < 5; i++ {
			file, err := os.Create(filepath.Join(f.folder, f.nextBackupFilename()))
			require.NoError(t, err)
			err = file.Close()
			require.NoError(t, err)
		}
		fs, err := f.sortBackups()
		require.NoError(t, err)
		require.Equal(t, 5, len(fs))
		err = f.Close()
		require.NoError(t, err)

		// set max age to 100ms
		f.option.MaxAge = 100 * time.Millisecond
		time.Sleep(1000 * time.Millisecond)
		bks, err := f.cleanBackups()
		require.NoError(t, err)
		require.Equal(t, 0, len(bks))
	})

}

func TestRotatingFileRotate(t *testing.T) {

	testDir := t.TempDir()
	defer os.RemoveAll(testDir)
	testFile := filepath.Join(testDir, lib.RandString(6))
	f, err := NewRotatingFile(testFile)
	require.NoError(t, err)
	defer f.Close()

	//not found src file
	osRename = func(oldpath, newpath string) error {
		return os.ErrNotExist
	}
	buf := &bytes.Buffer{}
	errors.SetWarningOutput(buf)
	//defer errors.SetWarningOutput(os.Stderr)
	err = f.rotate()
	require.NoError(t, err)
	require.Contains(t, buf.String(), "failed to backup file")
	osRename = os.Rename
	errors.SetWarningOutput(os.Stderr)

	// failed to rename (unknown error)
	osRename = func(oldpath, newpath string) error {
		return os.ErrInvalid
	}
	err = f.rotate()
	require.ErrorIs(t, err, os.ErrInvalid)
	osRename = os.Rename

	// failed to create new file
	osOpenFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		return nil, os.ErrPermission
	}
	err = f.rotate()
	require.ErrorIs(t, err, os.ErrPermission)
	osOpenFile = os.OpenFile

}

func TestRotatingFileOpenWriter(t *testing.T) {
	testDir := t.TempDir()
	defer os.RemoveAll(testDir)
	testFile := filepath.Join(testDir, lib.RandString(6))

	// failed to create file

	// failed to get file stat

	//used > maxSize

	// failed rotate
	fd, err := os.Create(testFile)
	require.NoError(t, err)
	n, err := fd.WriteString(lib.RandString(64))
	require.NoError(t, err)
	require.Equal(t, 64, n)
	err = fd.Close()
	require.NoError(t, err)

	f, err := NewRotatingFile(testFile, WithMaxSize(10), WithDuration(-1))
	require.NoError(t, err)
	defer f.Close()
	osRename = func(oldpath, newpath string) error {
		return os.ErrInvalid
	}
	defer func() {
		osRename = os.Rename
	}()
	n, err = f.Write(nil)
	require.Equal(t, 0, n)
	require.ErrorIs(t, err, os.ErrInvalid)

}

func TestNewRotatingFile(t *testing.T) {
	testDir := t.TempDir()
	defer os.RemoveAll(testDir)
	t.Run("default", func(t *testing.T) {
		filename := lib.RandString(6)
		testFile := filepath.Join(testDir, "not-existed-dir", filename)
		f, err := NewRotatingFile(testFile)
		require.NoError(t, err)
		defer f.Close()

		require.Equal(t, testFile, f.file)
		require.Equal(t, filename, f.filename)
		require.Equal(t, defaultOption.MaxSize, f.option.MaxSize)
		require.Equal(t, defaultOption.Duration, f.option.Duration)
		require.Equal(t, defaultOption.Backups, f.option.Backups)
		require.Equal(t, int64(0), f.used)
		n, err := io.WriteString(f, "hello")
		require.NoError(t, err)
		require.Equal(t, 5, n)
	})
	t.Run("empty option", func(t *testing.T) {
		testFile := filepath.Join(testDir, lib.RandString(6))
		f, err := NewRotatingFile(testFile)
		require.True(t, err == nil)
		defer f.Close()
		require.Equal(t, f.file, testFile)
	})
	t.Run("duration rotate", func(t *testing.T) {
		testFile := filepath.Join(testDir, lib.RandString(6))
		f, err := NewRotatingFile(testFile, WithDuration(lib.Day))
		require.NoError(t, err)
		defer f.Close()
		require.Equal(t, testFile, f.file)
	})
	t.Run("prefix", func(t *testing.T) {
		testFile := filepath.Join(testDir, lib.RandString(6))
		// invalid chars
		f, err := NewRotatingFile(testFile, WithBackupPrefix("!"))
		require.ErrorContains(t, err, "backup prefix contains invalid character")
		require.Nil(t, f)
		// too long prefix
		f, err = NewRotatingFile(testFile, WithBackupPrefix(lib.RandString(130)))
		require.ErrorIs(t, err, InvalidBackupPrefixError)
		require.Nil(t, f)
		// success
		f, err = NewRotatingFile(testFile, WithBackupPrefix("test-"))
		require.NoError(t, err)
		require.Equal(t, "test-", f.option.BackupPrefix)
	})
	t.Run("no specify file", func(t *testing.T) {
		f, err := NewRotatingFile("", nil)
		require.ErrorIs(t, err, paths.InvalidPathError)
		require.Nil(t, f)
	})
	t.Run("no compress level", func(t *testing.T) {
		f, err := NewRotatingFile(filepath.Join(testDir, lib.RandString(6)), WithCompressLevel(-1))
		require.NoError(t, err)
		require.NotNil(t, f)
		require.Equal(t, -1, f.option.CompressLevel)
	})

	t.Run("invalid compress level", func(t *testing.T) {
		f, err := NewRotatingFile(filepath.Join(testDir, lib.RandString(6)), WithCompressLevel(11))
		require.ErrorIs(t, err, InvalidCompressionLevelError)
		require.Nil(t, f)
	})

	t.Run("invalid mode perm", func(t *testing.T) {
		f, err := NewRotatingFile(filepath.Join(testDir, lib.RandString(6)), WithModePerm(0o001))
		require.ErrorIs(t, err, ModePermissionError)
		require.Nil(t, f)
	})

	t.Run("not limit backups", func(t *testing.T) {
		buf := &bytes.Buffer{}
		errors.SetWarningOutput(buf)
		defer errors.SetWarningOutput(os.Stderr)
		f, err := NewRotatingFile(filepath.Join(testDir, lib.RandString(6)), WithBackups(-1))
		require.NoError(t, err)
		require.Equal(t, -1, f.option.Backups)
		require.Contains(t, buf.String(), "not limited by backups")
	})
}

// -·-·-·-·-·-·--·-·-·-·-
//
//	BENCHMARK TEST
//
// -·-·-·-·-·-·--·-·-·-·-
func BenchmarkWrite(b *testing.B) {
	testDir := b.TempDir()
	defer os.RemoveAll(testDir)

	b.Run("size mode", func(b *testing.B) {
		testFile := filepath.Join(testDir, "size_rotate")
		f, err := NewRotatingFile(testFile, WithMaxSize(lib.EB), WithDuration(-1))
		require.NoError(b, err)
		defer f.Close()
		n, err := f.WriteString("hello world!\n")
		require.Equal(b, 13, n)
		require.NoError(b, err)
		for i := 0; i < b.N; i++ {
			n, err := f.WriteString("hello world!\n")
			require.Equal(b, 13, n)
			require.NoError(b, err)
		}
	})

	b.Run("duration mode", func(b *testing.B) {
		testFile := filepath.Join(testDir, "duration_rotate")
		f, err := NewRotatingFile(testFile, WithMaxSize(-1), WithDuration(lib.Day))
		require.NoError(b, err)
		defer f.Close()
		for i := 0; i < b.N; i++ {
			n, err := f.WriteString("hello world!\n")
			require.Equal(b, 13, n)
			require.NoError(b, err)
		}
	})

	b.Run("multi mode", func(b *testing.B) {
		testFile := filepath.Join(testDir, "multi_rotate")
		f, err := NewRotatingFile(testFile, WithMaxSize(lib.EB), WithDuration(lib.Day))
		require.NoError(b, err)
		defer f.Close()
		for i := 0; i < b.N; i++ {
			n, err := f.WriteString("hello world!\n")
			require.Equal(b, 13, n)
			require.NoError(b, err)
		}
	})

	b.Run("file system mode", func(b *testing.B) {
		testFile := filepath.Join(testDir, "fs_rotate")
		f, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY, 0644)
		require.NoError(b, err)
		defer f.Close()
		for i := 0; i < b.N; i++ {
			n, err := f.WriteString("hello world!\n")
			require.Equal(b, 13, n)
			require.NoError(b, err)
		}
	})
}

// -·-·-·-·-·-·--·-·-·-·-
//
//	LOGICAL TEST
//
// -·-·-·-·-·-·--·-·-·-·-
func TestLogicTidyBackups(t *testing.T) {

	t.Run("max age = 0", func(t *testing.T) {
		testDir := t.TempDir()
		defer os.RemoveAll(testDir)
		testFile := filepath.Join(testDir, "clean_rotate")
		f, err := NewRotatingFile(testFile, WithMaxAge(0))
		require.NoError(t, err)
		defer f.Close()
		require.True(t, f.option.MaxSize != 0)
		for i := 0; i < 10; i++ {
			n, err := f.WriteString("hello go")
			require.Equal(t, 8, n)
			require.NoError(t, err)
		}
		files, err := f.sortBackups()
		require.NoError(t, err)
		f.Close()
		require.Equal(t, 0, len(files))
	})

	t.Run("max backups = 0", func(t *testing.T) {
		testDir := t.TempDir()
		defer os.RemoveAll(testDir)

		testFile := filepath.Join(testDir, "clean_rotate")
		f, err := NewRotatingFile(testFile, WithMaxSize(10), WithBackups(0), WithMaxAge(-1))
		require.NoError(t, err)
		defer f.Close()

		for i := 0; i < 10; i++ {
			n, err := f.WriteString("hello go")
			require.Equal(t, 8, n)
			require.NoError(t, err)
		}

		files, err := f.sortBackups()
		require.NoError(t, err)
		f.Close()
		require.Equal(t, 0, len(files))
	})

	t.Run("no backups", func(t *testing.T) {
		testDir := t.TempDir()
		defer os.RemoveAll(testDir)

		testFile := filepath.Join(testDir, "clean_rotate")
		f, err := NewRotatingFile(testFile, WithMaxSize(10), WithBackups(0), WithMaxAge(-1))
		require.NoError(t, err)
		defer f.Close()
		f.tidyBackups()
		files, err := f.sortBackups()
		require.NoError(t, err)
		require.Equal(t, 0, len(files))
	})

	t.Run("max age and max backups and compress backups", func(t *testing.T) {
		testDir := t.TempDir()
		defer os.RemoveAll(testDir)
		testFile := filepath.Join(testDir, "clean_rotate")
		duration := 500 * time.Millisecond
		f, err := NewRotatingFile(testFile, WithMaxSize(10), WithMaxAge(duration), WithBackups(2))
		require.NoError(t, err)
		for i := 0; i < 10; i++ {
			n, err := f.WriteString("hello go")
			require.Equal(t, 8, n)
			require.NoError(t, err)
		}
		// wait for rotate
		time.Sleep(duration + 200*time.Millisecond)
		err = f.Close()
		require.NoError(t, err)
		// ensure all backups has been compressed
		f.tidyBackups()
		err = f.Close()
		require.NoError(t, err)
		files, err := f.sortBackups()
		require.NoError(t, err)
		require.True(t, len(files) <= 2)
	})

	t.Run("compress backups", func(t *testing.T) {

		testDir := t.TempDir()
		defer os.RemoveAll(testDir)
		testFile := filepath.Join(testDir, "clean_rotate")
		f, err := NewRotatingFile(testFile, WithMaxSize(10), WithCompressLevel(9))
		require.NoError(t, err)
		require.NotNil(t, f)
		number := rand.Intn(10)
		for i := 0; i < number; i++ {
			content := lib.RandInternalString(64, 128)
			n, err := f.WriteString(content)
			require.Equal(t, len(content), n)
			require.NoError(t, err)
		}
		// wait for rotate
		err = f.Close()
		require.NoError(t, err)

		// ensure all backups has been compressed
		f.tidyBackups()
		err = f.Close()
		require.NoError(t, err)
		files, err := f.sortBackups()
		require.NoError(t, err)
		require.Equal(t, number, len(files))
		for index := range files {
			require.True(t, strings.HasSuffix(files[index].file, compressExtension))
		}
	})

	t.Run("disable compress backups", func(t *testing.T) {
		testDir := t.TempDir()
		defer os.RemoveAll(testDir)
		testFile := filepath.Join(testDir, "clean_rotate")
		f, err := NewRotatingFile(testFile, WithMaxSize(10), WithCompressLevel(0))
		require.NoError(t, err)
		require.NotNil(t, f)
		number := rand.Intn(10)
		for i := 0; i < number; i++ {
			content := lib.RandInternalString(64, 128)
			n, err := f.WriteString(content)
			require.Equal(t, len(content), n)
			require.NoError(t, err)
		}
		// wait for rotate
		err = f.Close()
		require.NoError(t, err)
		files, err := f.sortBackups()
		require.NoError(t, err)
		require.Equal(t, number, len(files))
		for index := range files {
			require.False(t, strings.HasSuffix(files[index].file, compressExtension))
		}
	})
}

func TestLogicCompress(t *testing.T) {

	testDir := t.TempDir()
	defer os.RemoveAll(testDir)
	testFile := filepath.Join(testDir, "compress_rotate")
	f, err := NewRotatingFile(testFile, WithCompressLevel(6))
	require.NoError(t, err)
	defer f.Close()
	// TODO: add compress test
}

func TestLogicRotate(t *testing.T) {

	// test size rotate
	t.Run("size rotate", func(t *testing.T) {

		testDir := t.TempDir()
		defer os.RemoveAll(testDir)
		testFile := filepath.Join(testDir, "size_rotate.txt")
		f, err := NewRotatingFile(testFile, WithMaxSize(10), WithDuration(0))
		require.NoError(t, err)
		// ensure config is correct
		require.Nil(t, f.timer)
		require.True(t, f.rotatingTime.IsZero())
		require.Equal(t, int64(10), f.option.MaxSize)
		require.Equal(t, int64(0), f.used)

		n, err := f.WriteString(lib.RandString(15))
		require.NoError(t, err)
		require.Equal(t, 15, n)
		require.Equal(t, int64(0), f.used)
		err = f.Close()
		require.NoError(t, err)
		files, err := f.sortBackups()
		require.NoError(t, err)
		require.Equal(t, 1, len(files))
	})

	// test duration rotate
	t.Run("duration rotate", func(t *testing.T) {
		testDir := t.TempDir()
		defer os.RemoveAll(testDir)
		testFile := filepath.Join(testDir, "duration_rotate.txt")
		duration := 1000 * time.Millisecond
		f, err := NewRotatingFile(testFile, WithMaxSize(0), WithDuration(duration))
		require.NoError(t, err)

		// ensure config is correct
		require.NotNil(t, f.timer)
		require.True(t, f.rotatingTime.IsZero())
		require.Equal(t, int64(0), f.used)
		require.Equal(t, int64(0), f.option.MaxSize)

		// writer is nil, so cannot rotate.
		time.Sleep(time.Duration(float64(duration) * 1.5))
		err = f.Close()
		require.NoError(t, err)
		files, err := f.sortBackups()
		require.NoError(t, err)
		require.Equal(t, 0, len(files))

		// ensure backup file is created
		f.timer.Reset(duration)
		n, err := f.WriteString(lib.RandString(15))
		require.NoError(t, err)
		require.Equal(t, 15, n)
		require.Equal(t, int64(0), f.used)
		time.Sleep(time.Duration(float64(duration) * 1.5))
		err = f.Close()
		files, err = f.sortBackups()
		require.NoError(t, err)
		require.Equal(t, 1, len(files))
	})

	// test multi rotate
	t.Run("multi rotate", func(t *testing.T) {
		testDir := t.TempDir()
		defer os.RemoveAll(testDir)
		duration := 1000 * time.Millisecond
		testFile := filepath.Join(testDir, "multi_rotate.txt")
		f, err := NewRotatingFile(
			testFile,
			WithMaxSize(20),
			WithDuration(duration),
		)
		require.NoError(t, err)
		// ensure config is correct
		require.NotNil(t, f.timer)
		require.True(t, f.rotatingTime.IsZero())
		require.Equal(t, int64(0), f.used)
		require.Equal(t, int64(20), f.option.MaxSize)
		require.Equal(t, duration, f.option.Duration)

		// writer is nil, so cannot rotate.
		time.Sleep(time.Duration(float64(duration) * 1.5))
		err = f.Close()
		require.NoError(t, err)
		files, err := f.sortBackups()
		require.NoError(t, err)
		require.Equal(t, 0, len(files))

		// ensure backup file is created by duration rotate
		f.timer.Reset(duration)
		require.True(t, f.rotatingTime.IsZero())
		n, err := f.WriteString(lib.RandString(15))
		require.NoError(t, err)
		require.Equal(t, 15, n)
		require.Equal(t, int64(15), f.used)
		time.Sleep(time.Duration(float64(duration) * 1.5))
		err = f.Close()
		require.False(t, f.rotatingTime.IsZero())
		files, err = f.sortBackups()
		require.NoError(t, err)
		require.Equal(t, 1, len(files))
		durationRotateTime := f.rotatingTime

		// ensure backup file is created by size rotate
		n, err = f.WriteString(lib.RandString(25))
		require.NoError(t, err)
		require.Equal(t, 25, n)

		// ensure not reached max size rotate
		require.Equal(t, int64(0), f.used)
		require.True(t, f.rotatingTime.After(durationRotateTime))
		err = f.Close()
		require.NoError(t, err)

	})
}

func TestLogicNewRotatingFile(t *testing.T) {
	testDir := t.TempDir()
	defer os.RemoveAll(testDir)
	// use left file
	t.Run("use left file", func(t *testing.T) {
		testFile := filepath.Join(testDir, "left_rotate.txt")
		f, err := os.Create(testFile)
		require.NoError(t, err)
		n, err := f.WriteString(lib.RandString(64))
		require.NoError(t, err)
		require.Equal(t, 64, n)
		err = f.Close()
		require.NoError(t, err)

		rf, err := NewRotatingFile(testFile, WithMaxSize(32), WithDuration(0))
		require.NoError(t, err)
		n, err = rf.Write(nil)
		require.NoError(t, err)
		require.Equal(t, 0, n)
		require.Equal(t, int64(0), rf.used)
		err = rf.Close()
		require.NoError(t, err)

	})
}
