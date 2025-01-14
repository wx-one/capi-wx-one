/*
Copyright 2025.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// MachineFinalizer allows ReconcileDOCluster to clean up WX-ONE resources associated with WXOneMachine before
	// removing it from the apiserver.
	MachineFinalizer = "wxonemachine.infrastructure.cluster.x-k8s.io"
)

// WXOneMachineSpec defines the desired state of WXOneMachine.
type WXOneMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Flavor WXOneFlavor `json:"flavor,omitempty"`
	Image  WXOneImage  `json:"image,omitempty"`

	// ProviderID is the unique identifier as specified by the cloud provider.
	// +optional
	ProviderID *string `json:"providerID,omitempty"`
}

// WXOneMachineStatus defines the observed state of WXOneMachine.
type WXOneMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:default=false
	Ready bool `json:"ready"`
	// addresses contains the associated addresses for the machine.
	// +optional
	Addresses []clusterv1.MachineAddress `json:"addresses,omitempty"`
	Flavor    WXOneResourceReference     `json:"flavor,omitempty"`
	Image     WXOneResourceReference     `json:"image,omitempty"`
	Instance  WXOneResourceReference     `json:"instance,omitempty"`
	// +optional
	FloatingIPAttachement WXOneResourceReference `json:"floatingIPAttachement"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:conversion:hub
// +kubebuilder:subresource:status

// WXOneMachine is the Schema for the wxonemachines API.
type WXOneMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WXOneMachineSpec   `json:"spec,omitempty"`
	Status WXOneMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WXOneMachineList contains a list of WXOneMachine.
type WXOneMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WXOneMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WXOneMachine{}, &WXOneMachineList{})
}
