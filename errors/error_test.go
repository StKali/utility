package errors

import (
	"bytes"
	stderr "errors"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stkali/utility/lib"
)

var regxMatchErrorHeader = regexp.MustCompile(`(?m)^Error: .*`)
var regxMatchErrorTrace = regexp.MustCompile(`(?m)^Traceback:\n`)

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

func TestExit(t *testing.T) {
	originExit := osExit
	defer func() {
		osExit = originExit
	}()
	actualExitCode := 0
	mockExit := func(code int) {
		actualExitCode = code
	}
	osExit = mockExit
	wantExitCode := 100
	SetExitHook(func(code int, msg string, tracer Tracer) {
		require.Equal(t, msg, "")
		require.NotNil(t, tracer)
	})
	defer SetExitHook(nil)
	Exit(wantExitCode)
	require.Equal(t, wantExitCode, actualExitCode)
}

func TestExitf(t *testing.T) {

	// mock exit function
	var actualCode int
	var actualMessage string
	var wantCode int
	var wantMessage string
	mockExit := func(code int) {
		actualCode = code
	}
	oldExit := osExit
	osExit = mockExit
	defer func() { osExit = oldExit }()

	t.Run("errPrefix", func(t *testing.T) {
		// prefix == ""
		buf := &bytes.Buffer{}
		// ensure prefix is empty
		SetErrPrefix("")
		SetErrOutput(buf)
		wantCode = rand.Intn(255)
		wantMessage = lib.RandInternalString(8, 24)
		Exitf(wantCode, wantMessage)
		require.Equal(t, wantCode, actualCode)
		require.Equal(t, wantMessage, buf.String())

		// prefix != ""
		buf.Reset()
		prefix := "test prefix"
		SetErrPrefix("test prefix")
		wantCode = rand.Intn(255)
		wantMessage = lib.RandInternalString(8, 24)
		Exitf(wantCode, wantMessage)
		require.Equal(t, wantCode, actualCode)
		require.Equal(t, fmt.Sprintf("%s: %s", prefix, wantMessage), buf.String())

	})

	t.Run("exit hook", func(t *testing.T) {
		buf := &bytes.Buffer{}
		SetErrPrefix("")
		var actualTracer Tracer
		SetExitHook(func(code int, msg string, tracer Tracer) {
			actualCode = code
			actualMessage = msg
			actualTracer = tracer
		})
		SetErrOutput(buf)
		wantCode = rand.Intn(255)
		wantMessage = lib.RandInternalString(8, 24)
		Exitf(wantCode, wantMessage)
		require.Equal(t, wantCode, actualCode)
		require.Equal(t, wantMessage, actualMessage)
		require.NotNil(t, actualTracer)
	})

}

func TestSetExitHook(t *testing.T) {
	originHook := exitHook
	defer func() {
		SetExitHook(originHook)
	}()
	wantMsg := lib.RandInternalString(8, 16)
	wantTracer := GetTrace(3)
	wantExitCode := 100
	var actualMsg string
	var actualTracer Tracer
	var actualExitCode int
	hook := func(code int, msg string, tracer Tracer) {
		actualExitCode = code
		actualMsg = msg
		actualTracer = tracer
	}
	SetExitHook(hook)
	require.NotNil(t, exitHook)
	exitHook(wantExitCode, wantMsg, wantTracer)
	require.Equal(t, wantExitCode, actualExitCode)
	require.Equal(t, wantMsg, actualMsg)
	require.Equal(t, wantTracer, actualTracer)
}

func TestSetErrPrefix(t *testing.T) {
	originErrPrefix := errPrefix
	defer func() {
		SetErrPrefix(originErrPrefix)
	}()
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
			SetErrPrefix(c.Prefix)
			require.Equal(t, c.Prefix, errPrefix)
		})
	}
}

func TestSetErrPrefixf(t *testing.T) {

	originErrPrefix := errPrefix
	defer func() {
		SetErrPrefix(originErrPrefix)
	}()
	SetErrPrefixf("%s err", "program")
	prefix := fmt.Sprintf("%s err", "program")
	require.Equal(t, errPrefix, prefix)
}

func TestCheckErr(t *testing.T) {

	testError := Error("test error")
	testIError := Newf("with tracer error")
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
			testError,
			"prefix: test error\n",
		},
		{
			"no preifx",
			"",
			testError,
			"test error\n",
		},
		{
			"with tracer error",
			"prefix",
			testIError,
			"prefix: with tracer error\n",
		},
	}
	var wantExitCode int
	originExit := osExit
	mockExit := func(code int) { wantExitCode = code }
	osExit = mockExit
	defer func() { osExit = originExit }()

	originErrPrefix := errPrefix
	defer func() {
		SetErrPrefix(originErrPrefix)
	}()

	originOutput := errOutput
	SetErrOutput(output)
	defer func() {
		SetErrOutput(originOutput)
	}()

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			SetErrPrefix(c.prefix)
			output.Reset()
			CheckErr(c.err)
			if c.err != nil {
				require.Equal(t, wantExitCode, 1)
			}
			require.Equal(t, c.expect, output.String())
		})
	}
}

func TestDesc(t *testing.T) {
	err := Newf("this is a simple error")
	w1err := Newf("wrapper1 error: %s", err)
	w2err := Newf("wrapper2 error: %s", w1err)
	require.True(t, Is(w2err, err))
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
