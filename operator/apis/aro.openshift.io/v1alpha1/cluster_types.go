package v1alpha1

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const SingletonClusterName = "cluster"

var (
	InternetReachable status.ConditionType = "InternetReachable"
)

type GenevaLoggingSpec struct {
	Namespace                string `json:"namespace,omitempty"`
	ConfigVersion            string `json:"configVersion,omitempty"`
	MonitoringTenant         string `json:"monitoringTenant,omitempty"`
	MonitoringGCSRegion      string `json:"monitoringGCSRegion,omitempty"`
	MonitoringGCSEnvironment string `json:"monitoringGCSEnvironment,omitempty"`
}

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// ResourceID is the Azure resourceId of the cluster
	ResourceID    string            `json:"resourceId,omitempty"`
	ACRName       string            `json:"acrName,omitempty"`
	GenevaLogging GenevaLoggingSpec `json:"genevaLogging,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	Conditions     status.Conditions        `json:"conditions,omitempty"`
	RelatedObjects []corev1.ObjectReference `json:"relatedObjects,omitempty"`
}

// +kubebuilder:object:root=true

// Cluster is the Schema for the clusters API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
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
