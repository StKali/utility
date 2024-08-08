package errors

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
)

// Tracer is an interface that represents a stack trace.
// It provides methods to print or manipulate the stack trace.
type Tracer interface {
	StackTrace(fd io.Writer)
	RangeFrames(handle func(frame runtime.Frame))
	fmt.Stringer
}

// depth defines the maximum depth of the stack trace to capture.
// It is set to 2^5 (32) for efficiency and to avoid capturing too much stack information.
const depth = 1 << 5

// trace represents a slice of program counters that can be used to reconstruct a stack trace.
type trace []uintptr

// String implements fmt.Stringer.
func (t trace) String() string {
	buf := &bytes.Buffer{}
	t.StackTrace(buf)
	return buf.String()
}

var _ Tracer = (*trace)(nil)

// RangeFrames iterates over the stack trace and calls the provided handle function
// for each stack frame. If the handle function is nil, it panics.
func (t trace) RangeFrames(handle func(frame runtime.Frame)) {
	if handle == nil {
		handle = defaultFrameHandle
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

func defaultFrameHandle(frame runtime.Frame) {
	_, _ = fmt.Fprintf(errOutput, "    %s(...)\n", frame.Function)
	_, _ = fmt.Fprintf(errOutput, "         %s:%d\n", frame.File, frame.Line)
}

// StackTrace writes a formatted stack trace to the provided io.Writer.
// It uses a default handler that prints the function name and file/line information for each frame.
func (t trace) StackTrace(fd io.Writer) {
	_, _ = fmt.Fprintln(fd, "Traceback:")
	t.RangeFrames(func(frame runtime.Frame) {
		_, _ = fmt.Fprintf(fd, "    %s(...)\n", frame.Function)
		_, _ = fmt.Fprintf(fd, "         %s:%d\n", frame.File, frame.Line)
	})
}

// GetTrace captures the current goroutine's stack trace, skipping the specified number of frames.
// It returns a Tracer interface that can be used to print or manipulate the stack trace.
func GetTrace(skip int) Tracer {
	pcs := make(trace, depth, depth)
	count := runtime.Callers(skip, pcs[:])
	return pcs[:count]
}

// StackTrace writes the traceback information of the caller to the specified io.Writer.
// It starts capturing the stack trace from the caller's caller (i.e., 3 levels up the call stack
// to exclude the current function and its immediate caller).
func StackTrace(fd io.Writer) {
	tc := GetTrace(3)
	tc.StackTrace(fd)
}

// GetTraceback returns traceback stack string
func GetTraceback() string {
	tc := GetTrace(4)
	return tc.String()
}
