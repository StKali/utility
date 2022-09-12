package tool

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
)

var depth = 32

func SetDepth(d int) {
	depth = d
}

func PrintStack(skip int) {
	stack(os.Stdout, skip+2)
}

func SaveStack(fd io.Writer, skip int) {
	stack(fd, skip+2)
}

func stack(fd io.Writer, skip int) {
	pcs := make([]uintptr, depth, depth)
	count := runtime.Callers(skip, pcs[:])
	callers := pcs[:count]
	fs := runtime.CallersFrames(callers)
	var frame runtime.Frame
	ok := true
	for ; ok; frame, ok = fs.Next() {
		if frame.Function != "" {
			_, _ = fmt.Fprintf(fd, "%s(...)\n", frame.Function)
			_, _ = fmt.Fprintf(fd, "\t%s:%d\n", frame.File, frame.Line)
		}
	}
}

func GetStack(skip int) string {
	buf := new(bytes.Buffer)
	stack(buf, skip+2)
	return buf.String()
}

func Recovery(fn func(e any, exception string)) {
	if err := recover(); err != nil {
		fn(err, GetStack(3))
	}
}
