package errors

import (
	"bytes"
	stderr "errors"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

var regxMatchErrorHeader = regexp.MustCompile(`(?m)^Error: .*`)
var regxMatchErrorTrace = regexp.MustCompile(`(?m)^Trace:\n`)

func TestIs(t *testing.T) {

	inner1Error := stderr.New("inner error")
	inner2Error := New("inner 2 error")
	wrapperError := Newf("new error include inner error: %s, inner2error: %s", inner1Error, inner2Error)

	// true
	require.True(t, Is(wrapperError, inner1Error))
	require.True(t, Is(wrapperError, inner2Error))
	require.True(t, Is(wrapperError, wrapperError))
	require.False(t, Is(wrapperError, stderr.New("xxx")))

}

func TestDesc(t *testing.T) {
	err := Newf("this is a simple error")
	w1err := Newf("wrapper1 error: %s", err)
	w2err := Newf("wrapper2 error: %s", w1err)
	fmt.Println(Is(w2err, err))
}

func TestError(t *testing.T) {

	testStructureA := TestStructure{
		Name:   "TestA",
		Age:    10,
		Weight: 30.12,
	}

	cases := []struct {
		name string
		desc string
		args []any
	}{
		{
			"no-args-error",
			"no args error string",
			nil,
		},
		{
			"one-args",
			"error type: %q",
			[]any{"no permission"},
		},
		{
			"two-args",
			"error type: %q, desc: %s",
			[]any{"badEvent", "occurred a bad io event"},
		},
		{
			"complex-error",
			"test error format, integer: %d, string: %s, float: %f, struct: %v, wrapString: %q, err: %s",
			[]any{100, "tag", 3.14, testStructureA, "wrap", os.ErrPermission},
		},
	}

	for _, _case := range cases {
		t.Run(_case.name, func(t *testing.T) {
			err := Newf(_case.desc, _case.args...)
			require.Error(t, err)

			expectedErrString := fmt.Sprintf(_case.desc, _case.args...)

			// Error() %s
			require.Equal(t, expectedErrString, fmt.Sprintf("%s", err))

			// Error() %q
			require.Equal(t, fmt.Sprintf("%q", expectedErrString), fmt.Sprintf("%q", err))

			// Error() %v
			//   verify traceback
			tracebackString := fmt.Sprintf("%v", err)
			//     Error: ...
			require.True(t, regxMatchErrorHeader.MatchString(tracebackString))
			//     Trace: ...
			require.True(t, regxMatchErrorTrace.MatchString(tracebackString))

		})
	}
}

type TestStructure struct {
	Name   string
	Age    int
	Weight float64
}

func TestJoinReturnsNil(t *testing.T) {
	if err := Join(); err != nil {
		require.Nil(t, err)
	}
	if err := Join(nil); err != nil {
		require.Nil(t, err)
	}
	if err := Join(nil, nil); err != nil {
		require.Nil(t, err)
	}
}

func TestJoin(t *testing.T) {
	err1 := New("err1")
	err2 := New("err2")
	cases := []struct {
		name   string
		errs   []error
		expect []error
	}{
		{
			"one error",
			[]error{err1},
			[]error{err1},
		},
		{
			"two error",
			[]error{err1, err2},
			[]error{err1, err2},
		},
		{
			"not align error",
			[]error{err1, nil, err2, nil},
			[]error{err1, err2},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := Join(c.errs...).(interface{ Unwrap() []error }).Unwrap()
			require.Equal(t, actual, c.expect)
		})
	}
}

func TestJoinErrorMethod(t *testing.T) {
	err1 := New("err1")
	err2 := New("err2")

	cases := []struct {
		name   string
		errs   []error
		expect string
	}{
		{
			"simple",
			[]error{err1},
			err1.Error(),
		},
		{
			"two",
			[]error{err1, err2},
			"err1\nerr2",
		},
		{
			"contain nil",
			[]error{err1, nil, err2},
			"err1\nerr2",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			errString := Join(c.errs...).Error()
			require.Equal(t, c.expect, errString)
		})
	}
}

func TestSetErrPrefix(t *testing.T) {

	cases := []struct {
		Name   string
		Prefix string
	}{
		{
			"empty-string",
			"",
		},
		{
			"general-string",
			"Error",
		},
		{
			"contain-space-string",
			"meet error",
		},
		{
			"contain-line-string",
			"meet_error",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			old := errPrefix
			SetErrPrefix(c.Prefix)
			require.NotEqual(t, c.Prefix, old)
		})
	}
}

func TestSetErrPrefixf(t *testing.T) {

	old := errPrefix
	SetErrPrefixf("%s err", "program")
	require.NotEqual(t, fmt.Sprintf("%s err", "program"), old)
}

func TestCheckErr(t *testing.T) {

	testErr := Error("test error")
	output := &bytes.Buffer{}

	cases := []struct {
		name   string
		prefix string
		err    error
		expect string
	}{
		{
			"empty error",
			"prefix",
			nil,
			"",
		},
		{
			"test error",
			"prefix",
			testErr,
			"prefix: test error\n",
		},
		{
			"no preifx",
			"",
			testErr,
			"test error\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			SetErrPrefix(c.prefix)
			SetErrOutput(output)
			output.Reset()
			CheckErr(c.err)
			require.Equal(t, c.expect, output.String())
		})
	}
}
