package v1alpha1

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SingletonClusterName                             = "cluster"
	InternetReachableFromMaster status.ConditionType = "InternetReachableFromMaster"
	InternetReachableFromWorker status.ConditionType = "InternetReachableFromWorker"
	MachineValid                status.ConditionType = "MachineValid"
)

func AllConditionTypes() []status.ConditionType {
	return []status.ConditionType{InternetReachableFromMaster, InternetReachableFromWorker, MachineValid}
}

type GenevaLoggingSpec struct {
	// +kubebuilder:validation:Pattern:=`[0-9]+.[0-9]+`
	ConfigVersion string `json:"configVersion,omitempty"`
	// +kubebuilder:validation:Enum=DiagnosticsProd;Test
	MonitoringGCSEnvironment string `json:"monitoringGCSEnvironment,omitempty"`
}

type InternetCheckerSpec struct {
	URLs []string `json:"urls,omitempty"`
}

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// ResourceID is the Azure resourceId of the cluster
	ResourceID      string              `json:"resourceId,omitempty"`
	ACRDomain       string              `json:"acrDomain,omitempty"`
	Location        string              `json:"location,omitempty"`
	GenevaLogging   GenevaLoggingSpec   `json:"genevaLogging,omitempty"`
	InternetChecker InternetCheckerSpec `json:"internetChecker,omitempty"`

	Features FeaturesSpec `json:"features,omitempty"`
}

// FeaturesSpec defines ARO operator feature gates
type FeaturesSpec struct {
	PersistentPrometheus bool `json:"persistentPrometheus,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	OperatorVersion string            `json:"operatorVersion,omitempty"`
	Conditions      status.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +genclient
// +genclient:nonNamespaced
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
