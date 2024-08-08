package lib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test2String(t *testing.T) {
	cases := []struct {
		Name   string
		Bytes  []byte
		Expect string
	}{
		{
			"empty",
			[]byte{},
			"",
		},
		{
			"integer",
			[]byte("1"),
			"1",
		},

		{
			"return",
			[]byte("\r"),
			"\r",
		},

		{
			"newline",
			[]byte("\n"),
			"\n",
		},
		{
			"other",
			[]byte("\r\n928176\tasljh\tt"),
			"\r\n928176\tasljh\tt",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			require.Equal(t, c.Expect, ToString(c.Bytes))
		})
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			require.Equal(t, c.Bytes, ToBytes(c.Expect))
		})
	}
}
