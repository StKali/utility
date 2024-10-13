package errors

import (
	"bytes"
	"fmt"
	"github.com/stkali/utility/lib"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
)

func TestExit(t *testing.T) {
	actualExitCode := 0
	defer ReplaceExit(func(code int) {
		actualExitCode = code
	})()
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
	defer ReplaceExit(func(code int) {
		actualCode = code
	})()

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

func TestSetExit(t *testing.T) {
	SetExit(nil)
	require.True(t, osExit != nil)

	wantCode := 100
	expectCode := 0
	SetExit(func(code int) {
		expectCode = code
	})
	Exit(wantCode)
	require.Equal(t, wantCode, expectCode)

}
