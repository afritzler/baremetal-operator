package bmc

import "github.com/stmcginnis/gofish/redfish"

func getSystemWithSytemID(systems []*redfish.ComputerSystem, id string) *redfish.ComputerSystem {
	for _, system := range systems {
		if system.ID == id {
			return system
		}
	}
	return nil
}
