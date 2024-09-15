package lib

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMax(t *testing.T) {
	require.Equal(t, Max(1), 1)

	// int
	maxInt := Max(1, 1010, 111)
	require.Equal(t, 1010, maxInt)

	// uint
	maxUint := Max(uint(1), uint(111), uint(100), uint(19))
	require.Equal(t, uint(111), maxUint)

	// int8
	maxInt8 := Max(int8(1), int8(100), int8(111))
	require.Equal(t, int8(111), maxInt8)

	// empty
	require.Equal(t, 0, Max([]int{}...))
}

func TestMin(t *testing.T) {

	require.Equal(t, Min(1), 1)

	// int
	minInt := Min(1, 100, 111)
	require.Equal(t, 1, minInt)

	// uint
	minUint := Min(uint(111), uint(1), uint(100), uint(111))
	require.Equal(t, uint(1), minUint)

	// int8
	minInt8 := Min(int8(122), int8(100), int8(111))
	require.Equal(t, int8(100), minInt8)

	// empty
	require.Equal(t, 0, Min([]int{}...))
}
