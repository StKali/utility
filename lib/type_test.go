package lib

import (
	"github.com/stretchr/testify/require"
	"testing"
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

func TestSize2String(t *testing.T) {
	// normal path test
	s, err := Size2String(KB)
	require.NoError(t, err)
	require.Equal(t, "1.00 KB", s)

	// edge case test
	s, err = Size2String(0)
	require.NoError(t, err)
	require.Equal(t, "0 B", s)

	s, err = Size2String(-1)
	require.Error(t, err)

	s, err = Size2String(EB)
	require.NoError(t, err)
	require.Equal(t, "1.00 EB", s)

	s, err = Size2String(int64(float64(EB) * 1.2))
	require.NoError(t, err)
	require.Equal(t, "1.20 EB", s)

	sizes := []int64{KB, MB, GB, TB, PB, EB}
	labels := []string{"1.00 KB", "1.00 MB", "1.00 GB", "1.00 TB", "1.00 PB", "1.00 EB"}
	// all line test
	for index := range sizes {
		s, err = Size2String(sizes[index])
		require.NoError(t, err)
		require.Equal(t, labels[index], s)
	}
}

func TestString2Size(t *testing.T) {
	// normal path test
	size, err := String2Size("1024 KB")
	require.NoError(t, err)
	require.Equal(t, 1024*KB, size)

	size, err = String2Size("1.1k")
	require.NoError(t, err)

	// edge case test
	size, err = String2Size("")
	require.NoError(t, err)
	require.Equal(t, int64(0), size)

	size, err = String2Size("0")
	require.NoError(t, err)
	require.Equal(t, int64(0), size)

	size, err = String2Size("1 EB")
	require.NoError(t, err)
	require.Equal(t, EB, size)

	// error case test
	size, err = String2Size("-1 k")
	require.Error(t, err)

	size, err = String2Size("0..1001")
	require.Error(t, err)

	_, err = String2Size("1024ABC")
	require.Error(t, err)

	_, err = String2Size("ABC")
	require.Error(t, err)

	// right all line test
	sizes := []string{"1 KB", "1 MB", "1 GB", "1 TB", "1 PB", "1 EB"}
	labels := []int64{KB, MB, GB, TB, PB, EB}
	for index := range sizes {
		size, err = String2Size(sizes[index])
		require.NoError(t, err)
		require.Equal(t, labels[index], size)
	}

	// error all line test
	for index := range sizes {
		size, err = String2Size("-" + sizes[index])
		require.Errorf(t, err, "invalid size ")
	}
}
