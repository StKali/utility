package log

import (
	stderr "errors"
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

var InvalidDurationError = stderr.New("invalid duration (must be >= 0)")
var InvalidSizeError = stderr.New("invalid size (must be > 0)")
var InvalidAgeError = stderr.New("invalid Age (must be >= 0)")
var InvalidBackupsError = stderr.New("invalid backups (must be >= 0)")
var InvalidTimeFormatError = stderr.New("invalid time format string")
var InvalidRotateFileError = stderr.New("invalid rotating file")

type baseRotateFile struct {
	// filenameTime suffix format string
	backupTimeFormat string
	// the number of copies of the file that can be retained
	backups int
	// the directory where the rotating file is saved
	folder string
	// filename and backup file prefix
	name string
	// filename extension
	ext string
	// the writable object that is currently rotating
	fd io.WriteCloser
	// mutex ensures that the merger and penalty calls will not cause data insecurity problems
	mtx sync.Mutex
	// mark if there is currently a GC task being executed
	_cleaning AtomicBool
	// since it is expensive to clean up a backup, creating asynchronous processing will be the
	// preferred solution
	block bool
	// the maximum time for a dungeon to be born and wrong
	age time.Duration
	// the default permission bit when creating a rotating file
	modePerm os.FileMode
}

// noCopy may be added to structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
//
// Note that it must not be embedded, due to the Lock and Unlock methods.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// A Bool is an atomic boolean value.
// The zero value is false.
type AtomicBool struct {
	_ noCopy
	v uint32
}

// Load atomically loads and returns the value stored in x.
func (x *AtomicBool) Load() bool { return atomic.LoadUint32(&x.v) != 0 }

// Store atomically stores val into x.
func (x *AtomicBool) Store(val bool) { atomic.StoreUint32(&x.v, b32(val)) }

// Swap atomically stores new into x and returns the previous value.
func (x *AtomicBool) Swap(new bool) (old bool) { return atomic.SwapUint32(&x.v, b32(new)) != 0 }

// CompareAndSwap executes the compare-and-swap operation for the boolean value x.
func (x *AtomicBool) CompareAndSwap(old, new bool) (swapped bool) {
	return atomic.CompareAndSwapUint32(&x.v, b32(old), b32(new))
}

// b32 returns a uint32 0 or 1 representing b.
func b32(b bool) uint32 {
	if b {
		return 1
	}
	return 0
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
	for i := range format {
		if '0' <= format[i] && format[i] <= '6'{
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
		err = errors.Join(err, os.Remove(file))
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

// DurationRotateFile ...
type DurationRotateFile struct {
	// time granularity of backup
	duration time.Duration
	baseRotateFile
	// rotating timer
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

// rotate rotate file and reset timer
func (d *DurationRotateFile) rotate(block bool) error {

	if err := d.baseRotateFile.rotate(); err != nil {
		return err
	}
	d.setTimer(d.duration)
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
		d.timer.Reset(duration)
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
		d.setTimer(d.duration)
		return d.makeRotateFile(file)
	}
	if err != nil {
		return errors.Newf("failed to open file: %q, err: %s", file, err)
	}
	// open the leftover rotating file
	created, err := paths.GetFdCreated(info)
	if err != nil {
		return err
	}
	now := time.Now()
	expired := created.Add(d.duration)
	// use the leftover file if it is not expired
	if expired.After(now) {
		// update rotate timer
		d.setTimer(expired.Sub(now))
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


type SizeRotateFile struct {
	// 分片的大小
	size int64
	// 当前使用的大小
	used int64
	// 当前写入的文件对象
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
