// Rotated files can be sharded based on time and size, and the writing efficiency
// of rotated files is almost the same as that of native files.
//	==============================================================
//	name                              times           ns/op
//	--------------------------------------------------------------
//	system_file_write-10         	  808066	      1503
//	size_rotate_write-10  	          696607	      1484
//	duration_rotate_write-10      	  748053	      1481
//	==============================================================

package rotate

import (
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/stkali/utility/errors"
	"github.com/stkali/utility/paths"
)

// RotateFiler is the interface that define the rotating file
type RotateFiler interface {
	SetAge(age time.Duration) error
	SetBackups(count int) error
	SetBackupTimeFormat(format string) error
	Folder() string
	Age() time.Duration
	Backups() int
	BackupTimeFormat() string
	Rotate(block bool) error
	DropRotateFiles() error
	io.WriteCloser
}

const (
	defaultBackupTimeFormat = "2006-01-02-150405"
	defaultBackups          = 30
	defaultDuration         = 30 * 24 * time.Hour
	defaultSize             = 1 << 26             // 64MB
	defaultAge              = 30 * 24 * time.Hour // days
	defaultName             = "rotating"
	defaultExt              = ".log"
	defaultFolder           = "."
	defaultModePerm         = 0o644
)

var InvalidDurationError = errors.Error("invalid duration (must be >= 0)")
var ResetTimerError = errors.Error("failed to reset timer")
var InvalidSizeError = errors.Error("invalid size (must be > 0)")
var InvalidAgeError = errors.Error("invalid Age (must be >= 0)")
var InvalidBackupsError = errors.Error("invalid backups (must be >= 0)")
var InvalidTimeFormatError = errors.Error("invalid time format string")
var InvalidRotateFileError = errors.Error("invalid rotating file")

// baseRotateFile is a foundational struct for implementing log file rotation functionality.
// It provides the basic mechanisms and configuration options for rotating log files based on various criteria.
type baseRotateFile struct {
	// backupTimeFormat specifies the format string used to generate the timestamp suffix for backup files.
	// This format is applied to the current time when a log file is rotated to create a unique backup filename.
	backupTimeFormat string

	// backups defines the maximum number of backup files that can be retained after rotation.
	// When this limit is reached, the oldest backup file will be deleted to make room for new ones.
	backups int

	// folder specifies the directory where the rotating log file and its backups are saved.
	folder string

	// name is the prefix used for the main log file and its backup files.
	// The backup files will have a timestamp suffix appended to this prefix.
	name string

	// ext is the file extension used for the log file and its backups.
	// This allows for easy identification of the file type.
	ext string

	// fd is the current file descriptor (io.WriteCloser) that is being written to.
	// It represents the currently active log file.
	fd io.WriteCloser

	// mtx is a mutex that ensures thread-safe access to the struct's fields and methods.
	// It prevents data races and ensures that log rotation and writing operations are synchronized.
	mtx sync.Mutex

	// _cleaning (using an underscore prefix to avoid accidental use as a public field)
	// is an atomic.Bool that indicates whether a garbage collection (cleanup) task is currently being executed.
	// This allows for safe and efficient cleanup of old backup files.
	_cleaning atomic.Bool

	// block specifies whether backup file cleanup should be performed synchronously or asynchronously.
	// If true, cleanup will be performed in a separate goroutine to avoid blocking the main logging thread.
	block bool

	// age defines the maximum age that a backup file can have before it is considered for cleanup.
	// Files older than this duration will be deleted during the cleanup process.
	age time.Duration

	// modePerm specifies the default file permission bits used when creating new log files.
	// This ensures that the log files are created with the desired security settings.
	modePerm os.FileMode
}

// newBaseRotateFile create a new baseRotateFile
func newBaseRotateFile() baseRotateFile {
	return baseRotateFile{
		backupTimeFormat: defaultBackupTimeFormat,
		backups:          defaultBackups,
		folder:           paths.ToAbsPath(defaultFolder),
		name:             defaultName,
		ext:              defaultExt,
		modePerm:         defaultModePerm,
		age:              defaultAge,
	}
}

// SetBackups set max backups count, if the number of backup exceeds the set value, the earliest
// // copy will be deleted.
func (b *baseRotateFile) SetBackups(backups int) error {
	if backups < 0 {
		return InvalidBackupsError
	}
	b.backups = backups
	return nil
}

// Backups returns the max backup count.
func (b *baseRotateFile) Backups() int {
	return b.backups
}

// SetAge sets the max alive age, if the time elapsed since the file was created exceeds
// the maximum live age, it will be deleted.
func (b *baseRotateFile) SetAge(age time.Duration) error {
	if age < 0 {
		return InvalidAgeError
	}
	if age < time.Hour {
		errors.Warning("the age < 1 hour, it will be created lots of backup files")
	}
	b.age = age
	return nil
}

// Age returns the maximum live age of the backup
func (b *baseRotateFile) Age() time.Duration {
	return b.age
}

// Folder returns the storage directory of the rotating file.
func (b *baseRotateFile) Folder() string {
	tailIndex := len(b.folder) - 1
	if tailIndex > -1 && b.folder[tailIndex] == os.PathSeparator {
		return b.folder[:tailIndex]
	}
	return b.folder
}

// SetBackupTimeFormat sets the suffix format of the duplicate file, which should be a valid
// time formatting string.
func (b *baseRotateFile) SetBackupTimeFormat(format string) error {
	// validates the format
	if validateTimeFormat(format) {
		b.backupTimeFormat = format
		return nil
	}
	return InvalidTimeFormatError
}

// validateTimeFormat validates time format string
func validateTimeFormat(format string) bool {
	if len(format) == 0 {
		return false
	}
	for i := range format {
		if '0' <= format[i] && format[i] <= '6' {
			return true
		}
	}
	return false
}

// BackupTimeFormat returns the maximum number of copies allowed to exist
func (b *baseRotateFile) BackupTimeFormat() string {
	return b.backupTimeFormat
}

// DropRotateFiles deletes all rotating files, including backups and files in use.
func (b *baseRotateFile) DropRotateFiles() error {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	fs, err := b.getBackupFiles()
	if err != nil {
		return err
	}
	// add current filename to rotate file slice
	fs = append(fs, b.filename())
	for _, file := range fs {
		delErr := os.Remove(file)
		if !os.IsNotExist(delErr) {
			err = errors.Join(err, delErr)
		}
	}
	return err
}

// rotate rotates the files that have reached the critical condition, and when
// the backups filename exists, start numbering the files with the same backup
// name from 1 to prevent file overwrite. After rotating, create a new rotating
// file to replace the original file object.
func (b *baseRotateFile) rotate() error {
	if err := b.close(); err != nil {
		return err
	}
	// changed the old rotating file
	filename, backupFile := b.filename(), b.backupFile()
	if _, err := os.Stat(backupFile); err == nil {
		index := 1
		p := len(backupFile) - len(b.ext)
		var sb strings.Builder
		for err == nil {
			sb.Reset()
			sb.Grow(len(backupFile) + 2)
			sb.WriteString(backupFile[:p])
			sb.WriteByte('.')
			sb.WriteString(strconv.Itoa(index))
			sb.WriteString(backupFile[p:])
			_, err = os.Stat(sb.String())
			index++
		}
		backupFile = sb.String()
	}
	// rename filename to backups name
	if err := os.Rename(filename, backupFile); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Newf("failed to rename back rotating file, err: %s", err)
	}
	// create new rotating file
	return b.makeRotateFile(filename)
}

// filename generates the name of the rotating file from the current time.
func (b *baseRotateFile) filename() string {
	var sb strings.Builder
	folder := b.Folder()
	if folder == "" {
		sb.Grow(len(b.name) + len(b.ext))
	} else {
		sb.Grow(len(folder) + 1 + len(b.name) + len(b.ext))
		sb.WriteString(folder)
		sb.WriteByte(os.PathSeparator)
	}
	sb.WriteString(b.name)
	sb.WriteString(b.ext)
	return sb.String()
}

// backupFile returns a backup filepath
func (b *baseRotateFile) backupFile() string {
	var sb strings.Builder
	name := b.backupName(time.Now())
	folder := b.Folder()
	sb.Grow(len(folder) + 1 + len(name))
	sb.WriteString(folder)
	sb.WriteByte(os.PathSeparator)
	sb.WriteString(name)
	return sb.String()
}

// backupName returns the backups file name based on the time passed in
func (b *baseRotateFile) backupName(t time.Time) string {
	date := t.Format(b.backupTimeFormat)
	var sb strings.Builder
	sb.Grow(len(b.name) + len(date) + len(b.ext) + 1)
	sb.WriteString(b.name)
	sb.WriteByte('-')
	sb.WriteString(date)
	sb.WriteString(b.ext)
	return sb.String()
}

// makeRotateFile creates a new rotating file
func (b *baseRotateFile) makeRotateFile(filename string) error {
	err := os.MkdirAll(b.folder, os.ModePerm)
	if err != nil {
		return errors.Newf("failed to create new log file: %s, err: %s", filename, err)
	}
	// the file will be cleaned when others created it
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, b.modePerm)
	if err != nil {
		return errors.Newf("failed to create new log file: %s, err: %s", filename, err)
	}
	b.fd = f
	return nil
}

// close the file if it is open
func (b *baseRotateFile) close() error {
	if b.fd == nil {
		return nil
	}
	err := b.fd.Close()
	b.fd = nil
	if err != nil {
		return errors.Newf("failed to close %s, err: %s", b.filename(), err)
	}
	return nil
}

// isRotatingFile determines whether the pass file name is a backup of the rotated file
func (b *baseRotateFile) isRotatingFile(name string) bool {
	return len(name) >= len(b.name)+len(b.ext)+len(b.backupTimeFormat)+1 &&
		strings.HasPrefix(name, b.name) &&
		strings.HasSuffix(name, b.ext)
}

// getBackupFiles returns a list of all current backup files
func (b *baseRotateFile) getBackupFiles() ([]string, error) {
	fs, err := os.ReadDir(b.folder)
	if err != nil {
		return nil, errors.Newf("cannot read log folder: %s, err: %s", b.folder, err)
	}
	folder := b.Folder()
	var sb strings.Builder
	var backups []string
	for _, f := range fs {
		if f.IsDir() || !b.isRotatingFile(f.Name()) {
			continue
		}
		sb.Grow(len(folder) + 1 + len(f.Name()))
		sb.WriteString(folder)
		sb.WriteByte(os.PathSeparator)
		sb.WriteString(f.Name())
		backups = append(backups, sb.String())
		sb.Reset()
	}
	return backups, nil
}

// clean clean up expired backup files
func (b *baseRotateFile) clean() error {
	if b.backups == 0 && b.age == 0 {
		return nil
	}
	backups, err := b.getBackupFiles()
	if err != nil {
		return err
	}
	// sort the backup files
	// Because the file names are generated uniformly, they are generally sorted by file name,
	// which is also sorted by time. Problems may occur when the time format is modified.
	sort.Strings(backups)
	backups, err = b.cleanByBackups(backups)
	return errors.Join(err, b.cleanByAges(backups))
}

// cleanByBackups expiring backup files are cleaned up based on the number of backup
func (b *baseRotateFile) cleanByBackups(orderBackups []string) ([]string, error) {
	if b.backups == 0 || len(orderBackups) < b.backups {
		return orderBackups, nil
	}
	var err error
	gap := len(orderBackups) - b.backups
	for _, file := range orderBackups[:gap] {
		err = errors.Join(err, os.Remove(file))
	}
	if err != nil {
		return nil, errors.Newf("remove backup failed, err: %s", err)
	}
	return orderBackups[gap:], nil
}

// cleanByAges expiring backup files are cleaned up based on the live age
func (b *baseRotateFile) cleanByAges(backups []string) (err error) {
	if b.age == 0 || len(backups) == 0 {
		return nil
	}
	expire := time.Now().Add(-b.age)
	oldest := b.backupName(expire)
	gap := len(b.Folder()) + 1
	for i := range backups {
		if backups[i][gap:] <= oldest {
			err = errors.Join(os.Remove(backups[i]))
		}
	}
	return err
}

// cleanBackups clean up expired backups
// checks whether there is any goroutine performing the cleaning operation.
// If so, abandon the cleanup. If not, start the cleanup.
func (b *baseRotateFile) cleanBackups(block bool) error {
	// existed a running cleanup goroutine
	if !b._cleaning.CompareAndSwap(false, true) {
		return nil
	}
	// block the groutine until the clean finished
	if block {
		defer b._cleaning.Store(false)
		return b.clean()
	}
	// start a cleanup goroutine to delete the expired backups
	go func() {
		defer b._cleaning.Store(false)
		errors.Warning(b.clean())
	}()
	return nil
}

// useLeftoverFile use leftover files as rotating file
// raise no such file err when the leftover file not found in folder
func (b *baseRotateFile) useLeftoverFile(filename string) error {
	fd, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, b.modePerm)
	if err != nil {
		return errors.Newf("failed to open rotating file: %q, err: %s", filename, err)
	}
	b.fd = fd
	return nil
}

// DurationRotateFile represents a file rotation mechanism based on a specified time duration.
// It utilizes a timer to automatically create new files for data logging or storage
// at regular intervals, allowing for the management of large or continuously growing files.
type DurationRotateFile struct {
	baseRotateFile
	// duration specifies the time interval after which a new file should be created
	// for further data logging or storage. This value represents the duration between
	// file rotations.
	duration time.Duration
	// timer is a pointer to a time.Timer that is used to schedule the next file rotation
	// based on the duration specified. When the timer expires, a new file is created
	// and the timer is reset for the next rotation.
	timer *time.Timer
}

var _ RotateFiler = (*DurationRotateFile)(nil)

// NewDurationRotateFile create a duration rotating file object.
func NewDurationRotateFile(file string, duration time.Duration) (*DurationRotateFile, error) {
	if duration < 0 {
		return nil, InvalidDurationError
	}
	if duration < time.Hour {
		errors.Warning("duration < 1 hour, it will created lots of backup files")
	}
	f := &DurationRotateFile{
		duration:       duration,
		baseRotateFile: newBaseRotateFile(),
	}
	if file != "" {
		file = paths.ToAbsPath(file)
		if info, err := os.Stat(file); err == nil && info.IsDir() {
			return nil, InvalidRotateFileError
		}
		f.folder, f.name, f.ext = paths.SplitWithExt(file)
	}
	go func() {
		for {
			if f.timer != nil {
				select {
				case <-f.timer.C:
					if err := f.Rotate(f.block); err != nil {
						errors.Warning(err)
					}
				}
			}
		}
	}()
	return f, nil
}

// DefaultDurationRotateFile returns default durationRotateFile
func DefaultDurationRotateFile() *DurationRotateFile {
	return &defaultDurationRotateFile
}

var defaultDurationRotateFile = DurationRotateFile{
	duration:       defaultDuration,
	baseRotateFile: newBaseRotateFile(),
	timer:          time.NewTimer(defaultDuration),
}

// SetDuration set rotating duration
func (d *DurationRotateFile) SetDuration(duration time.Duration) error {
	if duration < time.Second {
		return InvalidDurationError
	}
	if duration < time.Hour {
		errors.Warning("duration < 1 hour, it will created lots of backup files")
	}
	d.duration = duration
	return nil
}

// Rotate files according to the size and age.
func (d *DurationRotateFile) Rotate(block bool) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	return d.rotate(block)
}

// rotate file and reset timer
func (d *DurationRotateFile) rotate(block bool) error {
	if err := d.baseRotateFile.rotate(); err != nil {
		return err
	}
	if err := d.setTimer(d.duration); err != nil {
		return err
	}
	// clean old backups
	return d.cleanBackups(block)
}

// setTimer reset the timer if timer is existed else create a new timer for rotating
func (d *DurationRotateFile) setTimer(duration time.Duration) error {
	if duration < 1 {
		return InvalidDurationError
	}
	if d.timer == nil {
		d.timer = time.NewTimer(duration)
	} else {
		if !d.timer.Reset(duration) {
			return ResetTimerError
		}
	}
	return nil
}

// Write implements io.Writer.
// It will create if file not found in folder else use the leftover file.
func (d *DurationRotateFile) Write(p []byte) (int, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if d.fd == nil {
		if err := d.montRotateFile(d.filename()); err != nil {
			return 0, err
		}
	}
	return d.fd.Write(p)
}

// montRotateFile create rotating file if the rotate file not found in folder else
// use the leftover file.
func (d *DurationRotateFile) montRotateFile(file string) error {
	info, err := os.Stat(file)
	// creates the rotating file when not found
	if os.IsNotExist(err) {
		if err = d.setTimer(d.duration); err != nil {
			return err
		}
		return d.makeRotateFile(file)
	}
	if err != nil {
		return errors.Newf("failed to open file: %q, err: %s", file, err)
	}
	// open the leftover rotating file
	now := time.Now()
	expired := paths.GetFdCreated(info).Add(d.duration)
	// use the leftover file if it is not expired
	if expired.After(now) {
		// update rotate timer
		if err = d.setTimer(expired.Sub(now)); err != nil {
			return err
		}
		return d.useLeftoverFile(file)
	}
	return d.rotate(d.block)
}

// Close implements io.Closer, and closes the current rotating file.
func (d *DurationRotateFile) Close() error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	return d.close()
}

// SizeRotateFile represents a log rotation file structure that triggers rotation based on file size.
// When the amount of log data written to the file reaches a predefined size limit,
// a new log file is automatically created to continue logging.
// This struct embeds the baseRotateFile which provides the fundamental log rotation capabilities.
type SizeRotateFile struct {
	// size defines the threshold size (in bytes) that triggers log file rotation.
	// When the used space in the log file exceeds this value, a new log file is created.
	size int64

	// used tracks the amount of space already used in the current log file (in bytes).
	// This value increases as log data is written and resets to 0 when the size threshold is reached.
	used int64
	baseRotateFile
}

var _ RotateFiler = (*SizeRotateFile)(nil)

// NewSizeRotateFile create a size rotating file object.
func NewSizeRotateFile(file string, size int64) (*SizeRotateFile, error) {

	if size <= 0 {
		return nil, errors.Newf("size will be set 64MB when size is 0")
	}
	f := &SizeRotateFile{
		size:           size,
		baseRotateFile: newBaseRotateFile(),
	}

	if file != "" {
		file = paths.ToAbsPath(file)
		if info, err := os.Stat(file); err == nil && info.IsDir() {
			return nil, InvalidRotateFileError
		}
		f.folder, f.name, f.ext = paths.SplitWithExt(file)
	}
	return f, nil
}

// DefaultSizeRotateFile returns default defaultSizeRotateFile
func DefaultSizeRotateFile() *SizeRotateFile {
	return &defaultSizeRotateFile
}

var defaultSizeRotateFile = SizeRotateFile{
	size:           defaultSize,
	baseRotateFile: newBaseRotateFile(),
}

// SetSize set rotating size
func (s *SizeRotateFile) SetSize(size int) error {

	// invalid size
	if size <= 0 {
		return InvalidSizeError
	}

	// 4Mb
	if size < 1<<22 {
		errors.Warning("file size < 4MB will likely result in a large number of backups of the rotated file.")
	}

	s.size = int64(size)
	return nil
}

// Rotate files according to the size and age.
func (s *SizeRotateFile) Rotate(block bool) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.rotate(block)
}

// rotate rotate file and reset the used = 0
func (s *SizeRotateFile) rotate(block bool) error {
	if err := s.baseRotateFile.rotate(); err != nil {
		return err
	}
	s.used = 0
	// clean old backups
	return s.cleanBackups(block)
}

// Write implements io.Writer.
// when the file does not exist, the file will be created implicitly. each time writing is completed,
// it will check whether the file exceeds the limit(user > size). When the limit is exceeded, the
// current file will be saved as a backup and a new file with the same name will be created to replace
// the original file.
func (s *SizeRotateFile) Write(p []byte) (n int, err error) {

	sLen := int64(len(p))
	if sLen > s.size {
		return 0, errors.Newf("write length %d exceeds maximum file size %d", sLen, s.size)
	}
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.fd == nil {
		if err = s.montRotateFile(s.filename()); err != nil {
			return 0, err
		}
	}
	n, err = s.fd.Write(p)
	if err != nil {
		return n, errors.Newf("failed to write %s, err: %s", s.filename(), err)
	}
	s.used += int64(n)
	if s.used < s.size {
		return n, nil
	}
	return n, s.rotate(s.block)
}

// montRotateFile create rotating file if the rotate file not found in folder else
// use the leftover file.
func (s *SizeRotateFile) montRotateFile(file string) error {

	info, err := os.Stat(file)
	// creates the rotating file when not found
	if os.IsNotExist(err) {
		// cannot ensure the `used` is zero
		s.used = 0
		return s.makeRotateFile(file)
	}
	if err != nil {
		return errors.Newf("failed to open file: %q, err: %s", file, err)
	}
	// open the leftover rotating file and update `used`
	if info.Size() < s.size {
		s.used = info.Size()
		return s.useLeftoverFile(file)
	}
	return s.rotate(s.block)
}

// Close implements io.Closer, and closes the current logfile.
func (s *SizeRotateFile) Close() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.close()
}
