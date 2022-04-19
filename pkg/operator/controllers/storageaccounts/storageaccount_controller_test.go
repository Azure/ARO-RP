package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
)

func TestControllerReconcile(t *testing.T) {
	resourceId := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openshiftClusters/myCluster"
	for _, tt := range []struct {
		name    string
		cluster *arov1alpha1.Cluster
		wantErr string
	}{
		{
			name: "cluster.aro.openshift.io cluster not found",
			cluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "non-existing",
				},
			},
			wantErr: `clusters.aro.openshift.io "cluster" not found`,
		},
		{
			name: "storage account controller not enabled, do nothing",
			cluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: "false",
					},
				},
			},
		},
		{
			name: "storage account controller managed not set, do nothing",
			cluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: "true",
						controllerManaged: "",
					},
				},
			},
		},
		{
			name: "azure environment invalid",
			cluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: "true",
						controllerManaged: "true",
					},
					AZEnvironment: "ʇɐɔoɔɐʇ",
				},
			},
			wantErr: `cloud environment "ʇɐɔoɔɐʇ" is unsupported by ARO`,
		},
		{
			name: "resource id invalid ",
			cluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: "true",
						controllerManaged: "true",
					},
					AZEnvironment: "AzurePublicCloud",
				},
			},
			wantErr: "parsing failed for . Invalid resource Id format",
		},
		{
			name: "azure-credentials secret not found ",
			cluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: "true",
						controllerManaged: "true",
					},
					AZEnvironment: "AzurePublicCloud",
					ResourceID:    resourceId,
				},
			},
			wantErr: `secrets "azure-credentials" not found`,
		},
		// TODO - finish unit tests once we can mock clusterauthorizer.NewAzRefreshableAuthorizer
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			r := Reconciler{
				arocli:        arofake.NewSimpleClientset(tt.cluster),
				kubernetescli: fake.NewSimpleClientset(),
			}

			_, err := r.Reconcile(context.Background(), reconcile.Request{})
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("got error '%v', wanted error '%v'", err, tt.wantErr)
			}

			if err == nil && tt.wantErr != "" {
				t.Errorf("did not get an error, but wanted error '%v'", tt.wantErr)
			}
		})
	}
}
