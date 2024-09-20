// Copyright 2021-2024 The utility Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in the
// LICENSE file

// Package rotate provides a rotating file writer that can be used to write data to.
// It implements the io.WriteCloser interface and WriteString method.

package rotate

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/stkali/utility/errors"
	"github.com/stkali/utility/lib"
	"github.com/stkali/utility/paths"
)

const (
	writeMode         = 0o200
	saltWidth         = 8
	compressExtension = ".gz"
)

var (
	// define errors for the package.
	ModePermissionError          = errors.Error("invalid mode permission")
	InvalidBackupPrefixError     = errors.Error("invalid backup prefix")
	InvalidCompressionLevelError = errors.Error("invalid compression level")

	// for testing, we override the default functions used by the package.
	osOpen     = os.Open
	osOpenFile = os.OpenFile
	osRemove   = os.Remove
	osRename   = os.Rename
	osReadDir  = os.ReadDir
	osMkdirAll = os.MkdirAll
	ioCopy     = io.Copy
)

// Option is a configuration option for rotating files. default is `defaultOption`
type Option struct {

	// MaxSize(default: 1 GB) defines the threshold size (in bytes) that triggers a
	// file rotation.
	// <= 0 means no rotation based on file size.
	MaxSize int64

	// Duration(default: 1 day) specifies the time interval after which a new file
	// should be created.
	// <= 0 means no rotation based on time interval.
	// NOTE:
	//   The timer is updated every time rotating, and this includes MaxSize-based updates
	Duration time.Duration

	// MaxAge(default: 30 days) defines the maximum age that a backup file can have before
	// it is considered for cleanup.
	// Files older than this duration will be deleted during the cleanup goroutine.
	// = 0 means no backup files are retained.
	// < 0 the backup deletion strategy based on `MaxAge` will not work.
	MaxAge time.Duration

	// ModePerm(default: 0o644) specifies the default file permission bits used when
	// creating new rotating files.
	ModePerm os.FileMode

	// Backups(default: 30) defines the maximum number of backup files that can be
	// retained after rotation. When this limit is reached, the oldest backup file
	// will be deleted to make space for new ones.
	// = 0 means no backup files are retained.
	// < 0 the backup deletion strategy based on `Backups` will not work.
	Backups int

	// CompressLevel(default: 6) specifies the compression level used when compressing
	// rotating files.
	// <= 0 means no compression.
	CompressLevel int

	// BackupFilePrefix specifies the time format used when creating backup files.
	BackupPrefix string
}

var defaultOption = &Option{
	Duration:     lib.Day,
	MaxSize:      lib.GB,
	Backups:      30,
	MaxAge:       lib.Month,
	ModePerm:     0o644,
	BackupPrefix: "rotating-",
	// Available compression levels are 1-9, 9 is highest compression.
	// I think 6 is a good compromise between speed and compression ratio.
	CompressLevel: 6,
}

// clone returns a copy of the Option.
func (o *Option) clone() *Option {
	cp := *o
	return &cp
}

type backupFile struct {
	// modTime is the modification time of the backup file.
	modTime time.Time
	// file is abs path of the backup file.
	file string
}

// String implements the Stringer interface for backupFile.
func (b backupFile) String() string {
	return fmt.Sprintf("backupFile(%s created at %s)", b.file, b.modTime)
}

// deleteFile deletes the specified file.
// It prints a warning if the deletion fails.
func deleteFile(file string) {
	err := osRemove(file)
	if err != nil {
		errors.Warningf("failed to remove file %q, err: %s", file, err)
	}
}

// deleteBackupFiles deletes the specified backup files.
// It prints a warning if any deletion fails.
func deleteBackupFiles(files []backupFile) {
	for index := range files {
		deleteFile(files[index].file)
	}
}

// compressFile uses gzip to compress the specified file and delete the original file.
// If compression or deletion fails, it prints a warning and retains the source file
// as much as possible
func compressFile(src, dst string, level int) (err error) {

	f, err := osOpen(src)
	if err != nil {
		errors.Warningf("failed to read source file %q, err: %s", src, err)
		return nil
	}

	defer func() {
		f.Close()
		// if no error occurred, delete source file
		if err == nil {
			deleteFile(src)
		}
	}()

	info, err := f.Stat()
	if err != nil {
		return errors.Newf("failed to get backup file %q info, err: %s", src, err)
	}

	// os.O_TRUNC ensure file is truncated before writing to it.
	gzipFile, err := osOpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
	if err != nil {
		return errors.Newf("failed to open compressed backup file %q, err: %s", src, err)
	}

	defer gzipFile.Close()

	writer, err := gzip.NewWriterLevel(gzipFile, level)
	if err != nil {
		return errors.Newf("failed to create gzip level writer: %s", err)
	}

	defer writer.Close()

	if _, err = ioCopy(writer, f); err != nil {
		return errors.Newf("failed to compress rotating file %q, err: %s", src, err)
	}

	return err
}

// RotatingFile is a rotating file that can be used to write data to.
// It implements the io.Writer interface.
type RotatingFile struct {
	// writer is the current file descriptor (io.Writer) that is being written to.
	// It is created on the first write, and the call `Close` closes and is set to nil.
	writer io.Writer

	// option contains the configuration options for the rotating file.
	option *Option

	// mtx to protect the security of concurrent writes.
	mtx sync.Mutex

	// used keeps track of the amount of space (in bytes) that has been used in
	// the  current rotating file. This value increases as rotating data is written,
	// and triggers rotate when the size threshold is reached, after which it is
	// set back to 0
	used int64

	// file is the abs path of the current rotating file.
	file string
	// folder is the abs path of the folder where the rotating files are stored.
	folder string
	// filename is the name of the rotating file with extension.
	filename string

	// timer is the timer that triggers the rotating rotation based on the duration interval.
	// It is reset when a new rotating file is created.
	timer        *time.Timer
	rotatingTime time.Time

	// cleaning (using an underscore prefix to avoid accidental use as a public field)
	// is an atomic.Bool that indicates whether a garbage collection (cleanup) task
	// is currently being executed.
	cleaning atomic.Bool
}

// String implements the Stringer interface for RotatingFile.
func (r *RotatingFile) String() string {
	return fmt.Sprintf("RotatingFile(%s)", r.filename)
}

// Write writes the specified data to the rotating file.
// It returns the number of bytes written and an error if any.
//
// NOTE:
//
//	When rotating files based on the `MaxSize` threshold, the decision to rotate
//
// is made after writing occurs, rather than before. This can lead to the issue where
// files may become slightly larger than the set `MaxSize`. However, the benefit of
// this approach is that it ensures at least one write operation is allowed to complete.
//
//	In scenarios where a single write exceeds the `MaxSize` threshold, continuous file
//
// rotation would potentially block program execution due to inability to write further.
// To prevent this, returning an error to halt such writes could be implemented, but in
// practical applications, we often prefer not to do so. Therefore, our implementation
// allows for at least one such oversized write to proceed, even if it exceeds the threshold.
func (r *RotatingFile) Write(b []byte) (int, error) {

	r.mtx.Lock()
	defer r.mtx.Unlock()
	// ensure the writer is open
	if r.writer == nil {
		if err := r.openWriter(); err != nil {
			return 0, err
		}
	}
	n, err := r.writer.Write(b)
	if err != nil {
		return n, errors.Newf("failed to write %s to file: %s, err: %s",
			lib.ToString(b), r.filename, err)
	}
	// update used space if MaxSize is set
	if r.option.MaxSize > 0 {
		r.used += int64(n)
		if r.used > r.option.MaxSize {
			if err = r.rotate(); err != nil {
				return 0, err
			}
		}
	}
	return n, nil
}

// WriteString writes the specified string to the rotating file.
func (r *RotatingFile) WriteString(s string) (int, error) {
	return r.Write(lib.ToBytes(s))
}

// Close implements the io.Closer interface.
// It closes the rotating file and releases any associated resources.
func (r *RotatingFile) Close() error {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	// close the current writer
	err := r.close()
	if err != nil {
		return err
	}
	// wait for the cleanup goroutine to finish
	for r.cleaning.Load() {
	}
	return nil
}

// close the rotating file if writer implements the io.Closer interface.
// Updates writer, used, and timer.
func (r *RotatingFile) close() error {
	if closer, ok := r.writer.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return errors.Newf("failed to close writer: %s, err: %s", r.writer, err)
		}
	}
	r.writer = nil
	r.used = 0
	if r.timer != nil {
		r.timer.Stop()
	}
	return nil
}

// openWriter opens a new rotating file for writing.
// It will create the folder if it does not exist.
// If the file already exists, it will be opened for appending.
func (r *RotatingFile) openWriter() error {

	writer, err := r.createFile(r.file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, r.option.ModePerm)
	if err != nil {
		return errors.Newf("failed to open rotating file: %q, err: %s", r.file, err)
	}
	// update used space if MaxSize is set
	if r.option.MaxSize > 0 {
		var info os.FileInfo
		info, err = writer.Stat()
		if err != nil {
			return errors.Newf("failed to stat rotating file: %q, err: %s", r.file, err)
		}
		r.used = info.Size()
		// determines whether the left file meets the rotation condition
		if r.used > r.option.MaxSize {
			if err = r.rotate(); err != nil {
				return err
			}
		}
	}
	r.writer = writer
	return nil
}

// createFile creates a new file with the specified name and permission bits.
// It creates the folder if it does not exist.
func (r *RotatingFile) createFile(file string, flag int, perm os.FileMode) (fd *os.File, err error) {
	fd, err = osOpenFile(file, flag, perm)
	if err != nil {
		if os.IsNotExist(err) {
			err = osMkdirAll(r.folder, os.ModePerm)
			if err != nil {
				return nil, errors.Newf("failed to create rotating folder: %s, err: %s", r.folder, err)
			}
			return osOpenFile(file, flag, perm)
		}
	}
	return fd, err
}

// rotate closes the current file descriptor and creates a new rotated file.
// It also attempts to clean up and compress the backups files asynchronously.
func (r *RotatingFile) rotate() error {
	err := r.close()
	if err != nil {
		return errors.Newf("failed to close file: %s, err: %s", r.file, err)
	}
	// when both Backups and MaxAge are not equal to 0, a new file is created.
	if r.option.Backups != 0 && r.option.MaxAge != 0 {
		backupFile := filepath.Join(r.folder, r.nextBackupFilename())
		err = osRename(r.file, backupFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				errors.Warningf("failed to backup file: %q, err: %s", r.file, err)
			} else {
				return errors.Newf("failed to backup file: %q, err: %s", backupFile, err)
			}
		}
		// cleanup expired backups and compress backup files
		r.tidyBackups()
	}
	// ensure the file is truncated before writing to it.
	fd, err := r.createFile(r.file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, r.option.ModePerm)
	if err != nil {
		return errors.Newf("failed to open rotating file: %s", err)
	}
	r.writer = fd
	// update rotatingTime and reset timer if used time-based rotation is enabled
	if r.option.Duration > 0 {
		r.rotatingTime = time.Now()
		r.timer.Reset(r.option.Duration)
	}
	if r.option.MaxSize > 0 {
		r.used = 0
	}
	return nil
}

// nextBackupFilename returns the name of the next backup file.
func (r *RotatingFile) nextBackupFilename() string {
	sb := &strings.Builder{}
	sb.Grow(len(r.option.BackupPrefix) + saltWidth + 1 + len(r.filename))
	sb.WriteString(r.option.BackupPrefix)
	text := lib.RandString(saltWidth)
	sb.WriteString(text)
	sb.WriteByte('-')
	sb.WriteString(r.filename)
	return sb.String()
}

// tidyBackups deletes the expired backups and compresses backup files
func (r *RotatingFile) tidyBackups() {
	// existed a running cleanup goroutine
	if !r.cleaning.CompareAndSwap(false, true) {
		return
	}
	// start a cleanup goroutine to delete the expired backups
	go func() {
		defer r.cleaning.Store(false)
		bks, err := r.cleanBackups()
		errors.Warning(err)
		// compress backup files if compressLevel > 0
		if r.option.CompressLevel <= 0 {
			return
		}
		for _, bk := range bks {
			// avoid compressed file
			if !strings.HasSuffix(bk.file, compressExtension) {
				errors.Warning(compressFile(
					bk.file,
					bk.file+compressExtension,
					r.option.CompressLevel))
			}
		}
	}()
}

// cleanBackups performs garbage collection (cleanup) of old backup files.
// It deletes the oldest backup files until the maximum number of backup files is reached.
func (r *RotatingFile) cleanBackups() ([]backupFile, error) {

	backups, err := r.sortBackups()
	if err != nil {
		return nil, err
	}

	length := len(backups)
	if length == 0 {
		return nil, nil
	}

	deleteIndex := 0
	// calculate the index of the oldest backup file to delete based on Backups
	if r.option.Backups > 0 {
		if left := length - r.option.Backups; left > 0 {
			deleteIndex = left
		}
	}

	// calculate the index of the oldest backup file to delete based on MaxAge
	if r.option.MaxAge > 0 {
		expired := time.Now().Add(-r.option.MaxAge)
		index := slices.IndexFunc(backups, func(backup backupFile) bool {
			return expired.Equal(backup.modTime) || expired.Before(backup.modTime)
		})
		if index == -1 {
			deleteIndex = length
		} else {
			deleteIndex = lib.Max(index, deleteIndex)
		}
	}
	if deleteIndex > 0 {
		deleteBackupFiles(backups[:deleteIndex])
	}
	return backups[deleteIndex:], nil
}

// sortBackups returns a list of backup files sorted by modification time.
func (r *RotatingFile) sortBackups() ([]backupFile, error) {
	files, err := osReadDir(r.folder)
	if err != nil {
		return nil, errors.Newf("failed to list backup files, err: %s", err)
	}
	backups := make([]backupFile, 0, len(files))
	var info os.FileInfo
	for index := range files {
		name := files[index].Name()

		if files[index].IsDir() ||
			!strings.HasPrefix(name, r.option.BackupPrefix) ||
			// backup file and compressed file
			!(strings.HasSuffix(name, r.filename) || strings.HasSuffix(name, r.filename+compressExtension)) {
			continue
		}
		info, err = files[index].Info()
		if err != nil {
			return nil, errors.Newf("failed to get file: %q, err: %s", name, err)
		}
		bk := backupFile{
			file:    filepath.Join(r.folder, name),
			modTime: info.ModTime(),
		}
		backups = append(backups, bk)
	}
	// sort backups by modification time
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.Before(backups[j].modTime)
	})
	return backups, nil
}

// SetOption is configuring rotating file function types
type SetOption func(*Option) error

func WithMaxSize(size int64) SetOption {
	return func(opt *Option) error {
		if size > 0 && size < 1<<12 {
			errors.Warningf("too small max size:%d, it may cause frequent rotation", size)
		}
		opt.MaxSize = size
		return nil
	}
}

func WithMaxAge(age time.Duration) SetOption {
	return func(opt *Option) error {
		if age < 0 {
			errors.Warningf("max age:%s is less than zero, not limited by max age", age)
		}
		opt.MaxAge = age
		return nil
	}
}

func WithBackups(backups int) SetOption {
	return func(opt *Option) error {
		if backups < 0 {
			errors.Warningf("backups:%d is less than zero, not limited by backups", backups)
		}
		opt.Backups = backups
		return nil
	}
}

func WithBackupPrefix(prefix string) SetOption {
	return func(opt *Option) error {
		length := len(prefix)
		if length == 0 || length > 128 {
			return InvalidBackupPrefixError
		}
		for _, char := range prefix {
			if !unicode.IsLetter(char) && char != '-' {
				return errors.Newf("backup prefix contains invalid character '%c'", char)
			}
		}
		opt.BackupPrefix = prefix
		return nil
	}
}

func WithModePerm(perm os.FileMode) SetOption {
	return func(opt *Option) error {
		if perm&writeMode == 0 {
			return ModePermissionError
		}
		opt.ModePerm = perm
		return nil
	}
}

func WithCompressLevel(level int) SetOption {
	return func(opt *Option) error {
		// level <= 0 means no compression
		if level > 9 {
			return InvalidCompressionLevelError
		}
		opt.CompressLevel = level
		return nil
	}
}

func WithDuration(duration time.Duration) SetOption {
	return func(opt *Option) error {
		if duration > 0 && duration < time.Hour {
			errors.Warningf("too short duration:%s, it may cause frequent rotation", duration)
		}
		opt.Duration = duration
		return nil
	}
}

// NewRotatingFile creates a new rotating file with the specified options.
func NewRotatingFile(file string, opts ...SetOption) (*RotatingFile, error) {

	absFile, err := paths.Abs(file)
	if err != nil {
		return nil, err
	}

	folder, filename := filepath.Split(absFile)
	r := &RotatingFile{
		file:     absFile,
		folder:   folder,
		filename: filename,
		option:   defaultOption.clone(),
	}

	// config rotating file options
	for _, opt := range opts {
		if opt != nil {
			err = errors.Join(err, opt(r.option))
		}
	}
	if err != nil {
		return nil, errors.Newf("failed to set option, err: %s", err)
	}

	// active daemon goroutine
	if r.option.Duration > 0 {
		r.timer = time.NewTimer(r.option.Duration)
		go func() {
			for {
				select {
				case now := <-r.timer.C:
					func() {
						r.mtx.Lock()
						defer r.mtx.Unlock()
						if r.writer != nil && now.Sub(r.rotatingTime) > r.option.Duration {
							errors.Warning(r.rotate())
						}
					}()
				default:
				}
			}
		}()
	}
	return r, nil
}
