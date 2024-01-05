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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DHCPSpec defines the desired state of DHCP
type DHCPSpec struct {
	BareMetalHostRef v1.LocalObjectReference `json:"bareMetalHostRef"`
}

type DHCPState string

const (
	DHCPStateReady   DHCPState = "Ready"
	DHCPStateApplied DHCPState = "Applied"
	DHCPStateCreated DHCPState = "Created"
	DHCPStateFailed  DHCPState = "Failed"
)

// DHCPStatus defines the observed state of DHCP
type DHCPStatus struct {
	State DHCPState `json:"state,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DHCP is the Schema for the dhcps API
// +kubebuilder:printcolumn:name="BareMetalHost",type="string",JSONPath=".spec.bareMetalHostRef.name"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type DHCP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DHCPSpec   `json:"spec,omitempty"`
	Status DHCPStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DHCPList contains a list of DHCP
type DHCPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DHCP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DHCP{}, &DHCPList{})
}
