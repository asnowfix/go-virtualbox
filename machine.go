package virtualbox

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MachineState stores the last retrieved VM state.
type MachineState string

const (
	// Poweroff is a MachineState value.
	Poweroff = MachineState("poweroff")
	// Running is a MachineState value.
	Running = MachineState("running")
	// Paused is a MachineState value.
	Paused = MachineState("paused")
	// Saved is a MachineState value.
	Saved = MachineState("saved")
	// Aborted is a MachineState value.
	Aborted = MachineState("aborted")
)

// Flag is an active VM configuration toggle
type Flag int

// Flag names in lowercases to be consistent with VBoxManage options.
const (
	acpi Flag = 1 << iota
	ioapic
	rtcuseutc
	cpuhotplug
	pae
	longmode
	synthcpu
	hpet
	hwvirtex
	triplefaultreset
	nestedpaging
	largepages
	vtxvpid
	vtxux
	accelerate3d
)

// Convert bool to "on"/"off"
func bool2string(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// Get tests if flag is set. Return "on" or "off".
func (f Flag) Get(o Flag) string {
	return bool2string(f&o == o)
}

// Machine information.
type Machine struct {
	Name       string
	UUID       string
	State      MachineState
	CPUs       uint
	Memory     uint // main memory (in MB)
	VRAM       uint // video memory (in MB)
	CfgFile    string
	BaseFolder string
	OSType     string
	Flag       Flag
	BootOrder  []string // max 4 slots, each in {none|floppy|dvd|disk|net}
}

// Refresh reloads the machine information.
func (m *Machine) Refresh() error {
	id := m.Name
	if id == "" {
		id = m.UUID
	}
	mm, err := GetMachine(id)
	if err != nil {
		return err
	}
	*m = *mm
	return nil
}

// Start starts the machine.
func (m *Machine) Start() error {
	switch m.State {
	case Paused:
		return Manage().run("controlvm", m.Name, "resume")
	case Poweroff, Saved, Aborted:
		return Manage().run("startvm", m.Name, "--type", "headless")
	}
	return nil
}

// DisconnectSerialPort sets given serial port to disconnected.
func (m *Machine) DisconnectSerialPort(portNumber int) error {
	return Manage().run("modifyvm", m.Name, fmt.Sprintf("--uartmode%d", portNumber), "disconnected")
}

// Save suspends the machine and saves its state to disk.
func (m *Machine) Save() error {
	switch m.State {
	case Paused:
		if err := m.Start(); err != nil {
			return err
		}
	case Poweroff, Aborted, Saved:
		return nil
	}
	return Manage().run("controlvm", m.Name, "savestate")
}

// Pause pauses the execution of the machine.
func (m *Machine) Pause() error {
	switch m.State {
	case Paused, Poweroff, Aborted, Saved:
		return nil
	}
	return Manage().run("controlvm", m.Name, "pause")
}

// Stop gracefully stops the machine.
func (m *Machine) Stop() error {
	switch m.State {
	case Poweroff, Aborted, Saved:
		return nil
	case Paused:
		if err := m.Start(); err != nil {
			return err
		}
	}

	for m.State != Poweroff { // busy wait until the machine is stopped
		if err := Manage().run("controlvm", m.Name, "acpipowerbutton"); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
		if err := m.Refresh(); err != nil {
			return err
		}
	}
	return nil
}

// Poweroff forcefully stops the machine. State is lost and might corrupt the disk image.
func (m *Machine) Poweroff() error {
	switch m.State {
	case Poweroff, Aborted, Saved:
		return nil
	}
	return Manage().run("controlvm", m.Name, "poweroff")
}

// Restart gracefully restarts the machine.
func (m *Machine) Restart() error {
	switch m.State {
	case Paused, Saved:
		if err := m.Start(); err != nil {
			return err
		}
	}
	if err := m.Stop(); err != nil {
		return err
	}
	return m.Start()
}

// Reset forcefully restarts the machine. State is lost and might corrupt the disk image.
func (m *Machine) Reset() error {
	switch m.State {
	case Paused, Saved:
		if err := m.Start(); err != nil {
			return err
		}
	}
	return Manage().run("controlvm", m.Name, "reset")
}

// Delete deletes the machine and associated disk images.
func (m *Machine) Delete() error {
	if err := m.Poweroff(); err != nil {
		return err
	}
	return Manage().run("unregistervm", m.Name, "--delete")
}

// GetMachine finds a machine by its name or UUID.
func GetMachine(id string) (*Machine, error) {
	stdout, stderr, err := Manage().runOutErr("showvminfo", id, "--machinereadable")
	if err != nil {
		if reMachineNotFound.FindString(stderr) != "" {
			return nil, ErrMachineNotExist
		}
		return nil, err
	}
	s := bufio.NewScanner(strings.NewReader(stdout))
	m := &Machine{}
	for s.Scan() {
		res := reVMInfoLine.FindStringSubmatch(s.Text())
		if res == nil {
			continue
		}
		key := res[1]
		if key == "" {
			key = res[2]
		}
		val := res[3]
		if val == "" {
			val = res[4]
		}

		switch key {
		case "name":
			m.Name = val
		case "UUID":
			m.UUID = val
		case "VMState":
			m.State = MachineState(val)
		case "memory":
			n, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return nil, err
			}
			m.Memory = uint(n)
		case "cpus":
			n, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return nil, err
			}
			m.CPUs = uint(n)
		case "vram":
			n, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return nil, err
			}
			m.VRAM = uint(n)
		case "CfgFile":
			m.CfgFile = val
			m.BaseFolder = filepath.Dir(val)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

// ListMachines lists all registered machines.
func ListMachines() ([]*Machine, error) {
	out, err := Manage().runOut("list", "vms")
	if err != nil {
		return nil, err
	}
	ms := []*Machine{}
	s := bufio.NewScanner(strings.NewReader(out))
	for s.Scan() {
		res := reVMNameUUID.FindStringSubmatch(s.Text())
		if res == nil {
			continue
		}
		m, err := GetMachine(res[1])
		if err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return ms, nil
}

// CreateMachine creates a new machine. If basefolder is empty, use default.
func CreateMachine(name, basefolder string) (*Machine, error) {
	if name == "" {
		return nil, fmt.Errorf("machine name is empty")
	}

	// Check if a machine with the given name already exists.
	ms, err := ListMachines()
	if err != nil {
		return nil, err
	}
	for _, m := range ms {
		if m.Name == name {
			return nil, ErrMachineExist
		}
	}

	// Create and register the machine.
	args := []string{"createvm", "--name", name, "--register"}
	if basefolder != "" {
		args = append(args, "--basefolder", basefolder)
	}
	if err = Manage().run(args...); err != nil {
		return nil, err
	}

	m, err := GetMachine(name)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Modify changes the settings of the machine.
func (m *Machine) Modify() error {
	args := []string{"modifyvm", m.Name,
		"--firmware", "bios",
		"--bioslogofadein", "off",
		"--bioslogofadeout", "off",
		"--bioslogodisplaytime", "0",
		"--biosbootmenu", "disabled",

		"--ostype", m.OSType,
		"--cpus", fmt.Sprintf("%d", m.CPUs),
		"--memory", fmt.Sprintf("%d", m.Memory),
		"--vram", fmt.Sprintf("%d", m.VRAM),

		"--acpi", m.Flag.Get(acpi),
		"--ioapic", m.Flag.Get(ioapic),
		"--rtcuseutc", m.Flag.Get(rtcuseutc),
		"--cpuhotplug", m.Flag.Get(cpuhotplug),
		"--pae", m.Flag.Get(pae),
		"--longmode", m.Flag.Get(longmode),
		"--synthcpu", m.Flag.Get(synthcpu),
		"--hpet", m.Flag.Get(hpet),
		"--hwvirtex", m.Flag.Get(hwvirtex),
		"--triplefaultreset", m.Flag.Get(triplefaultreset),
		"--nestedpaging", m.Flag.Get(nestedpaging),
		"--largepages", m.Flag.Get(largepages),
		"--vtxvpid", m.Flag.Get(vtxvpid),
		"--vtxux", m.Flag.Get(vtxux),
		"--accelerate3d", m.Flag.Get(accelerate3d),
	}

	for i, dev := range m.BootOrder {
		if i > 3 {
			break // Only four slots `--boot{1,2,3,4}`. Ignore the rest.
		}
		args = append(args, fmt.Sprintf("--boot%d", i+1), dev)
	}
	if err := Manage().run(args...); err != nil {
		return err
	}
	return m.Refresh()
}

// AddNATPF adds a NAT port forarding rule to the n-th NIC with the given name.
func (m *Machine) AddNATPF(n int, name string, rule PFRule) error {
	return Manage().run("controlvm", m.Name, fmt.Sprintf("natpf%d", n),
		fmt.Sprintf("%s,%s", name, rule.Format()))
}

// DelNATPF deletes the NAT port forwarding rule with the given name from the n-th NIC.
func (m *Machine) DelNATPF(n int, name string) error {
	return Manage().run("controlvm", m.Name, fmt.Sprintf("natpf%d", n), "delete", name)
}

// SetNIC set the n-th NIC.
func (m *Machine) SetNIC(n int, nic NIC) error {
	args := []string{"modifyvm", m.Name,
		fmt.Sprintf("--nic%d", n), string(nic.Network),
		fmt.Sprintf("--nictype%d", n), string(nic.Hardware),
		fmt.Sprintf("--cableconnected%d", n), "on",
	}

	if nic.Network == "hostonly" {
		args = append(args, fmt.Sprintf("--hostonlyadapter%d", n), nic.HostonlyAdapter)
	}
	return Manage().run(args...)
}

// AddStorageCtl adds a storage controller with the given name.
func (m *Machine) AddStorageCtl(name string, ctl StorageController) error {
	args := []string{"storagectl", m.Name, "--name", name}
	if ctl.SysBus != "" {
		args = append(args, "--add", string(ctl.SysBus))
	}
	if ctl.Ports > 0 {
		args = append(args, "--portcount", fmt.Sprintf("%d", ctl.Ports))
	}
	if ctl.Chipset != "" {
		args = append(args, "--controller", string(ctl.Chipset))
	}
	args = append(args, "--hostiocache", bool2string(ctl.HostIOCache))
	args = append(args, "--bootable", bool2string(ctl.Bootable))
	return Manage().run(args...)
}

// DelStorageCtl deletes the storage controller with the given name.
func (m *Machine) DelStorageCtl(name string) error {
	return Manage().run("storagectl", m.Name, "--name", name, "--remove")
}

// AttachStorage attaches a storage medium to the named storage controller.
func (m *Machine) AttachStorage(ctlName string, medium StorageMedium) error {
	return Manage().run("storageattach", m.Name, "--storagectl", ctlName,
		"--port", fmt.Sprintf("%d", medium.Port),
		"--device", fmt.Sprintf("%d", medium.Device),
		"--type", string(medium.DriveType),
		"--medium", medium.Medium,
	)
}

// CloneMachine clones the given machine name into a new one.
func CloneMachine(baseImageName string, newImageName string, register bool) error {
	if register {
		return Manage().run("clonevm", baseImageName, "--name", newImageName, "--register")
	}
	return Manage().run("clonevm", baseImageName, "--name", newImageName)
}
