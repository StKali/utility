package errors

import (
	"fmt"
	"io"
	"runtime"
)

const depth = 1 << 5

type trace []uintptr

func (t trace) StackWithHandle(handle func(frame runtime.Frame)) {
	if handle == nil {
		panic("handle is nil")
	}
	fs := runtime.CallersFrames(t)
	var ok = true
	var frame runtime.Frame
	for ; ok; frame, ok = fs.Next() {
		if frame.Function != "" {
			handle(frame)
		}
	}
}

func (t trace) Stack(fd io.Writer) {
	t.StackWithHandle(func(frame runtime.Frame) {
		_, _ = fmt.Fprintf(fd, "%s(...)\n", frame.Function)
		_, _ = fmt.Fprintf(fd, "\t%s:%d\n", frame.File, frame.Line)
	})
}

func GetTrace(skip int) Tracer {
	pcs := make(trace, depth, depth)
	count := runtime.Callers(skip, pcs[:])
	return pcs[:count]
}

type Tracer interface {
	Stack(fd io.Writer)
	StackWithHandle(handle func(frame runtime.Frame))
}
