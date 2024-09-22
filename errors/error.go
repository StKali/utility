package errors

import (
	stderr "errors"
	"fmt"
	"io"
	"os"
)

var (
	// Standard error functions from errors package
	Is     = stderr.Is
	As     = stderr.As
	Error  = stderr.New
	Unwrap = stderr.Unwrap
	// for testing
	osExit = os.Exit
)

// iErr represents a custom error type that can hold multiple errors and a tracer.
// tracer will keep the first error information.
type iErr struct {
	errs      []error
	argErrNum int
	// Tracer interface for stack tracing
	Tracer
}

// Ensure iErr implements the error interface.
var _ error = (*iErr)(nil)

// Ensure iErr implements the fmt.Formatter interface.
var _ fmt.Formatter = (*iErr)(nil)

// New creates a new iErr with a single error and a tracer.
func New(text string) error {
	return &iErr{
		errs:   []error{stderr.New(text)},
		Tracer: GetTrace(3),
	}
}

// Newf creates a new iErr with a formatted error message and potentially multiple errors.
func Newf(format string, a ...any) error {
	// Initialize the error and handle cases without additional errors.
	err := &iErr{}
	length := len(a)
	if length == 0 {
		return &iErr{
			errs:      []error{stderr.New(format)},
			argErrNum: 0,
			Tracer:    GetTrace(3),
		}
	}
	// Iterate over arguments to find errors and potential tracer.
	for i := length - 1; i >= 0; i-- {
		// Count errors and set tracer if not already set.
		if _, ok := a[i].(error); ok {
			err.argErrNum++
		}
		if err.Tracer == nil {
			if v, ok := a[i].(*iErr); ok {
				err.Tracer = v.Tracer
			}
		}
	}
	// Allocate errors slice with the expected size.
	err.errs = make([]error, 0, err.argErrNum+1)
	// Append all errors and the formatted error message.
	for _, e := range a {
		if argErr, ok := e.(error); ok {
			err.errs = append(err.errs, argErr)
		}
	}
	err.errs = append(err.errs, Error(fmt.Sprintf(format, a...)))
	// Ensure tracer is set.
	if err.Tracer == nil {
		err.Tracer = GetTrace(3)
	}
	return err
}

// Unwrap returns the list of errors wrapped by iErr.
func (i *iErr) Unwrap() []error {
	return i.errs
}

// Is checks if the error chain contains a specific error.
func (i *iErr) Is(err error) bool {
	for _, e := range i.errs {
		if Is(e, err) {
			return true
		}
	}
	return false
}

// Error returns a formatted string of the errors after skipping the first argErrNum errors.
func (i *iErr) Error() string {
	var b []byte
	for index, err := range i.errs[i.argErrNum:] {
		if index > 0 {
			b = append(b, '\n')
		}
		b = append(b, err.Error()...)
	}
	return string(b)
}

// Format implements the fmt.Formatter interface.
// %s %q will print error string.
// %v will print error string with trace stack information.
func (i *iErr) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		_, _ = fmt.Fprintf(f, "Error: %s\n", i.Error())
		i.Traceback(f)
	case 'q':
		_, _ = fmt.Fprintf(f, "%q", i.Error())
	default:
		_, _ = io.WriteString(f, i.Error())
	}
}

// Join combines multiple errors into a single error, ignoring nil values.
// It is useful for scenarios where multiple errors may occur during a function's execution,
// and the caller wishes to handle them collectively rather than individually.
//
// The function iterates through the provided slice of errors (errs...).
// If the slice is empty or contains only nil values, it returns nil indicating no error.
// Otherwise, it constructs a new error type  *iErr,  that encapsulates all non-nil errors found in the input slice.
// If any of the input errors are of type *iErr and contain trace information (Tracer),
// the first encountered trace information will be propagated to the new combined error.
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
