package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	mock_platformworkloadidentity "github.com/Azure/ARO-RP/pkg/util/mocks/platformworkloadidentity"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestGeneratePlatformWorkloadIdentitySecrets(t *testing.T) {
	tenantId := "00000000-0000-0000-0000-000000000000"
	subscriptionId := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	location := "eastus"

	successCases := []struct {
		name       string
		identities []api.PlatformWorkloadIdentity
		roles      []api.PlatformWorkloadIdentityRole
		want       []corev1.Secret
	}{
		{
			name:       "no identities, no secrets",
			identities: []api.PlatformWorkloadIdentity{},
			roles:      []api.PlatformWorkloadIdentityRole{},
			want:       []corev1.Secret{},
		},
		{
			name: "converts cluster PWIs if a role definition is present",
			identities: []api.PlatformWorkloadIdentity{
				{
					OperatorName: "foo",
					ClientID:     "00f00f00-0f00-0f00-0f00-f00f00f00f00",
				},
				{
					OperatorName: "bar",
					ClientID:     "00ba4ba4-0ba4-0ba4-0ba4-ba4ba4ba4ba4",
				},
			},
			roles: []api.PlatformWorkloadIdentityRole{
				{
					OperatorName: "foo",
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-foo",
						Name:      "azure-cloud-credentials",
					},
				},
				{
					OperatorName: "bar",
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-bar",
						Name:      "azure-cloud-credentials",
					},
				},
			},
			want: []corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "openshift-foo",
						Name:      "azure-cloud-credentials",
					},
					Type: corev1.SecretTypeOpaque,
					StringData: map[string]string{
						"azure_client_id":            "00f00f00-0f00-0f00-0f00-f00f00f00f00",
						"azure_subscription_id":      subscriptionId,
						"azure_tenant_id":            tenantId,
						"azure_region":               location,
						"azure_federated_token_file": "/var/run/secrets/openshift/serviceaccount/token",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "openshift-bar",
						Name:      "azure-cloud-credentials",
					},
					Type: corev1.SecretTypeOpaque,
					StringData: map[string]string{
						"azure_client_id":            "00ba4ba4-0ba4-0ba4-0ba4-ba4ba4ba4ba4",
						"azure_subscription_id":      subscriptionId,
						"azure_tenant_id":            tenantId,
						"azure_region":               location,
						"azure_federated_token_file": "/var/run/secrets/openshift/serviceaccount/token",
					},
				},
			},
		},
		{
			name: "ignores identities with no role present",
			identities: []api.PlatformWorkloadIdentity{
				{
					OperatorName: "foo",
					ClientID:     "00f00f00-0f00-0f00-0f00-f00f00f00f00",
				},
				{
					OperatorName: "bar",
					ClientID:     "00ba4ba4-0ba4-0ba4-0ba4-ba4ba4ba4ba4",
				},
			},
			roles: []api.PlatformWorkloadIdentityRole{},
			want:  []corev1.Secret{},
		},
	}
	for _, tt := range successCases {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			pwiRolesByVersion := mock_platformworkloadidentity.NewMockPlatformWorkloadIdentityRolesByVersion(controller)
			platformWorkloadIdentityRolesByRoleName := map[string]api.PlatformWorkloadIdentityRole{}
			for _, role := range tt.roles {
				platformWorkloadIdentityRolesByRoleName[role.OperatorName] = role
			}
			pwiRolesByVersion.EXPECT().GetPlatformWorkloadIdentityRolesByRoleName().AnyTimes().Return(platformWorkloadIdentityRolesByRoleName)

			m := manager{
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
								PlatformWorkloadIdentities: tt.identities,
							},
						},
					},
				},
				subscriptionDoc: &api.SubscriptionDocument{
					ID: subscriptionId,
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							TenantID: tenantId,
						},
					},
				},

				platformWorkloadIdentityRolesByVersion: pwiRolesByVersion,
			}
			got, err := m.generatePlatformWorkloadIdentitySecrets()

			utilerror.AssertErrorMessage(t, err, "")
			less := func(a, b corev1.Secret) bool {
				return strings.Compare(a.ObjectMeta.Namespace+"/"+a.ObjectMeta.Name, b.ObjectMeta.Namespace+"/"+b.ObjectMeta.Name) < 0
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.SortSlices(less)); diff != "" {
				t.Errorf("got %v, want %v, diff %s", got, tt.want, diff)
			}
		})
	}

	errorCases := []struct {
		name    string
		doc     *api.OpenShiftClusterDocument
		wantErr string
	}{
		{
			name: "returns error if cluster is not using workload identity",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Location: location,
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: nil,
						ServicePrincipalProfile: &api.ServicePrincipalProfile{
							ClientID:     "id",
							ClientSecret: "secret",
						},
					},
				},
			},
			wantErr: "generatePlatformWorkloadIdentitySecrets called for a CSP cluster",
		},
	}
	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			m := manager{
				doc: tt.doc,
			}

			_, err := m.generatePlatformWorkloadIdentitySecrets()
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
