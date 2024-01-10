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

// BareMetalHostClaimSpec defines the desired state of BareMetalHostClaim
type BareMetalHostClaimSpec struct {
	Power            PowerState               `json:"power"`
	BareMetalHostRef v1.LocalObjectReference  `json:"bareMetalHostRef"`
	IgnitionRef      *v1.LocalObjectReference `json:"ignitionRef,omitempty"`
	Image            string                   `json:"image"`
}

// BareMetalHostClaimStatus defines the observed state of BareMetalHostClaim
type BareMetalHostClaimStatus struct {
	Phase Phase `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced,shortName=hostclaim

// BareMetalHostClaim is the Schema for the baremetalhostclaims API
// +kubebuilder:printcolumn:name="BareMetalHost",type="string",JSONPath=".spec.bareMetalHostRef.name"
// +kubebuilder:printcolumn:name="Ignition",type="string",JSONPath=".spec.ignitionRef.name"
// +kubebuilder:printcolumn:name="Image",type="string",JSONPath=".spec.image"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type BareMetalHostClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BareMetalHostClaimSpec   `json:"spec,omitempty"`
	Status BareMetalHostClaimStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BareMetalHostClaimList contains a list of BareMetalHostClaim
type BareMetalHostClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BareMetalHostClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BareMetalHostClaim{}, &BareMetalHostClaimList{})
}
