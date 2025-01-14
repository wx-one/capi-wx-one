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

const (
	// ClusterFinalizer allows ReconcileDOCluster to clean up WX-ONE resources associated with WXONECluster before
	// removing it from the apiserver.
	ClusterFinalizer = "wxonecluster.infrastructure.cluster.x-k8s.io"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WXOneClusterSpec defines the desired state of WXOneCluster.
type WXOneClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Project WXOneProject `json:"project,omitempty"`

	// +kubebuilder:validation:Enum=wx_dus_1
	AvailabilityZone string       `json:"availabilityZone,omitempty"`
	Network          WXOneNetwork `json:"network,omitempty"`
	SSHKey           WXOneSSHKey  `json:"sshKey,omitempty"`

	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`
}

// WXOneClusterStatus defines the observed state of WXOneCluster.
type WXOneClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:default=false
	Ready      bool                    `json:"ready"`
	Project    WXOneResourceReference  `json:"project,omitempty"`
	Network    WXOneNetworkResource    `json:"network,omitempty"`
	SSHKey     WXOneResourceReference  `json:"sshKey,omitempty"`
	FloatingIP WXOneFloatingIPResource `json:"floatingIP,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:conversion:hub
// +kubebuilder:subresource:status

// WXOneCluster is the Schema for the wxoneclusters API.
type WXOneCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WXOneClusterSpec   `json:"spec,omitempty"`
	Status WXOneClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WXOneClusterList contains a list of WXOneCluster.
type WXOneClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WXOneCluster `json:"items"`
}

type WXOneResourceStatus string

var (
	// WXOneResourceStatusNew is the string representing a DigitalOcean resource just created and in a provisioning state.
	WXOneResourceStatusNew = WXOneResourceStatus("new")
	// WXOneResourceStatusRunning is the string representing a DigitalOcean resource already provisioned and in a active state.
	WXOneResourceStatusRunning = WXOneResourceStatus("active")
	// WXOneResourceStatusErrored is the string representing a DigitalOcean resource in a errored state.
	WXOneResourceStatusErrored = WXOneResourceStatus("errored")
	// WXOneResourceStatusOff is the string representing a DigitalOcean resource in off state.
	WXOneResourceStatusOff = WXOneResourceStatus("off")
	// WXOneResourceStatusArchive is the string representing a DigitalOcean resource in archive state.
	WXOneResourceStatusArchive = WXOneResourceStatus("archive")
)

type WXOneResourceReference struct {
	// ID of WX-ONE resource
	// +optional
	ResourceID string `json:"resourceId,omitempty"`
	// Status of WX-ONE resource
	// +optional
	ResourceStatus WXOneResourceStatus `json:"resourceStatus,omitempty"`
}

type WXOneProject struct {
	Name string `json:"name"`
}

type WXOneNetwork struct {
	Name    string        `json:"name"`
	Subnets []WXOneSubnet `json:"subnets"`
}

type WXOneSubnet struct {
	Name string `json:"name"`

	// +kubebuilder:validation:Enum=IPv4
	IPVersion string `json:"ipversion"`
	// +kubebuilder:validation:Pattern=`^(([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2})$`
	CIDR string `json:"cidr"`
}

type WXOneNetworkResource struct {
	// +optional
	ResourceID string `json:"resourceId,omitempty"`
	// +optional
	SubnetReference WXOneResourceReference `json:"subnetReference,omitempty"`
}

type WXOneFloatingIPResource struct {
	// +optional
	ResourceID string `json:"resourceId,omitempty"`
	// +optional
	IP string `json:"ip,omitempty"`
}

type WXOneSSHKey struct {
	Name        string `json:"name"`
	PublicKey   string `json:"publicKey"`
	ProjectWide bool   `json:"projectWide"`
}

func init() {
	SchemeBuilder.Register(&WXOneCluster{}, &WXOneClusterList{})
}
