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

var (
	// PXEFinalizer is the finalizer for a PXE object.
	PXEFinalizer = "boot.afritzler.github.io/pxe"
)

// PXESpec defines the desired state of PXE
type PXESpec struct {
	SystemUUID            string                   `json:"systemUuid"`
	BareMetalHostClaimRef v1.LocalObjectReference  `json:"bareMetalHostRef"`
	IgnitionRef           *v1.LocalObjectReference `json:"ignitionRef,omitempty"`
	Image                 string                   `json:"image,omitempty"`
}

type PXEState string

const (
	PXEStateReady   PXEState = "Ready"
	PXEStateApplied PXEState = "Applied"
	PXEStateCreated PXEState = "Created"
	PXEStateFailed  PXEState = "Failed"
)

// PXEStatus defines the observed state of PXE
type PXEStatus struct {
	State PXEState `json:"state,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PXE is the Schema for the pxes API
// +kubebuilder:printcolumn:name="BareMetalHost",type="string",JSONPath=".spec.bareMetalHostRef.name"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type PXE struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PXESpec   `json:"spec,omitempty"`
	Status PXEStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PXEList contains a list of PXE
type PXEList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PXE `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PXE{}, &PXEList{})
}
