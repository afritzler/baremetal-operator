package bmc

import (
	"context"
	"fmt"

	"github.com/afritzler/baremetal-operator/api/metal/v1alpha1"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

var _ BMC = (*RedfishBMC)(nil)

// RedfishBMC is an implementation of the BMC interface for Redfish.
type RedfishBMC struct {
	systemId string
	client   *gofish.APIClient
}

// NewRedfishBMC creates a new RedfishLocalBMC with the given connection details.
func NewRedfishBMC(ctx context.Context, systemId string, bmcConfig v1alpha1.BMCConfiguration, username, password string) (*RedfishBMC, error) {
	clientConfig := gofish.ClientConfig{
		Endpoint:  bmcConfig.Address,
		Username:  username,
		Password:  password,
		Insecure:  true,
		BasicAuth: bmcConfig.BasicAuth,
	}
	client, err := gofish.ConnectContext(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redfish endpoint: %w", err)
	}
	return &RedfishBMC{systemId: systemId, client: client}, nil
}

// Logout closes the BMC client connection by logging out
func (r *RedfishBMC) Logout() {
	r.client.Logout()
}

// PowerOn powers on the system using Redfish.
func (r *RedfishBMC) PowerOn() error {
	systems, err := r.client.GetService().Systems()
	if err != nil {
		return fmt.Errorf("failed to get systems: %w", err)
	}

	system := getSystemWithSytemID(systems, r.systemId)
	if system == nil {
		return fmt.Errorf("no system found for system ID %s", r.systemId)
	}

	if err := system.Reset(redfish.OnResetType); err != nil {
		return fmt.Errorf("failed to reset system to power on state: %w", err)
	}

	return nil
}

// PowerOff powers off the system using Redfish.
func (r *RedfishBMC) PowerOff() error {
	systems, err := r.client.GetService().Systems()
	if err != nil {
		return fmt.Errorf("failed to get systems: %w", err)
	}

	system := getSystemWithSytemID(systems, r.systemId)
	if system == nil {
		return fmt.Errorf("no system found for system ID %s", r.systemId)
	}

	if err := system.Reset(redfish.GracefulShutdownResetType); err != nil {
		return fmt.Errorf("failed to reset system to graceful shutdown: %w", err)
	}

	return nil
}

// Reset performs a reset on the system using Redfish.
func (r *RedfishBMC) Reset() error {
	// Implementation details...
	return nil
}

// SetPXEBootOnce sets the boot device for the next system boot using Redfish.
func (r *RedfishBMC) SetPXEBootOnce(systemID string) error {
	service := r.client.GetService()

	systems, err := service.Systems()
	if err != nil {
		return fmt.Errorf("failed to get systems: %w", err)
	}

	for _, system := range systems {
		if system.ID == systemID {
			if err := system.SetBoot(redfish.Boot{
				BootSourceOverrideEnabled: redfish.OnceBootSourceOverrideEnabled,
				BootSourceOverrideMode:    redfish.UEFIBootSourceOverrideMode,
				BootSourceOverrideTarget:  redfish.PxeBootSourceOverrideTarget,
			}); err != nil {
				return fmt.Errorf("failed to set the boot order: %w", err)
			}
		}
	}

	return nil
}

// GetSystemInfo retrieves information about the system using Redfish.
func (r *RedfishBMC) GetSystemInfo() (SystemInfo, error) {
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
