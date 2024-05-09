package errors

import (
	"testing"

	"github.com/stkali/utility/tool"
)

func TestMain(m *testing.M) {
	preExit := tool.Exit
	tool.Exit = func(code int) {}
	code := m.Run()
	preExit(code)
}
