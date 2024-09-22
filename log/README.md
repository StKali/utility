## log
Package log provides a simple logging interface with levels

# Usage

## Levels
A total of 6 levels are supported：
- `TRACE`
- `DEBUG`
- `DEBUG`
- `INFO`
- `WARN`
- `ERROR`

## Methods
- `Trace(args...any)`
- `Tracef(farmat string, args...any)`
- `Debug(args...any)`
- `Debugf(format string, args...any)`
- `Info(args...any)`
- `Infof(format string, args...any)`
- `Warn(args...any)`
- `Warnf(format string, args...any)`
- `Error(args...any)`
- `Errorf(format string, args...any)`

## Example

config log
```go
// message is ignored when the output method level is lower than the specified level.
log.SetLevel(log.INFO)

// Fully compatible with standard libraries
log.SetPrefix("[utility] xxxxxx ")
log.SetOutput(os.Stdout)
log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
```

use log
```go
// init log 
log.SetLevel(log.WARN)
log.SetPrefix("TEST-PREFIX")

// message will be ignored, TRACE < log.WARN
log.Trace("This message will not be output")

// Output: TEST-PREFIX: 2024/09/19 20:24:31 main.go:13: [WARN ] Hello, world!
log.Warn("Hello, world!")

// Output: TEST-PREFIX: 2024/09/19 20:24:31 main.go:13: [WARN ] Hello, world!
log.Warn("Hello, world!")

// Output: TEST-PREFIX: 2024/09/22 20:27:46 main.go:13: [WARN ] test number: 123, test nil: <nil>
log.Warnf("test number: %d, test nil: %v", 123, nil)
```
