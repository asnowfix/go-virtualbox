package virtualbox_test

import (
	"testing"

	virtualbox "github.com/asnowfix/go-virtualbox"
	"github.com/golang/mock/gomock"
)

func TestDHCPs(t *testing.T) {
	Setup(t)

	if ManageMock != nil {
		listDhcpServersOut := ReadTestData("vboxmanage-list-dhcpservers-1.out")
		gomock.InOrder(
			ManageMock.EXPECT().runOut("list", "dhcpservers").Return(listDhcpServersOut, nil).Times(1),
		)
	}
	m, err := virtualbox.DHCPs()
	if err != nil {
		t.Fatal(err)
	}

	for _, dhcp := range m {
		t.Logf("%+v", dhcp)
	}

	Teardown()
}
