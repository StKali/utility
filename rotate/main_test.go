package rotate

import (
	"github.com/stkali/utility/errors"
	"github.com/stkali/utility/log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.TRACE)
	log.SetOutput(os.Stdout)
	errors.Exit(m.Run())
}
