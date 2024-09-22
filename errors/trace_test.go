package errors

import (
	"bytes"
	"fmt"
	"regexp"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	// Traceback:
	regxMatchTracebackLine = regexp.MustCompile(`(?m)^Traceback:$`)
	//     /file1/file2/...func(...)
	regxMatchFunctionInfo = regexp.MustCompile(`(?m)^    .*\.\S+\(\.\.\.\)$`)
	//          file1/file2/x.go:111
	regxMatchFileAndLine = regexp.MustCompile(`(?m)^        .*\S+\.go:\d+$`)
)

func checkTracebackFormat(t *testing.T, traceback string) {
	require.True(t, regxMatchTracebackLine.MatchString(traceback))
	require.True(t, regxMatchFunctionInfo.MatchString(traceback))
	require.True(t, regxMatchFileAndLine.MatchString(traceback))
}

func TestTraceStackTrace(t *testing.T) {
	tc := GetTrace(3)
	buf := bytes.Buffer{}
	tc.Traceback(&buf)
	traceback := buf.String()
	checkTracebackFormat(t, traceback)
}

func TestTraceRangeFrames(t *testing.T) {
	regxMatchFile := regexp.MustCompile(`(?m)file: \S+\n`)
	regxMatchFunc := regexp.MustCompile(`(?m)func: \S+\n`)
	regxMatchLine := regexp.MustCompile(`(?m)line: \d+\n`)

	tc := GetTrace(2)
	buf := &bytes.Buffer{}
	tc.RangeFrames(func(frame runtime.Frame) {
		_, _ = fmt.Fprintf(buf, "file: %s\n", frame.File)
		_, _ = fmt.Fprintf(buf, "func: %s\n", frame.Function)
		_, _ = fmt.Fprintf(buf, "line: %d\n", frame.Line)
	})
	outString := buf.String()
	require.True(t, regxMatchFile.MatchString(outString))
	require.True(t, regxMatchFunc.MatchString(outString))
	require.True(t, regxMatchLine.MatchString(outString))

	// clear buffer
	buf.Reset()
	SetErrOutput(buf)
	tc.RangeFrames(nil)
	rangeFramesString := buf.String()
	require.True(t, regxMatchFunctionInfo.MatchString(rangeFramesString))
	require.True(t, regxMatchFileAndLine.MatchString(rangeFramesString))
}

func TestTraceString(t *testing.T) {
	tc := GetTrace(3)
	traceback := tc.String()
	checkTracebackFormat(t, traceback)
}

func TestStackTrace(t *testing.T) {
	buf := &bytes.Buffer{}
	Traceback(buf)
	traceback := buf.String()
	checkTracebackFormat(t, traceback)
}

func TestGetTraceback(t *testing.T) {
	traceback := GetTraceback()
	checkTracebackFormat(t, traceback)
}
