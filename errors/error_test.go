package errors

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestErrorf(t *testing.T) {
	cases := []struct {
		name   string
		format string
		args   []any
	}{
		{
			"only-format",
			"a simple error",
			[]any{},
		},
	}

	for _, _case := range cases {
		t.Run(_case.name, func(t *testing.T) {
			err := Errorf(_case.format, _case.args...)
			require.Error(t, err)
		})
	}
}

func TestWrap(t *testing.T) {

}
