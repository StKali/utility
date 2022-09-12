package tool

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

var panicMessage = "test recovery string"

func TestRecovery(t *testing.T) {
	func() {
		defer Recovery(func(e any, exception string) {
			output, ok := e.(string)
			require.True(t, ok)
			require.Equal(t, output, panicMessage)
			require.True(t, strings.Contains(exception, "TestRecovery(...)"))
			path, err := os.Getwd()
			require.NoError(t, err)
			require.True(t, strings.Contains(exception, path))
		})
		panic(panicMessage)
	}()
}

func TestSetDepth(t *testing.T) {
	SetDepth(100)
	require.Equal(t, 100, depth)
}

func TestPrintStack(t *testing.T) {
	PrintStack(3)
}

func TestSaveStack(t *testing.T) {
	defer func() {
		recover()
		buf := new(bytes.Buffer)
		SaveStack(buf, 3)
		require.Equal(t, buf.String(), GetStack(3))
	}()
	panic(panicMessage)
}
