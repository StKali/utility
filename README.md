# utility
A simple Go utility library.

## install

```shell
go get github.com/stkali/utility@latest
```

## errors
Package errors provides errors with traceback, error history, and simple warnings, as well as some exit hooks.

```go
file := "not exist file.txt"
// err is os.ErrNotExist
_, err := os.Open(file)

fmt.Println(errors.Is(err, os.ErrNotExist)) // Output: true

//err as a Newf parameter, which will be added to the error chain, along with the traceback
err = errors.Newf("failed to open file, err: %s", err)

// Returns true because os.ErrNotExist is added to the error chain
fmt.Println(errors.Is(err, os.ErrNotExist)) // Output: true

// Returns the error chain
errors.Unwrap(err)

// Output: failed to open file: not exist file.txt
fmt.Printf("%s\n", err)

// Output:
// Error: failed to open file, err: open not exist file.txt: no such file or directory
// Traceback:
//     main.main(...)
//         /home/user/project/main.go:13
//     runtime.main(...)
//         /usr/local/go/src/runtime/proc.go:250
fmt.Printf("%v\n", err)   // %v will print the traceback
```

[👉 more doc](errors/README.md)

## log
Package log provides a simple logging with levels

```go
// init log 
log.SetLevel(log.WARN)
log.SetPrefix("TEST-PREFIX")
log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

// message will be ignored, TRACE < log.WARN
log.Trace("This message will not be output")

// Output: TEST-PREFIX: 2024/09/19 20:24:31 main.go:13: [WARN ] Hello, world!
log.Warn("Hello, world!")

// Output: TEST-PREFIX: 2024/09/19 20:24:31 main.go:13: [WARN ] Hello, world!
log.Warn("Hello, world!")

// Output: TEST-PREFIX: 2024/09/22 20:27:46 main.go:13: [WARN ] test number: 123, test nil: <nil>
log.Warnf("test number: %d, test nil: %v", 123, nil)
```

[👉 more doc](log/README.md)

## rotate
Package rotate provides a rotating file writer that can be used to write data to.

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

[👉 more doc](rotate/README.md)

## LICENSE

[👉LICENSE](LICENSE)

## CHANGELOG

[👉CHANGELOG](CHANGELOG.md)
