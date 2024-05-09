package errors

import (
	stderr "errors"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/stkali/utility/tool"
)

var (
	errPrefix           = "occurred error"
	Is                  = stderr.Is
	As                  = stderr.As
	Error               = stderr.New
	errOutput io.Writer = os.Stderr
)

type iErr struct {
	errs      []error
	argErrNum int
	Tracer
}

var _ error = (*iErr)(nil)
var _ fmt.Formatter = (*iErr)(nil)

func New(text string) error {
	return &iErr{
		errs:   []error{stderr.New(text)},
		Tracer: GetTrace(3),
	}
}

func Newf(format string, a ...any) error {
	err := &iErr{}
	length := len(a)
	if length == 0 {
		return &iErr{
			errs:      []error{stderr.New(format)},
			argErrNum: 0,
			Tracer:    GetTrace(3),
		}
	}

	for i := length - 1; i >= 0; i-- {
		if _, ok := a[i].(error); ok {
			err.argErrNum++
		}
		if err.Tracer == nil {
			if v, ok := a[i].(*iErr); ok {
				err.Tracer = v.Tracer
			}
		}
	}

	err.errs = make([]error, 0, err.argErrNum+1)
	for _, e := range a {
		if argErr, ok := e.(error); ok {
			err.errs = append(err.errs, argErr)
		}
	}
	err.errs = append(err.errs, stderr.New(fmt.Sprintf(format, a...)))
	if err.Tracer == nil {
		err.Tracer = GetTrace(3)
	}
	return err
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

// Error [inner1, inner2, err(e1, e2), ]
func (i *iErr) Error() string {
	var b []byte
	for index, err := range i.errs[i.argErrNum:] {
		if index > 0 {
			b = append(b, '\n')
		}
		b = append(b, err.Error()...)
	}
	return tool.ToString(b)
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

func Join(errs ...error) error {

	length := len(errs)
	if length == 0 {
		return nil
	}
	errCount := 0
	for i := 0; i < length; i++ {
		if errs[i] != nil {
			errCount++
		}
	}
	if errCount == 0 {
		return nil
	}

	newErr := &iErr{
		errs: make([]error, 0, errCount),
	}
	for i := 0; i < length; i++ {
		if newErr.Tracer == nil {
			if v, ok := errs[i].(*iErr); ok {
				newErr.Tracer = v.Tracer
			}
		}
		if errs[i] != nil {
			newErr.errs = append(newErr.errs, errs[i])
		}
	}
	return newErr
}

// SetErrPrefix set prefix of CheckError output string
func SetErrPrefix(prefix string) {
	errPrefix = prefix
}

// SetErrorPrefixf set prefix of CheckError output string with args
func SetErrPrefixf(s string, args ...any) {
	errPrefix = fmt.Sprintf(s, args...)
}

// SetErrOutput set error output writable.
func SetErrOutput(writer io.Writer) {
	errOutput = writer
}

// CheckError prints the message with the prefix and exits with error code 1
// if the message is nil, it does nothing.
func CheckErr(err error) {
	if err == nil {
		return
	}
	if errPrefix == "" {
		_, _ = fmt.Fprintln(errOutput, err)
	} else {
		_, _ = fmt.Fprintf(errOutput, "%s: %+s\n", errPrefix, err)
	}
	tool.Exit(1)
}
