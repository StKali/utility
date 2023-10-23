package errors

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"regexp"
	"runtime"
	"testing"
)

var (
	// /file1/file2/...func(...)
	regxMatchFunctionInfo = regexp.MustCompile(`(?m)^\S+.*\.\S+\(\.\.\.\)$`)
	//     file1/file2/x.go:111
	regxMatchFileAndLine = regexp.MustCompile(`(?m)^	\S+.*\S+\.go:\d+$`)
)

// TestStack ...
func TestStack(t *testing.T) {
	trace := GetTrace(2)
	buf := bytes.Buffer{}
	trace.Stack(&buf)
	tracebackString := buf.String()
	require.True(t, regxMatchFunctionInfo.MatchString(tracebackString))
	require.True(t, regxMatchFileAndLine.MatchString(tracebackString))
}

// TestStackWithHandle ...
func TestStackWithHandle(t *testing.T) {
	assertFile := regexp.MustCompile(`(?m)file: \S+\n`)
	assertFunc := regexp.MustCompile(`(?m)func: \S+\n`)
	assertLine := regexp.MustCompile(`(?m)line: \d+\n`)

	trace := GetTrace(2)
	buf := bytes.Buffer{}
	trace.StackWithHandle(func(frame runtime.Frame) {
		_, _ = fmt.Fprintf(&buf, "file: %s\n", frame.File)
		_, _ = fmt.Fprintf(&buf, "func: %s\n", frame.Function)
		_, _ = fmt.Fprintf(&buf, "line: %d\n", frame.Line)
	})

	outString := buf.String()
	require.True(t, assertFile.MatchString(outString))
	require.True(t, assertFunc.MatchString(outString))
	require.True(t, assertLine.MatchString(outString))
}
