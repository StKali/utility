package errors

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	// errPrefix is a prefix string appended to the beginning of error messages.
	errPrefix = "occurred error"

	// errOutput is the writer used for error output, defaulting to os.Stderr.
	// SetErrOutput and CheckErr function will use it.
	errOutput io.Writer = os.Stderr

	// exitHook is a function hook that gets called before the program exits due to an error.
	// It is provided the error message and a tracer.
	exitHook ExitHook = nil
)

// ExitHook defines the signature of a function that can be set as a hook to execute before
// program exit.
type ExitHook func(code int, msg string, tracer Tracer)

// SetErrPrefix allows changing the prefix string used in error messages.
func SetErrPrefix(prefix string) {
	errPrefix = prefix
}

// SetErrPrefixf allows setting the prefix string of CheckErr output with formatted arguments.
func SetErrPrefixf(s string, args ...any) {
	errPrefix = fmt.Sprintf(s, args...)
}

// SetErrOutput set error output writable.
func SetErrOutput(writer io.Writer) {
	errOutput = writer
}

// SetExitHook sets a custom hook function to be called before the program exits due to an error.
func SetExitHook(hook ExitHook) {
	exitHook = hook
}

// Exit allows customizing the function used to exit behavior of the program,
// which is used in tests containing the os.Exit code.
// defaults to os.Exit.
func Exit(code int) {
	if exitHook != nil {
		exitHook(code, "", GetTrace(3))
	}
	osExit(code)
}

// Exitf prints a formatted error message to the error output, calls the exit hook (if set),
// and then exits the program with the given code.
func Exitf(code int, format string, args ...any) {
	if errPrefix != "" {
		var sb strings.Builder
		sb.Grow(len(errPrefix) + 2 + len(format))
		sb.WriteString(errPrefix)
		sb.WriteString(": ")
		sb.WriteString(format)
		format = sb.String()
	}
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprint(errOutput, msg)
	if exitHook != nil {
		exitHook(code, msg, GetTrace(3))
	}
	osExit(code)
}

// CheckErr prints an error message with the set prefix to stderr and exits the program with code 1
// if the provided error is not nil or empty.
func CheckErr(err any) {

	if err == nil || err == "" {
		return
	}
	var msg string
	if errPrefix == "" {
		msg = fmt.Sprintf("%s", err)
	} else {
		msg = fmt.Sprintf("%s: %s", errPrefix, err)
	}
	_, _ = fmt.Fprintln(errOutput, msg)
	if exitHook != nil {
		var tracer Tracer
		if errVal, ok := err.(*iErr); ok {
			tracer = errVal.Tracer
		} else {
			tracer = GetTrace(3)
		}
		exitHook(1, msg, tracer)
	}
	osExit(1)
}
