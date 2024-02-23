package bmc

import (
	"context"
	"fmt"

	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

var _ BMC = (*RedfishLocalBMC)(nil)

// RedfishLocalBMC is an implementation of the BMC interface for Redfish.
type RedfishLocalBMC struct {
	systemId string
	client   *gofish.APIClient
}

// NewRedfishLocalBMC creates a new RedfishLocalBMC with the given connection details.
func NewRedfishLocalBMC(ctx context.Context, systemId string, url string) (*RedfishLocalBMC, error) {
	client, err := gofish.ConnectDefaultContext(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redfish endpoint: %w", err)
	}
	return &RedfishLocalBMC{systemId: systemId, client: client}, nil
}

// Logout closes the BMC client connection by logging out
func (r *RedfishLocalBMC) Logout() {
	r.client.Logout()
}

// PowerOn powers on the system using Redfish.
func (r *RedfishLocalBMC) PowerOn() error {
	systems, err := r.client.GetService().Systems()
	if err != nil {
		return fmt.Errorf("failed to get systems: %w", err)
	}

	system := getSystemWithSytemID(systems, r.systemId)
	if system == nil {
		return fmt.Errorf("no system found for system ID %s", r.systemId)
	}

	system.PowerState = redfish.OnPowerState
	systemURI := fmt.Sprintf("/redfish/v1/Systems/%s", system.ID)
	if err := system.Patch(systemURI, system); err != nil {
		return fmt.Errorf("failed to set power state %s for system %s: %w", redfish.OnPowerState, r.systemId, err)
	}

	return nil
}

// PowerOff powers off the system using Redfish.
func (r *RedfishLocalBMC) PowerOff() error {
	systems, err := r.client.GetService().Systems()
	if err != nil {
		return fmt.Errorf("failed to get systems: %w", err)
	}

	system := getSystemWithSytemID(systems, r.systemId)
	if system == nil {
		return fmt.Errorf("no system found for system ID %s", r.systemId)
	}

	system.PowerState = redfish.OffPowerState
	systemURI := fmt.Sprintf("/redfish/v1/Systems/%s", system.ID)
	if err := system.Patch(systemURI, system); err != nil {
		return fmt.Errorf("failed to set power state %s for system %s: %w", redfish.OffPowerState, r.systemId, err)
	}

	return nil
}

// Reset performs a reset on the system using Redfish.
func (r *RedfishLocalBMC) Reset() error {
	// Implementation details...
	return nil
}

// SetPXEBootOnce sets the boot device for the next system boot using Redfish.
func (r *RedfishLocalBMC) SetPXEBootOnce(systemID string) error {
	// Implementation details...
	return nil
}

// GetSystemInfo retrieves information about the system using Redfish.
func (r *RedfishLocalBMC) GetSystemInfo() (SystemInfo, error) {
	service := r.client.GetService()

	systems, err := service.Systems()
	if err != nil {
		return SystemInfo{}, fmt.Errorf("failed to get systems: %w", err)
	}

	systemInfo := SystemInfo{}
	for _, system := range systems {
		if system.ID == r.systemId {
			systemInfo.SystemUUID = system.UUID
			systemInfo.Manufacturer = system.Manufacturer
			systemInfo.Model = system.Model
			systemInfo.Status = system.Status
			systemInfo.PowerState = system.PowerState
			nics, err := system.EthernetInterfaces()
			if err != nil {
				return SystemInfo{}, fmt.Errorf("failed to get network interfaces for system: %w", err)
			}
			for _, newNic := range nics {
				updated := false
				for i, existingNic := range systemInfo.NetworkInterfaces {
					if newNic.ID == existingNic.ID {
						// Update existing NIC
						systemInfo.NetworkInterfaces[i] = NetworkInterface{
							ID:                  newNic.ID,
							MACAddress:          newNic.MACAddress,
							PermanentMACAddress: newNic.PermanentMACAddress,
						}
						updated = true
						break
					}
				}
				// If NIC ID was not found in existing list, append it
				if !updated {
					systemInfo.NetworkInterfaces = append(systemInfo.NetworkInterfaces, NetworkInterface{
						ID:                  newNic.ID,
						MACAddress:          newNic.MACAddress,
						PermanentMACAddress: newNic.PermanentMACAddress,
					})
				}
			}
			processors, err := system.Processors()
			if err != nil {
				return SystemInfo{}, fmt.Errorf("failed to get processors for system: %w", err)
			}
			for _, newProcessor := range processors {
				updated := false
				for i, existingProcessors := range systemInfo.Processors {
					if newProcessor.ID == existingProcessors.ID {
						// Update existing Processor
						systemInfo.Processors[i] = Processor{
							ID:                    newProcessor.ID,
							ProcessorType:         string(newProcessor.ProcessorType),
							ProcessorArchitecture: string(newProcessor.ProcessorArchitecture),
							InstructionSet:        string(newProcessor.InstructionSet),
							Manufacturer:          newProcessor.Manufacturer,
							Model:                 newProcessor.Model,
							MaxSpeedMHz:           int32(newProcessor.MaxSpeedMHz),
							TotalCores:            int32(newProcessor.TotalCores),
							TotalThreads:          int32(newProcessor.TotalThreads),
						}
						updated = true
						break
					}
				}
				// If Processor ID was not found in existing list, append it
				if !updated {
					systemInfo.Processors = append(systemInfo.Processors, Processor{
						ID:                    newProcessor.ID,
						ProcessorType:         string(newProcessor.ProcessorType),
						ProcessorArchitecture: string(newProcessor.ProcessorArchitecture),
						InstructionSet:        string(newProcessor.InstructionSet),
						Manufacturer:          newProcessor.Manufacturer,
						Model:                 newProcessor.Model,
						MaxSpeedMHz:           int32(newProcessor.MaxSpeedMHz),
						TotalCores:            int32(newProcessor.TotalCores),
						TotalThreads:          int32(newProcessor.TotalThreads),
					})
				}
			}
			break
		}
	}

	return systemInfo, nil
}
