package paths

import (
	"os"
	"testing"

	"github.com/stkali/utility/errors"
)

var (
	homeDirectory string
	currentDirectory string
)

func TestMain(m *testing.M) {
	var err error
	
	homeDirectory, err = os.UserHomeDir()
	errors.CheckErr(err)

	currentDirectory, err = os.Getwd()
	errors.CheckErr(err)
	
	errors.Exit(m.Run())
}
