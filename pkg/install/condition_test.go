package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	consoleapi "github.com/openshift/console-operator/pkg/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

const errMustBeNilMsg = "err must be nil; condition is retried until timeout"

func TestBootstrapConfigMapReady(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name               string
		configMapName      string
		configMapNamespace string
		configMapStatus    string
		want               bool
	}{
		{
			name: "Can't get config maps for kube-system namespace",
		},
		{
			name:               "Can't get bootstrap config map",
			configMapNamespace: "kube-system",
		},
		{
			name:               "Status not complete",
			configMapName:      "bootstrap",
			configMapNamespace: "kube-system",
		},
		{
			name:               "Bootstrap config map is ready",
			configMapName:      "bootstrap",
			configMapNamespace: "kube-system",
			configMapStatus:    "complete",
			want:               true,
		},
	} {
		i := &Installer{
			kubernetescli: k8sfake.NewSimpleClientset(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.configMapName,
					Namespace: tt.configMapNamespace,
				},
				Data: map[string]string{
					"status": tt.configMapStatus,
				},
			}),
		}
		ready, err := i.bootstrapConfigMapReady(ctx)
		if err != nil {
			t.Error(errMustBeNilMsg)
		}
		if ready != tt.want {
			t.Error(ready)
		}
	}
}

func TestOperatorConsoleExists(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name        string
		consoleName string
		want        bool
	}{
		{
			name: "Can't get operator console",
		},
		{
			name:        "Operator console exists",
			consoleName: consoleapi.ConfigResourceName,
			want:        true,
		},
	} {
		i := &Installer{
			operatorcli: operatorfake.NewSimpleClientset(&operatorv1.Console{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.consoleName,
				},
			}),
		}
		ready, err := i.operatorConsoleExists(ctx)
		if err != nil {
			t.Error(errMustBeNilMsg)
		}
		if ready != tt.want {
			t.Error(ready)
		}
	}
}

func TestIsOperatorAvailable(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		availableCondition   configv1.ConditionStatus
		progressingCondition configv1.ConditionStatus
		want                 bool
	}{
		{
			name:                 "Available && Progressing; not available",
			availableCondition:   configv1.ConditionTrue,
			progressingCondition: configv1.ConditionTrue,
		},
		{
			name:                 "Available && !Progressing; available",
			availableCondition:   configv1.ConditionTrue,
			progressingCondition: configv1.ConditionFalse,
			want:                 true,
		},
		{
			name:                 "!Available && Progressing; not available",
			availableCondition:   configv1.ConditionFalse,
			progressingCondition: configv1.ConditionTrue,
		},
		{
			name:                 "!Available && !Progressing; not available",
			availableCondition:   configv1.ConditionFalse,
			progressingCondition: configv1.ConditionFalse,
		},
	} {

		operator := &configv1.ClusterOperator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "name",
			},
			Status: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: tt.availableCondition,
					},
					{
						Type:   configv1.OperatorProgressing,
						Status: tt.progressingCondition,
					},
				},
			},
		}
		available := isOperatorAvailable(operator)
		if available != tt.want {
			t.Error(available)
		}
	}
}

func TestClusterVersionReady(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name               string
		version            string
		availableCondition configv1.ConditionStatus
		want               bool
	}{
		{
			name: "Can't get cluster version",
		},
		{
			name:               "Cluster version not ready yet",
			version:            "version",
			availableCondition: configv1.ConditionFalse,
		},
		{
			name:               "Cluster version ready",
			version:            "version",
			availableCondition: configv1.ConditionTrue,
			want:               true,
		},
	} {
		i := &Installer{
			configcli: configfake.NewSimpleClientset(&configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.version,
				},
				Status: configv1.ClusterVersionStatus{
					Conditions: []configv1.ClusterOperatorStatusCondition{
						{
							Type:   configv1.OperatorAvailable,
							Status: tt.availableCondition,
						},
					},
				},
			}),
		}
		ready, err := i.clusterVersionReady(ctx)
		if err != nil {
			t.Error(errMustBeNilMsg)
		}
		if ready != tt.want {
			t.Error(ready)
		}
	}
}
