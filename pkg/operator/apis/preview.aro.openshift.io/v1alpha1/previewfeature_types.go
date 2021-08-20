package v1alpha1

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SingletonPreviewFeatureName = "cluster"
)

// NSGFlowLogs defines NSG flow logs configuration
type NSGFlowLogs struct {
	// Enabled defines the behaviour of the reconciler:
	// when true the controller will try to reach the desired configuration,
	// when false it will try to disable the flow logs.
	Enabled bool `json:"enabled"`

	// Version defines version of NSG flow log.
	// +kubebuilder:default:=2
	// +kubebuilder:validation:Enum=1;2
	Version int `json:"version,omitempty"`

	// StorageAccountResourceID should be defined if BYO-Storage account is used.
	// Must be in the same region of flow log.
	StorageAccountResourceID string `json:"storageAccountResourceId,omitempty"`

	// Retention period for data.
	// +kubebuilder:default:="2160h"
	Retention metav1.Duration `json:"retention,omitempty"`

	// Required for TrafficAnalytics.
	// Must be in the same region of flow log.
	TrafficAnalyticsLogAnalyticsWorkspaceID string `json:"trafficAnalyticsLogAnalyticsWorkspaceId,omitempty"`

	// Interval at which to conduct flow analytics. Values: 60m, 10m. Default: 60m.
	// +kubebuilder:default:="60m"
	// +kubebuilder:validation:Enum="60m";"10m"
	TrafficAnalyticsInterval metav1.Duration `json:"trafficAnalyticsInterval,omitempty"`
}

// PreviewFeatureSpec defines the preview feature for ARO
type PreviewFeatureSpec struct {
	// NSGFlowLogs contains configuration for NSG flow logs.
	// Omit the configuration if you don't want the controller
	// to reconcile NSG flow logs.
	NSGFlowLogs *NSGFlowLogs `json:"nsgFlowLogs,omitempty"`
}

// PreviewFeatureStatus defines the observed state of PreviewFeature
type PreviewFeatureStatus struct {
	OperatorVersion   string            `json:"operatorVersion,omitempty"`
	Conditions        status.Conditions `json:"conditions,omitempty"`
	RedHatKeysPresent []string          `json:"redHatKeysPresent,omitempty"`
}

// PreviewFeature is the Schema for the preview feature API
// +kubebuilder:object:root=true
// +genclient
// +genclient:nonNamespaced
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PreviewFeature struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PreviewFeatureSpec   `json:"spec,omitempty"`
	Status PreviewFeatureStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PreviewFeatureList contains a list of PreviewFeatures
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PreviewFeatureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PreviewFeature `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PreviewFeature{}, &PreviewFeatureList{})
}
