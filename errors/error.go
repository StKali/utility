package errors

import (
	stderr "errors"
	"fmt"
	"io"
	"runtime"
)

var Is = stderr.Is
var As = stderr.As
var Join = stderr.Join

type errString string

func (e errString) Error() string {
	return string(e)
}

type iErr struct {
	errs      []error
	argErrNum int
	Tracer
}

func (i *iErr) Unwrap() []error {
	return i.errs
}

func (i *iErr) Is(err error) bool {
	for _, e := range i.errs {
		if Is(e, err) {
			return true
		}
	}
	return false
}

func (i *iErr) Error() string {
	return i.errs[i.argErrNum].Error()
}

func (i *iErr) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		_, _ = fmt.Fprintf(f, "Error: %s\nTrace:\n", i.Error())
		i.StackWithHandle(func(frame runtime.Frame) {
			_, _ = fmt.Fprintf(f, "    %s(...)\n", frame.Function)
			_, _ = fmt.Fprintf(f, "         %s:%d\n", frame.File, frame.Line)
		})
	case 'q':
		_, _ = fmt.Fprintf(f, "%q", i.Error())
	default:
		_, _ = io.WriteString(f, i.Error())
	}
}

var _ error = (*iErr)(nil)
var _ fmt.Formatter = (*iErr)(nil)

func Newf(format string, a ...any) error {
	err := &iErr{}
	for _, e := range a {
		if argErr, ok := e.(error); ok {
			err.errs = append(err.errs, argErr)
			err.argErrNum++
		}
	}
	newErr := errString(fmt.Sprintf(format, a...))
	err.errs = append(err.errs, newErr)
	err.Tracer = GetTrace(3)
	return err
}

func New(text string) error {
	return &iErr{
		errs:   []error{errString(text)},
		Tracer: GetTrace(3),
	}
}
