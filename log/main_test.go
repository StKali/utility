package log

import (
	"testing"
)

func TestMain(m *testing.M) {
	preExit := Exit
	Exit = func(code int) {}
	code := m.Run()
	preExit(code)
}
