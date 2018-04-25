package virtualbox_test

import (
	"testing"

	virtualbox "github.com/asnowfix/go-virtualbox"
	"github.com/golang/mock/gomock"
)

func TestNATNets(t *testing.T) {
	Setup(t)

	if ManageMock != nil {
		listHostOnlyIfsOut := ReadTestData("vboxmanage-list-natnets-1.out")
		gomock.InOrder(
			ManageMock.EXPECT().runOut("list", "natnets").Return(listHostOnlyIfsOut, nil).Times(1),
		)
	}
	m, err := virtualbox.NATNets()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", m)

	Teardown()
}
