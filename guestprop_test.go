package virtualbox

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGuestProperty(t *testing.T) {
	var vm = os.Getenv("TEST_VM")
	if len(vm) <= 0 {
		vm = "go-virtualbox"
		t.Logf("Missing TEST_VM environment variable")
	}
	t.Logf("Using '%s'", vm)

	err := SetGuestProperty(vm, "test_key", "test_val")
	assert.Nil(t, err, "failed setting guestproperty")
	if Verbose {
		t.Logf("OK SetGuestProperty test_key=test_val")
	}

	val, err := GetGuestProperty(vm, "test_key")
	assert.Nil(t, err, "failed getting guestproperty")
	t.Logf("val='%s'", val)
	assert.Equal(t, val, "test_val", "guestproperty not set to the proper value")
	if Verbose {
		t.Logf("OK GetGuestProperty test_key=test_val")
	}

	// Now deletes it...
	err = DeleteGuestProperty(vm, "test_key")
	assert.Nil(t, err, "failed deleting guestproperty")
	if Verbose {
		t.Logf("OK DeleteGuestProperty test_key")
	}

	// ...and check that it is  no longer readable
	_, err = GetGuestProperty(vm, "test_key")
	assert.NotNil(t, err, "guestproperty not deleted")

	if Verbose {
		t.Logf("OK GetGuestProperty test_key=empty")
	}

}
