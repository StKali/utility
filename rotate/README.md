## Rotate

Package rotate provides a rotating file writer that can be used to write data to.
It implements the io.WriteCloser interface and WriteString method.



### Features

- Nearly lossless write speeds.
- Supported for size, time, or both rotation.
- Delete old backups by number of backups, maxAge, or both.
- Allow compression of backup files.
- Flexible configuration to cover most scenarios.
- 100% test coverage.



### Usage

Install

```shell
go get github.com/stkali/utility/rotate@latest
```



Sample

Configurations are in the form of WithXXX, and the specific meaning of each configuration is listed below.

> NOTE: rotate file instance need to be closed.

```go
// create a rotating file writer
f, err := rotate.NewRotatingFile("temporary/test.log",
		// set maximum file size to 1GB
		rotate.WithMaxSize(lib.GB),
		// set maximum age to 1 month
		rotate.WithMaxAge(lib.Month),
		// keep 30 backup files
		rotate.WithBackups(30),
		// compress backup files with level 9
		rotate.WithCompressLevel(9),
		// rotate files every day
		rotate.WithDuration(lib.Day),
		// set backup file prefix to "backup-"
		rotate.WithBackupPrefix("backup-"),
		// set backup file mode to 0644
		rotate.WithModePerm(0644),
)
if err != nil {
    panic(err)
}

// Close rotate file !!!
defer f.Close()

f.WriteString("hello world\n")

// impletments io.Writer
f.Write([]byte("hello world\n"))

// set the default logger output to the rotating file
log.SetOutput(f) 
```



**MaxSize** (default: 1 GB) 

MaxSize is the threshold value that triggers size-based file rotation.
<= 0 means no rotation based on file size.

**Duration**(default: 1 day)

Duration is the threshold value that triggers time-based
file rotation.
<= 0 means no rotation based on time interval.

> NOTE:
>   The timer is updated every time rotating, and this includes MaxSize-based updates
>   Duration time.Duration

**MaxAge**(default: 30 days) 

MaxAge is the maximum age that a backup file can have before it is considered for cleanup.
Files older than this duration will be deleted during the cleanup goroutine.
= 0 means no backup files are retained.
< 0 the backup deletion strategy based on `MaxAge` will not work.


**ModePerm**(default: 0o644) 

ModePerm is the default file permission bits used when creating new rotating files.


**Backups**(default: 30) 

Backups is the maximum number of backup files that can be retained after rotation. When this limit is reached, the oldest backup file will be deleted to make space for new ones.
= 0 means no backup files are retained.
< 0 the backup deletion strategy based on `Backups` will not work.


**CompressLevel**(default: 6)

Specifies the compression level(1-9) used when compressing rotating files.
<= 0 means no compression.

**BackupFilePrefix**(default: rotating-)

BackupPrefix is the prefix to use when creating backup files.



### Workflow

The writer is created on the first write, and if the file exists at that point it will determine if the file satisfies the rotation condition. If it does, the file is changed to a backup and a new file is created, otherwise the file continues to be used.

Whenever the rotation condition is met, it blocks the write and renames the current file to the backup file, then creates a new file with the same name to continue the write. In other words, we are always using the “same file”.

Each time a rotation is completed, an attempt is made to trigger an asynchronous tidy backups. There are two main parts to this: 

1 Deleting backups that don't meet the criteria.

2 Compressing undeleted backups that don't have compression if the compression level > 0.

It is not always possible to successfully trigger a tidy task after a rotation, and if there is already a tidy task being executed, no new task will be triggered. So there may be a delay in deleting and compressing the backup file, the chances of this are very small, and even if it occurs I think it is tolerable, in order to eliminate this effect. We do a compensating bailout during the Close phase to ensure that the backups are all as expected after Close. 



### Note 

In a size-based rotation strategy, whether rotation is required is judged after writing, not before. This may result in the file being slightly larger than the set `MaxSize`.  However, this method has the advantage of ensuring that at least one write operation is allowed to complete.

In the case of a single write that exceeds "MaxSize", if we determine whether rotation is required before writing, we will be stuck in an infinite rotation, because every write requires rotation first, and we will still face the same problem after rotation. 

At this point, we would have to prevent the write by returning an error to the upper level, but in practice, we usually don't want this to happen. Therefore, we choose to make the determination after the write so that at least one super-massive write can be performed, both to avoid unnecessary errors and for more extreme cases.







