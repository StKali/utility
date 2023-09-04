package errors

import (
	stderr "errors"
	"fmt"
	"github.com/stkali/utility/tool"
	"io"
	"runtime"
)

var Is = stderr.Is
var As = stderr.As
var Join = stderr.Join

type iErr struct {
	descriptions []string
	err          error
	Tracer
}

func (i *iErr) Unwrap() error {
	return i.err
}

func (i *iErr) Is(err error) bool {
	return stderr.Is(i.err, err)
}

func (i *iErr) desc() string {
	length := len(i.descriptions)
	if length == 0 {
		return ""
	}

	var b []byte
	for index := length - 1; index >= 0; index-- {
		if index > 0 {
			b = append(b, ',')
		}
		b = append(b, i.descriptions[index]...)
	}
	return tool.ToString(b)
}

func (i *iErr) Error() string {
	if desc := i.desc(); desc != "" {
		return fmt.Sprintf("%s, err: %s", desc, i.err.Error())
	}
	return i.err.Error()
}

func (i *iErr) Format(f fmt.State, verb rune) {
	fmt.Println(verb)
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

func Errorf(format string, a ...any) error {
	return &iErr{
		err:    fmt.Errorf(format, a...),
		Tracer: getTrace(3),
	}
}

func Wrap(desc string, err error) error {
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *iErr:
		e.descriptions = append(e.descriptions, desc)
		return e
	default:
		return &iErr{
			descriptions: []string{desc},
			err:          err,
			Tracer:       getTrace(3),
		}
	}
}
