package errors

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWarning(t *testing.T) {

	cases := []struct {
		name    string
		warning []any
		expect  string
		prefix  string
	}{
		{
			"no prefix",
			[]any{"this is warning"},
			"this is warning\n",
			"",
		},
		{
			"prefix",
			[]any{"this is warning"},
			"prefix: this is warning\n",
			"prefix",
		},
		{
			"type int",
			[]any{100},
			"warning: 100\n",
			"warning",
		},
		{
			"type point",
			[]any{&struct{}{}},
			"warning: &{}\n",
			"warning",
		},
		{
			"type 2 point",
			[]any{&struct{}{}, nil},
			"warning: &{} <nil>\n",
			"warning",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer
			SetWarningOutput(&out)
			SetWarningPrefix(c.prefix)
			Warning(c.warning...)
			require.Equal(t, c.expect, out.String())
		})
	}
}

func TestDisableWarning(t *testing.T) {
	var out bytes.Buffer
	SetWarningOutput(&out)
	DisableWarning()
	defer func() {
		disableWarning = false
	}()
	Warning("test warning string")
	require.Equal(t, out.String(), "")
	out.Reset()
	Warningf("age: %d", 18)
	require.Equal(t, out.String(), "")
}

func TestWarningf(t *testing.T) {
	cases := []struct {
		name   string
		format string
		args   []any
		prefix string
	}{
		{
			"only format",
			"format",
			[]any{},
			"prefix: ",
		},
		{
			"one param",
			"name: %s",
			[]any{"monkey"},
			"prefix: ",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer
			SetWarningOutput(&out)
			SetWarningPrefix(c.prefix)
			Warningf(c.format, c.args...)
			payload := fmt.Sprintf(c.format, c.args...)
			expect := fmt.Sprintf("%s: %s\n", c.prefix, payload)
			actual := out.String()
			require.Equal(t, expect, actual)
		})
	}

}

func TestSetWarningPrefixf(t *testing.T) {

	SetWarningPrefixf("%s warnings", "name")
	writer := &bytes.Buffer{}
	SetWarningOutput(writer)
	warningMsg := "this is warning message"
	Warning(warningMsg)
	require.Equal(t, fmt.Sprintf("name warnings: %s\n", warningMsg), writer.String())
}
