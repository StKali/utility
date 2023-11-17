package log

import (
	stderr "errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stkali/utility/errors"
	"github.com/stkali/utility/paths"
	"github.com/stkali/utility/tool"
	"github.com/stretchr/testify/require"
)

func TestBaseMakeRotateFile(t *testing.T) {
	testDir := t.TempDir()
	defer os.RemoveAll(testDir)
	testFile := filepath.Join(testDir, "test.rot")

	f, err := NewDurationRotateFile(testFile, defaultDuration)
	require.NoError(t, err)

	_, err = os.Create(testFile)
	require.NoError(t, err)
	f.folder = testFile
	err = f.makeRotateFile(filepath.Join(testFile, "mock.rot"))
	require.Error(t, err)
	errors.Is(err, os.ErrExist)
	f.folder = testDir

	noPermDir := filepath.Join(testDir, "test")
	require.NoError(t, os.MkdirAll(noPermDir, 0o000))
	err = f.makeRotateFile(filepath.Join(noPermDir, "file.rot"))
	require.Error(t, err)
}

func TestBaseRotateFileGetBackupFiles(t *testing.T) {
	defer os.RemoveAll(t.TempDir())
	noPermDir := filepath.Join(t.TempDir(), "noPermDir")
	err := os.MkdirAll(noPermDir, 0o000)
	require.NoError(t, err)
	sf := DefaultSizeRotateFile()
	sf.folder = noPermDir
	_, err = sf.getBackupFiles()
	require.True(t, errors.Is(err, os.ErrPermission))
}

func TestBaseRotateFileClean(t *testing.T) {
	tmp := t.TempDir()
	defer os.RemoveAll(tmp)
	noPermDir := filepath.Join(tmp, "noPermDir")
	err := os.MkdirAll(noPermDir, 0o000)
	require.NoError(t, err)

	// no clean
	b := baseRotateFile{backups: 0, age: 0, folder: noPermDir}
	err = b.clean()
	require.NoError(t, err)

	// cannot getBackupFiles
	b.age = 1
	err = b.clean()
	require.True(t, errors.Is(err, os.ErrPermission))

	b.age = 0
	b.backups = 0
	b.block = false
	err = b.clean()
	require.NoError(t, err)
}

func TestBaseRotateFileCleanByBackups(t *testing.T) {
	testDir := t.TempDir()
	defer os.RemoveAll(testDir)
	sf := DefaultSizeRotateFile()
	sf.folder = testDir
	count := 4 + rand.Intn(8)
	require.NoError(t, sf.SetBackups(count))

	n, err := io.WriteString(sf, "x")
	require.Equal(t, 1, n)
	require.NoError(t, err)
	for i := 0; i < count*2; i++ {
		err = sf.Rotate(true)
		io.WriteString(sf, "xxxxx")
		require.NoError(t, err)
	}
	fs, err := sf.getBackupFiles()
	require.NoError(t, err)
	require.Equal(t, count, len(fs))
}

func TestBaseRotateFileCleanByAge(t *testing.T) {
	errors.DisableWarning()
	testDir := t.TempDir()
	defer os.RemoveAll(testDir)

	rotatingFile := filepath.Join(testDir, "rotating.rot")
	sf, err := NewSizeRotateFile(rotatingFile, defaultSize)
	require.NoError(t, err)

	// 写入第一测试基本的写入功能是ok的，主要的是文件只有在调用写入接口时才会被创建。
	n, err := io.WriteString(sf, "xxx")
	require.Equal(t, 3, n)
	require.NoError(t, err)

	randBackups := 5 + rand.Intn(5)
	for i := 0; i < randBackups; i++ {
		err = sf.Rotate(true)
		require.NoError(t, err)
	}

	require.NoError(t, err)
	fs, err := sf.getBackupFiles()
	require.NoError(t, err)
	require.Equal(t, randBackups, len(fs))

	err = sf.SetAge(time.Second)
	time.Sleep(time.Second)
	require.NoError(t, err)
	for i := 0; i < randBackups; i++ {
		err = sf.Rotate(true)
		require.NoError(t, err)
	}
	fs, err = sf.getBackupFiles()
	require.NoError(t, err)
	require.Equal(t, randBackups, len(fs))

	sf.SetAge(1)
	sf.clean()
	fs, err = sf.getBackupFiles()
	require.NoError(t, err)
	require.Equal(t, 0, len(fs))
}

func TestBaseRotateFileFilename(t *testing.T) {
	cases := []struct {
		name   string
		folder string
		fName  string
		ext    string
		expect string
	}{
		{
			"empty",
			"",
			"",
			"",
			"",
		},
		{
			"no folder",
			"",
			"test",
			".log",
			"test.log",
		},
		{
			"no folder and name",
			"",
			"",
			".log",
			".log",
		},
		{
			"no ext",
			"folder",
			"name",
			"",
			filepath.Join("folder", "name"),
		},
		{
			"relative folder",
			"./hello/log",
			"rotating",
			".log",
			"./hello/log/rotating.log",
		},
		{
			"abs folder",
			"/home/user/hello/log",
			"rotating",
			".log",
			"/home/user/hello/log/rotating.log",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := baseRotateFile{
				folder: c.folder,
				name:   c.fName,
				ext:    c.ext,
			}
			require.Equal(t, c.expect, f.filename())
		})
	}
}

func TestBaseDropBackupsFiles(t *testing.T) {

	testDir := t.TempDir()
	fmt.Println(testDir)
	defer os.RemoveAll(testDir)
	fileCount := 5 + rand.Intn(5)
	f := newBaseRotateFile()
	f.folder = testDir
	now := time.Now()
	for i := 0; i < fileCount; i++ {
		now = now.Add(time.Second)
		file := filepath.Join(f.folder, f.backupName(now))
		_, err := os.OpenFile(file, os.O_CREATE, defaultModePerm)
		if os.IsExist(err) {
			continue
		}
		require.NoError(t, err)
	}
	fs, err := os.ReadDir(testDir)
	require.NoError(t, err)
	require.Equal(t, fileCount, len(fs))
	err = f.DropRotateFiles()
	require.Error(t, err)

	_, err = os.OpenFile(f.filename(), os.O_CREATE, defaultModePerm)
	require.NoError(t, err)
	err = f.DropRotateFiles()
	require.NoError(t, err)

	os.Chmod(testDir, 0o000)
	err = f.DropRotateFiles()
	require.Error(t, err)
}

func TestInterfaceMethods(t *testing.T) {

	df := DefaultDurationRotateFile()
	df.fd = &MockWriteCloserError{}
	sf := DefaultSizeRotateFile()
	sf.fd = &MockWriteCloserError{}

	rotaters := []RotateFiler{df, sf}
	for _, rotater := range rotaters {

		// SetBackuopTimeFotmat
		err := rotater.SetBackupTimeFormat("")
		require.Error(t, err)
		err = rotater.SetBackupTimeFormat("2006-01-02")
		require.NoError(t, err)
		err = rotater.SetBackupTimeFormat("150405")
		require.NoError(t, err)
		require.Equal(t, "150405", rotater.BackupTimeFormat())

		// SetAge
		require.NoError(t, rotater.SetAge(0))
		require.Equal(t, rotater.Age(), time.Duration(0))
		require.NoError(t, rotater.SetAge(1))
		require.Equal(t, rotater.Age(), time.Duration(1))
		require.Error(t, rotater.SetAge(-1))
		require.Equal(t, rotater.Age(), time.Duration(1))

		// SetBackups
		require.NoError(t, rotater.SetBackups(0))
		require.Equal(t, rotater.Backups(), 0)
		require.NoError(t, rotater.SetBackups(10))
		require.Equal(t, rotater.Backups(), 10)
		require.Error(t, rotater.SetBackups(-1))
		require.Equal(t, rotater.Backups(), 10)

		// Close
		require.Error(t, rotater.Close())
	}
}

func TestBaseRotateFileClose(t *testing.T) {

	f := newBaseRotateFile()
	f.folder = t.TempDir()
	f.makeRotateFile(f.filename())
	err := os.Remove(f.filename())
	require.NoError(t, err)
	n, err := io.WriteString(f.fd, "test")
	require.Equal(t, 4, n)
	require.NoError(t, err)
	err = f.close()
	require.NoError(t, err)
}

func TestUseLeftoverFile(t *testing.T) {

	f := newBaseRotateFile()
	f.folder = t.TempDir()

	// failed to open leftover file
	require.False(t, paths.IsExisted(f.filename()))
	err := f.useLeftoverFile(f.filename())
	require.ErrorIs(t, err, os.ErrNotExist)
	
	// create leftover file
	fd, err := os.OpenFile(f.filename(), os.O_CREATE|os.O_WRONLY, defaultModePerm)
	require.NoError(t, err)
	defer f.close()
	leftLength := 10
	text := tool.RandString(leftLength)
	io.WriteString(fd, text)
	err = fd.Close()
	require.NoError(t, err)

	err = f.useLeftoverFile(f.filename())
	require.NoError(t, err)
	fd = f.fd.(*os.File)
	st, err := fd.Stat()

	require.NoError(t, err)
	require.Equal(t, int64(leftLength), st.Size())
	err = f.close()
	require.NoError(t, err)

	err = os.Chmod(f.filename(), 0o000)
	require.NoError(t, err)

	err = f.useLeftoverFile(f.filename())
	require.ErrorIs(t, err, os.ErrPermission)
}

func TestNewDurationRotateFile(t *testing.T) {

	errors.DisableWarning()
	// invalid duration
	_, err := NewDurationRotateFile("test.rot", -1)
	require.Error(t, err)
	require.Equal(t, InvalidDurationError, err)

	// duration = 0
	_, err = NewDurationRotateFile("test.rot", 0)
	require.NoError(t, err)

	_, err = NewDurationRotateFile(".", 0)
	require.Equal(t, InvalidRotateFileError, err)
}

type MockWriteCloserError struct {
}

func (m *MockWriteCloserError) Close() error {
	return stderr.New("mock close error")
}

func (m *MockWriteCloserError) Write(b []byte) (int, error) {
	return 0, nil
}

func TestDurationRotateFileSetDuration(t *testing.T) {
	f, err := NewDurationRotateFile(filepath.Join(t.TempDir(), "test.rot"), 12*time.Hour)
	require.NoError(t, err)
	err = f.SetDuration(time.Minute * 30)
	require.NoError(t, err)

	err = f.SetDuration(time.Hour)
	require.NoError(t, err)

	err = f.SetDuration(-1)
	require.Equal(t, InvalidDurationError, err)
}

func TestDurationRotateFileSetTimer(t *testing.T) {
	f, err := NewDurationRotateFile(filepath.Join(t.TempDir(), "test.rot"), 12*time.Hour)
	require.NoError(t, err)
	require.ErrorIs(t, f.setTimer(0), InvalidDurationError)

	f.timer = nil
	require.NoError(t, f.setTimer(time.Hour))
	require.NoError(t, f.setTimer(2 * time.Hour))
}

func TestDurationRotateFileWrite(t *testing.T) {

	testFile := filepath.Join(t.TempDir(), "test.rot")
	//DurationRotateFile
	f, err := NewDurationRotateFile(testFile, defaultDuration)
	require.NoError(t, err)

	for i := 0; i < 100; i++ {
		text := tool.RandInternalString(0, 1<<10)
		n, err := io.WriteString(f, text)
		require.Equal(t, n, len(text))
		require.NoError(t, err)
	}
	//require.NoError(t, f.DropRotateFiles())
}

func TestDurationRotateFileRotate(t *testing.T) {

	folder := t.TempDir()
	filename := filepath.Join(folder, "test.rot")
	f, err := NewDurationRotateFile(filename, time.Hour*24)
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)

	err = f.Rotate(false)
	require.NoError(t, err)

	count := 5 + rand.Intn(10)
	for i := 0; i < count; i++ {
		_, err = io.WriteString(f, "1")
		require.NoError(t, err)
		require.NoError(t, f.Rotate(true))
	}
	fs, err := f.getBackupFiles()
	require.NoError(t, err)
	require.Equal(t, count, len(fs))
	require.NoError(t, f.DropRotateFiles())
}

func TestDurationRotateFileClean(t *testing.T) {
	folder := t.TempDir()
	rotating := filepath.Join(folder, "test.rotate")
	f, err := NewDurationRotateFile(rotating, time.Hour)
	require.NoError(t, err)
	_, err = io.WriteString(f, "1")
	require.NoError(t, err)
}

func TestDurationRotateFileMontRotateFile(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "test.rot")
	f, err := NewDurationRotateFile(testFile, time.Hour)
	require.NoError(t, err)
	length := 50 + rand.Intn(100)
	text := tool.RandString(length)
	n, err := io.WriteString(f, text)
	require.NoError(t, err)
	require.Equal(t, length, n)

	err = f.Close()
	require.NoError(t, err)

	n, err = io.WriteString(f, text)
	require.Equal(t, length, n)
	require.NoError(t, err)

}

func TestNewSizeRotateFile(t *testing.T) {

	testDir := t.TempDir()
	defer os.RemoveAll(testDir)
	testFile := filepath.Join(testDir, "test.rot")
	_, err := NewSizeRotateFile(testFile, defaultSize)
	require.NoError(t, err)

	_, err = NewSizeRotateFile(testFile, 0)
	require.Error(t, err)

	_, err = NewSizeRotateFile(testFile, -1)
	require.Error(t, err)

}

func TestSizeRotateFileSetSize(t *testing.T) {
	
	testFile := filepath.Join(t.TempDir(), "test.rot")

	f, err := NewSizeRotateFile(testFile, defaultSize)
	require.NoError(t, err)

	// invalid size
	require.ErrorIs(t, f.SetSize(-1), InvalidSizeError)

	// samll size
	require.NoError(t, f.SetSize(1000))
}

func TestSizeRotateFileWrite(t *testing.T) {

	testDir := t.TempDir()
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.rot")
	sf, err := NewSizeRotateFile(testFile, 8)
	require.NoError(t, err)

	n, err := io.WriteString(sf, "01234")
	require.NoError(t, err)
	require.Equal(t, 5, n)

	// write big content
	n, err = io.WriteString(sf, "0123456789")
	require.Error(t, err)
	require.Equal(t, 0, n)

	// active rotate
	n, err = io.WriteString(sf, "0123456")
	require.Equal(t, 7, n)
	require.NoError(t, err)
}

func TestMontSizeRotateFile(t *testing.T) {

	testFile := filepath.Join(t.TempDir(), "test.rot")
	fd, err := os.Create(testFile)
	require.NoError(t, err)

	// create leftoevr rotate file
	leftLength := 10
	n, err := io.WriteString(fd, tool.RandString(leftLength))
	require.Equal(t, leftLength, n)
	require.NoError(t, err)

	sf, err := NewSizeRotateFile(testFile, 18)
	require.Equal(t, sf.filename(), testFile)

	require.NoError(t, err)
	n, err = io.WriteString(sf, tool.RandString(leftLength))
	require.Equal(t, leftLength, n)
	require.NoError(t, err)

}

func BenchmarkBaseRotateFileFilename(b *testing.B) {
	f := DefaultSizeRotateFile()
	for i := 0; i < b.N; i++ {
		f.filename()
	}
}

func BenchmarkBaseRotateFileBackupFile(b *testing.B) {
	file := DefaultDurationRotateFile()
	for i := 0; i < b.N; i++ {
		file.backupFile()
	}
}

func BenchmarkValidateTimeFormat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		validateTimeFormat(defaultBackupTimeFormat)
	}
}

func TestValidateTimeFormat(t *testing.T) {
	cases := []struct{
		name string
		format string
		expect bool

	}{
		{
			"empty",
			"",
			false,
		},
		{
			"year",
			"2006",
			true,
		},
		{
			"month",
			"01",
			true,
		},
		{
			"day",
			"02",
			true,
		},
		{
			"hour-24",
			"15",
			true,
		},
		{
			"hour-12",
			"03",
			true,
		},
		{
			"mine",
			"04",
			true,
		},
		{
			"second",
			"05",
			true,
		},
		{
			"no-number",
			"xx",
			false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.expect, validateTimeFormat(c.format))
		})
	}
}