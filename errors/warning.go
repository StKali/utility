package errors

import (
	"fmt"
	"io"
	"os"
)

var (
	// disableWarning is a global flag that controls whether warnings are disabled.
	disableWarning bool

	// warningPrefix is the prefix used for warning messages.
	warningPrefix = "warning"

	// warningOutput is the io.Writer where warning messages are sent by default.
	// It is set to os.Stderr initially.
	warningOutput io.Writer = os.Stderr
)

// DisableWarning disables the global warning mechanism.
// After calling this function, no warnings will be output.
func DisableWarning() {
	disableWarning = true
}

// SetWarningOutput sets the output destination for warning messages.
// The provided io.Writer will be used to write warning messages.
func SetWarningOutput(output io.Writer) {
	warningOutput = output
}

// SetWarningPrefix sets the prefix used for warning messages.
// This prefix will be prepended to all warning messages.
func SetWarningPrefix(prefix string) {
	warningPrefix = prefix
}

// SetWarningPrefixf is a formatted version of SetWarningPrefix.
// It allows setting the prefix using a format string and arguments.
func SetWarningPrefixf(s string, args ...any) {
	warningPrefix = fmt.Sprintf(s, args...)
}

// warn is an internal function that writes a warning message to the specified output.
// It handles formatting and prefixing the message.
func warn(format *string, a ...any) {

	if warningPrefix != "" {
		_, _ = io.WriteString(warningOutput, warningPrefix)
		_, _ = io.WriteString(warningOutput, ": ")
	}
	if format == nil {
		var n any
		for index := range a {
			if index != 0 {
				_, _ = fmt.Fprint(warningOutput, n)
				_, _ = io.WriteString(warningOutput, ", ")
			}
			if e, ok := a[index].(error); ok {
				n = e.Error()
			} else {
				n = a[index]
			}
		}
		_, _ = fmt.Fprint(warningOutput, n)
	} else {
		_, _ = fmt.Fprintf(warningOutput, *format, a...)
	}
	_, _ = warningOutput.Write([]byte{'\n'})
}

// Warning writes a warning message to the specified output.
// It ignores warnings if the warning mechanism is disabled, or if no parameters are provided.
func Warning(a ...any) {
	// Check if warnings are disabled or no parameters are provided
	if disableWarning || a == nil || (len(a) == 1 && a[0] == nil) {
		return
	}
	warn(nil, a...)
}

// Warningf writes a formatted warning message to the specified output.
// It accepts a format string and corresponding parameters, and outputs the formatted message as a warning.
// It does not output the warning if the warning mechanism is disabled.
func Warningf(format string, a ...any) {
	if disableWarning {
		return
	}
	warn(&format, a...)
}
