package lib

import (
	"github.com/stretchr/testify/require"
	"testing"
)

type Student struct {
	name string
}

func (s *Student) GetName() string {
	return s.name
}

func TestReplace(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		name := "steven"
		func() {
			defer Replace(&name, "kali")()
			require.Equal(t, name, "kali")
		}()
		require.Equal(t, name, "steven")
	})

	t.Run("pointer", func(t *testing.T) {
		name := "steven"
		namePtr := &name
		func() {
			defer Replace(namePtr, "kali")()
			require.Equal(t, *namePtr, "kali")
		}()
		require.Equal(t, name, "steven")
	})

	t.Run("function", func(t *testing.T) {
		getName := func() string { return "steven" }
		func() {
			defer Replace(&getName, func() string { return "kali" })()
			require.Equal(t, getName(), "kali")
		}()
		require.Equal(t, getName(), "steven")
	})

	t.Run("struct", func(t *testing.T) {
		steven := Student{"steven"}
		func() {
			defer Replace(&steven, Student{"kali"})()
			require.Equal(t, steven.GetName(), "kali")
		}()
		require.Equal(t, steven.GetName(), "steven")
	})
}
