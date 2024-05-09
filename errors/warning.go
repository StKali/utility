package errors

import (
	"fmt"
	"io"
	"os"
)

var (
	disableWarning bool
	warningPrefix            = "warning"
	warningOutput  io.Writer = os.Stderr
)

// DisableWarning disable the global warning.
func DisableWarning() {
	disableWarning = true
}

// SetWarningOutput set warning output writable.
func SetWarningOutput(output io.Writer) {
	warningOutput = output
}

// SetWarningPrefix set the warning message prefix.
func SetWarningPrefix(prefix string) {
	warningPrefix = prefix
}

// SetWarningPrefix set the warning message prefix.
func SetWarningPrefixf(s string, args ... any) {
	warningPrefix = fmt.Sprintf(s, args...)
}

func warn(format *string, a ...any) {
	var msg string
	if format == nil {
		msg = fmt.Sprint(a...)
	} else {
		msg = fmt.Sprintf(*format, a...)
	}
	if warningPrefix == "" {
		_, _ = fmt.Fprintln(warningOutput, msg)
	} else {
		_, _ = fmt.Fprintf(warningOutput, "%s: %s\n", warningPrefix, msg)
	}
}

// Warning writes all parameters to the specified writable object, ignoring warnings
// when passed in empty
func Warning(a ...any) {
	if disableWarning || a == nil || len(a) == 0 || (len(a) == 1 && a[0] == nil) {
		return
	}
	warn(nil, a...)
}

// Warningf accepts the format string and corresponding parameters, and outputs
// the information as a warning to the specified writable object
func Warningf(format string, a ...any) {
	if disableWarning {
		return
	}
	warn(&format, a...)
}
