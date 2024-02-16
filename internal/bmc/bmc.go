package bmc

import (
	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
)

// BMC defines an interface for interacting with a Baseboard Management Controller.
type BMC interface {
	// PowerOn powers on the system.
	PowerOn() error

	// PowerOff powers off the system.
	PowerOff() error

	// Reset performs a reset on the system.
	Reset() error

	// SetPXEBootOnce sets the boot device for the next system boot.
	SetPXEBootOnce(systemID string) error

	// GetSystemInfo retrieves information about the system.
	GetSystemInfo() (SystemInfo, error)
}

type NetworkInterface struct {
	ID                  string
	MACAddress          string
	PermanentMACAddress string
}

type Processor struct {
	ID                    string
	ProcessorType         string
	ProcessorArchitecture string
	InstructionSet        string
	Manufacturer          string
	Model                 string
	MaxSpeedMHz           int32
	TotalCores            int32
	TotalThreads          int32
}

// SystemInfo represents basic information about the system.
type SystemInfo struct {
	Manufacturer      string
	Model             string
	Status            common.Status
	PowerState        redfish.PowerState
	NetworkInterfaces []NetworkInterface
	Processors        []Processor
}
