package tool

import (
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
	"unicode"
)

func setEnv(env []string, key, value string) []string {
	key = strings.TrimFunc(key, unicode.IsSpace) + "="
	newItem := key + strings.TrimFunc(value, unicode.IsSpace)
	for index, item := range env {
		if strings.HasPrefix(item, key) {
			env[index] = newItem
			return env
		}
	}
	env = append(env, newItem)
	return env
}

func TestLightCommandEnviron(t *testing.T) {

	cases := []struct {
		name   string
		env    map[string]string
		expect []string
	}{
		{
			"nil",
			nil,
			os.Environ(),
		},
		{
			"override",
			map[string]string{"PWD": "test path"},
			setEnv(os.Environ(), "PWD", "test path"),
		},
		{
			"add",
			map[string]string{"ADD_ITEM": "add case"},
			setEnv(os.Environ(), "ADD_ITEM", "add case"),
		},
		{
			"override&add",
			map[string]string{"PWD": "test path", "ADD_ITEM": "add case"},
			setEnv(setEnv(os.Environ(), "PWD", "test path"), "ADD_ITEM", "add case"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmd := LightCmd{Env: c.env}
			actual := cmd.Environ()
			require.Equal(t, c.expect, actual)
		})
	}
}
