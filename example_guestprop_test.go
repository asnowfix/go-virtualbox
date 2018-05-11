package virtualbox_test

import (
	"log"
	"time"

	virtualbox "github.com/asnowfix/go-virtualbox"
)

var VM = "MyVM"

func ExampleSetGuestProperty() {
	err := virtualbox.SetGuestProperty(VM, "test_name", "test_val")
	if err != nil {
		panic(err)
	}
}

func ExampleGetGuestProperty() {
	err := virtualbox.SetGuestProperty(VM, "test_name", "test_val")
	if err != nil {
		panic(err)
	}
	val, err := virtualbox.GetGuestProperty(VM, "test_name")
	if err != nil {
		panic(err)
	}
	log.Println("val:", val)
}

func ExampleDeleteGuestProperty() {
	err := virtualbox.SetGuestProperty(VM, "test_name", "test_val")
	if err != nil {
		panic(err)
	}
	err = virtualbox.DeleteGuestProperty(VM, "test_name")
	if err != nil {
		panic(err)
	}
}

func ExampleWaitGuestProperty() {

	go func() {
		second := time.Second
		time.Sleep(1 * second)
		virtualbox.SetGuestProperty(VM, "test_name", "test_val")
	}()

	name, val, err := virtualbox.WaitGuestProperty(VM, "test_*")
	if err != nil {
		panic(err)
	}
	log.Println("name:", name, ", value:", val)
}

func ExampleWaitGuestProperties() {
	go func() {
		second := time.Second

		time.Sleep(1 * second)
		virtualbox.SetGuestProperty(VM, "test_name", "test_val1")

		time.Sleep(1 * second)
		virtualbox.SetGuestProperty(VM, "test_name", "test_val2")

		time.Sleep(1 * second)
		virtualbox.SetGuestProperty(VM, "test_name", "test_val1")
	}()

	props := "test_*"
	propsC, doneC, wg := virtualbox.WaitGetProperties(VM, props)

	ok := true
	left := 3
	for ; ok && left > 0; left-- {
		var prop virtualbox.GuestProperty
		prop, ok = <-propsC
		log.Println("name:", prop.Name, ", value:", prop.Value)
	}

	doneC <- true // close channel
	wg.Wait()     // wait for gorouting
}