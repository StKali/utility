# Errors

Package errors provides errors with traceback, error history, and simple warnings, as well as some exit hooks.



## errors

Fully compatible with standard error function.

### Install

```shell
go get github.com/stkali/utility/errors@latest
```

### samples

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



### compatible with standard libraries

`Is`, `Join`, `As`, `Unwrap`, `New` are all compatible with the standard library,
and it should be noted that `New` in the standard library is equivalent to `errors.Error`.

The return values of both `errors.New` and `errors.Newf` follow the `interface{ Unwrap() []error}` and `errors.Tracer` interface.

| standard library errors | utility/errors | Description                  |
| ----------------------- |----------------| ---------------------------- |
| New                     | New            | identical                    |
| Is                      | Is             | identical                    |
| As                      | As             | identical                    |
| Unwrap                  | Unwrap         | identical                    |
| Join                    | Join           | compatible                   |
| -                       | Errorf/Newf    | returns error with traceback |
| -                       | Error          | returns error with traceback |

### GetTraceback

`GetTrace` captures the current goroutine's stack trace, skipping the specified number of frames.

```go
tb := GetTraceback()
// Output:
// Traceback:
//      main.main(...)
//          /home/user/project/main.go:7
//      runtime.main(...)
//          /usr/local/go/src/runtime/proc.go:271
fmt.Print(tb)
```



### Traceback

`Traceback` writes the traceback information of the caller to the specified io.Writer.

```go
// Output:
// Traceback:
//      main.main(...)
//          /home/user/project/main.go:9
//      runtime.main(...)
//          /usr/local/go/src/runtime/proc.go:271
errors.Traceback(os.Stdout)
```



### GetTrace

`GetTrace` returns a `Tracer` interface that can be used to print or manipulate the stack trace.

```go
trace := errors.GetTrace(2)
```



## warning
The errors package provides the warning function, which makes it easy to output warning messages.


### SetWarningPrefix, SetWarningPrefixf

set warning prefix, default: warning.
> note: the ': ' will fill between prefix and message

```go
errors.SetWarningPrefix("WARNING")

// specify prefix with format
errors.SetWarningPrefixf("%s Warning", "AppName")
```



### SetWarningOutput

set warning output to os.Stdout, default: os.Stderr.
```go
errors.SetWarningOutput(os.Stdout)
```



### Warning, Warningf

print warning messages to err output, default: os.Stderr.

```go
// when multiple warnings are given, they are split using commas
//Output: warning：message1, message2
errors.Warning("message1", "message2")

// Output: warning：number must be greater than 10
errors.Warningf("number must be greater than %d", number)
```


### DisableWarning

all warning messages will be ignored.
> note: disabling is irreversible
```go
errors.DisableWarning()
```



## exit

errors package provides `Exit`, `Exitf`, `CheckErr` three functions, you can exit the program before doing some necessary end work.
`ExitHook` can be set using the `SetExitHook` function.



### SetExitHook

set exit hook function, which will be called before exit the program.
code: exit code
msg: error message
tracer: error traceback
```go
errors.SetExitHook(func(code int, msg string, tracer Tracer) {
    ... // custom logic
})
```



### SetErrPrefix, SetErrPrefixf

Set error prefix, default: occurred error.
> ': ' will fill between prefix and message if prefix is not empty.

```go
errors.SetErrPrefix("Error")
errors.SetErrPrefixf("%s Error", "AppName")
```



### SetErrOutput

Set error output, default: os.Stderr
```go
errors.SetErrOutput(os.Stdout)
```



### Exit

Exit the program and call ExitHook before, only specify exit code.
```go
errors.Exit(1)
```



### Exitf

Exit the program and call ExitHook before, specify exit code and error message.
```go
errors.Exitf(1, "failed to do something")
```



### CheckErr

When the argument is not nil, exit the program and call `ExitHook` If it is not nil.
```go
err := setup()
errors.CheckErr(err)
err = doSomething()
errors.CheckErr(err)
```
