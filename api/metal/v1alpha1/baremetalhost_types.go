/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// BareMetalHostClaimFinalizer is the finalizer for BareMetalHostClaim.
	BareMetalHostClaimFinalizer = "metal.afritzler.github.io/baremetalhostclaim"
)

type BMCType string

const (
	BMCTypeRedfish      BMCType = "Redfish"
	BMCTypeRedfishLocal BMCType = "RedfishLocal"
)

type BMCConfiguration struct {
	Type      BMCType            `json:"type"`
	Address   string             `json:"address"`
	SecretRef v1.SecretReference `json:"secretRef"`
}

type PowerState string

const (
	PowerStateOn      PowerState = "On"
	PowerStateOff     PowerState = "Off"
	PowerStateUnknown PowerState = "Unknown"
)

// BareMetalHostSpec defines the desired state of BareMetalHost
type BareMetalHostSpec struct {
	SystemID string `json:"systemId"`
	// TODO: remove this later as this is a dummy code
	FooUUID  string              `json:"fooUuid,omitempty"`
	Power    PowerState          `json:"power"`
	ClaimRef *v1.ObjectReference `json:"claimRef,omitempty"`
	BMC      BMCConfiguration    `json:"bmc"`
	// +kubebuilder:validation:Pattern=`[0-9a-fA-F]{2}(:[0-9a-fA-F]{2}){5}`
	BootMACAddress string `json:"bootMACAddress,omitempty"`
}

type Phase string

const (
	PhaseBound   Phase = "Bound"
	PhaseUnbound Phase = "Unbound"
)

type HostState string

const (
	StateInitial     HostState = "Initial"
	StateAvailable   HostState = "Available"
	StateTainted     HostState = "Tainted"
	StateReserved    HostState = "Reserved"
	StateMaintenance HostState = "Maintenance"
)

type NetworkInterface struct {
	ID                  string `json:"id"`
	MACAddress          string `json:"macAddress,omitempty"`
	PermanentMACAddress string `json:"permanentMacAddress,omitempty"`
}

type Processor struct {
	ID                    string `json:"id"`
	ProcessorType         string `json:"processorType,omitempty"`
	ProcessorArchitecture string `json:"processorArchitecture,omitempty"`
	InstructionSet        string `json:"instructionSet,omitempty"`
	Manufacturer          string `json:"manufacturer,omitempty"`
	Model                 string `json:"model,omitempty"`
	MHz                   int32  `json:"mhz,omitempty"`
	Cores                 int32  `json:"cores,omitempty"`
	Threads               int32  `json:"threads,omitempty"`
}

// BareMetalHostStatus defines the observed state of BareMetalHost
type BareMetalHostStatus struct {
	Manufacturer      string             `json:"manufacturer,omitempty"`
	Model             string             `json:"model,omitempty"`
	SerialNumber      string             `json:"serialNumber,omitempty"`
	FirmwareVersion   string             `json:"firmwareVersion,omitempty"`
	PowerState        redfish.PowerState `json:"powerState,omitempty"`
	Health            common.Health      `json:"health,omitempty"`
	SystemState       common.State       `json:"systemState,omitempty"`
	Phase             Phase              `json:"phase,omitempty"`
	State             HostState          `json:"state,omitempty"`
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces,omitempty"`
	Processors        []Processor        `json:"processors"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster,shortName=host

// BareMetalHost is the Schema for the baremetalhosts API
// +kubebuilder:printcolumn:name="Manufacturer",type="string",JSONPath=".status.manufacturer"
// +kubebuilder:printcolumn:name="Model",type="string",JSONPath=".status.model"
// +kubebuilder:printcolumn:name="PowerState",type="string",JSONPath=".status.powerState"
// +kubebuilder:printcolumn:name="Health",type="string",JSONPath=".status.health"
// +kubebuilder:printcolumn:name="SystemState",type="string",JSONPath=".status.systemState"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type BareMetalHost struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BareMetalHostSpec   `json:"spec,omitempty"`
	Status BareMetalHostStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BareMetalHostList contains a list of BareMetalHost
type BareMetalHostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BareMetalHost `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BareMetalHost{}, &BareMetalHostList{})
}
