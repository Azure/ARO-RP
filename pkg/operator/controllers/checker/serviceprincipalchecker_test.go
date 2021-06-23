package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	azuretypes "github.com/openshift/installer/pkg/types/azure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
)

// TODO - once aad.GetToken is mockable add tests for other cases
func TestServicePrincipalValid(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name       string
		aroCluster *arov1alpha1.Cluster
		wantErr    string
	}{
		{
			name:    "fail: aro cluster resource doesn't exist",
			wantErr: `clusters.aro.openshift.io "cluster" not found`,
		},
		{
			name: "fail: invalid cluster resource",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					AZEnvironment: azuretypes.PublicCloud.Name(),
					ResourceID:    "invalid_resource",
				},
			},
			wantErr: `parsing failed for invalid_resource. Invalid resource Id format`,
		},
		{
			name: "fail: invalid az environment",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					AZEnvironment: "NEVERLAND",
					ResourceID:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/mycluster",
				},
			},
			wantErr: `cloud environment "NEVERLAND" is unsupported by ARO`,
		},
		{
			name: "fail: azure-credential secret doesn't exist",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					AZEnvironment: azuretypes.PublicCloud.Name(),
					ResourceID:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/mycluster",
				},
			},
			wantErr: `secrets "azure-credentials" not found`,
		},
	} {
		arocli := arofake.NewSimpleClientset()
		kubernetescli := fake.NewSimpleClientset()

		if tt.aroCluster != nil {
			arocli = arofake.NewSimpleClientset(tt.aroCluster)
		}

		sp := &ServicePrincipalChecker{
			arocli:        arocli,
			kubernetescli: kubernetescli,
		}

		t.Run(tt.name, func(t *testing.T) {
			err := sp.Check(ctx)

			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(fmt.Errorf("\n%s\n !=\n%s", err.Error(), tt.wantErr))
			}
		})
	}
}
