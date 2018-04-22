package virtualbox

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type manageGuestProp struct {
}

func (manageGuestProp) run(args ...string) error {
	return nil
}

func (manageGuestProp) runOut(args ...string) (string, error) {
	return "Value: test_val", nil
}

func (manageGuestProp) runOutErr(args ...string) (string, string, error) {
	return "", "", nil
}

// func init() {
// 	var ok = os.Getenv("TEST_FAKE_VBM")
// 	if len(ok) > 0 {
// 		Manage = new(manageGuestProp)
// 	}
// }

func TestGuestProperty(t *testing.T) {
	require := require.New(t)

	var vm = os.Getenv("TEST_VM")
	if len(vm) <= 0 {
		vm = "go-virtualbox"
		t.Logf("Missing TEST_VM environment variable")
	}
	t.Logf("Using '%s'", vm)

	err := SetGuestProperty(vm, "test_key", "test_val")
	require.Nil(err, "failed setting guestproperty")
	if Verbose {
		t.Logf("OK SetGuestProperty test_key=test_val")
	}

	val, err := GetGuestProperty(vm, "test_key")
	require.Nil(err, "failed getting guestproperty")
	t.Logf("val='%s'", val)
	require.Equal(val, "test_val", "guestproperty not set to the proper value")
	if Verbose {
		t.Logf("OK GetGuestProperty test_key=test_val")
	}

	// Now deletes it...
	err = DeleteGuestProperty(vm, "test_key")
	require.Nil(err, "failed deleting guestproperty")
	if Verbose {
		t.Logf("OK DeleteGuestProperty test_key")
	}

	// ...and check that it is  no longer readable
	_, err = GetGuestProperty(vm, "test_key")
	require.NotNil(err, "guestproperty not deleted")

	if Verbose {
		t.Logf("OK GetGuestProperty test_key=empty")
	}

}
