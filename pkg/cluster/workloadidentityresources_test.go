package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_platformworkloadidentity "github.com/Azure/ARO-RP/pkg/util/mocks/platformworkloadidentity"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestGenerateWorkloadIdentityResources(t *testing.T) {
	tenantId := "00000000-0000-0000-0000-000000000000"
	subscriptionId := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	location := "eastus"
	oidcIssuer := "https://testoidcissuer.local/cluster"

	for _, tt := range []struct {
		name                 string
		usesWorkloadIdentity bool
		identities           map[string]api.PlatformWorkloadIdentity
		roles                []api.PlatformWorkloadIdentityRole
		want                 map[string]kruntime.Object
		wantErr              string
	}{
		{
			name:    "returns error if cluster is not using workload identity",
			wantErr: "generateWorkloadIdentityResources called for a Cluster Service Principal cluster",
		},
		{
			name:                 "generates all expected resources",
			usesWorkloadIdentity: true,
			identities: map[string]api.PlatformWorkloadIdentity{
				"foo": {
					ClientID: "00f00f00-0f00-0f00-0f00-f00f00f00f00",
				},
				"bar": {
					ClientID: "00ba4ba4-0ba4-0ba4-0ba4-ba4ba4ba4ba4",
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
			want: map[string]kruntime.Object{
				"openshift-foo-azure-cloud-credentials-credentials.yaml": &corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
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
						"azure_federated_token_file": azureFederatedTokenFileLocation,
					},
				},
				"openshift-bar-azure-cloud-credentials-credentials.yaml": &corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
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
						"azure_federated_token_file": azureFederatedTokenFileLocation,
					},
				},
				ccoSecretFilename: &corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ccoSecretNamespace,
						Name:      ccoSecretName,
					},
					Type: corev1.SecretTypeOpaque,
					StringData: map[string]string{
						"azure_tenant_id": tenantId,
					},
				},
				authenticationConfigFilename: &configv1.Authentication{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "config.openshift.io/v1",
						Kind:       "Authentication",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: authenticationConfigName,
					},
					Spec: configv1.AuthenticationSpec{
						ServiceAccountIssuer: oidcIssuer,
					},
				},
			},
		},
	} {
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
							ClusterProfile: api.ClusterProfile{
								OIDCIssuer: (*api.OIDCIssuer)(&oidcIssuer),
							},
							PlatformWorkloadIdentityProfile: nil,
							ServicePrincipalProfile:         &api.ServicePrincipalProfile{},
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
			if tt.usesWorkloadIdentity {
				m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile = &api.PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: tt.identities,
				}
				m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile = nil
			}

			got, err := m.generateWorkloadIdentityResources()
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func TestGeneratePlatformWorkloadIdentitySecrets(t *testing.T) {
	tenantId := "00000000-0000-0000-0000-000000000000"
	subscriptionId := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	location := "eastus"

	for _, tt := range []struct {
		name       string
		identities map[string]api.PlatformWorkloadIdentity
		roles      []api.PlatformWorkloadIdentityRole
		want       []*corev1.Secret
	}{
		{
			name:       "no identities, no secrets",
			identities: map[string]api.PlatformWorkloadIdentity{},
			roles:      []api.PlatformWorkloadIdentityRole{},
			want:       []*corev1.Secret{},
		},
		{
			name: "converts cluster PWIs if a role definition is present",
			identities: map[string]api.PlatformWorkloadIdentity{
				"foo": {
					ClientID: "00f00f00-0f00-0f00-0f00-f00f00f00f00",
				},
				"bar": {
					ClientID: "00ba4ba4-0ba4-0ba4-0ba4-ba4ba4ba4ba4",
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
			want: []*corev1.Secret{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
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
						"azure_federated_token_file": azureFederatedTokenFileLocation,
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
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
						"azure_federated_token_file": azureFederatedTokenFileLocation,
					},
				},
			},
		},
		{
			name: "ignores identities with no role present",
			identities: map[string]api.PlatformWorkloadIdentity{
				"foo": {
					ClientID: "00f00f00-0f00-0f00-0f00-f00f00f00f00",
				},
				"bar": {
					ClientID: "00ba4ba4-0ba4-0ba4-0ba4-ba4ba4ba4ba4",
				},
			},
			roles: []api.PlatformWorkloadIdentityRole{},
			want:  []*corev1.Secret{},
		},
		{
			name: "skips ARO operator identity",
			identities: map[string]api.PlatformWorkloadIdentity{
				"foo": {
					ClientID: "00f00f00-0f00-0f00-0f00-f00f00f00f00",
				},
				"ServiceOperator": {
					ClientID: "00ba4ba4-0ba4-0ba4-0ba4-ba4ba4ba4ba4",
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
					OperatorName: "ServiceOperator",
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-bar",
						Name:      "azure-cloud-credentials",
					},
				},
			},
			want: []*corev1.Secret{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
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
						"azure_federated_token_file": azureFederatedTokenFileLocation,
					},
				},
			},
		},
	} {
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
			assert.ElementsMatch(t, got, tt.want)
		})
	}
}

func TestGenerateCloudCredentialOperatorSecret(t *testing.T) {
	tenantId := "00000000-0000-0000-0000-000000000000"

	for _, tt := range []struct {
		name                 string
		usesWorkloadIdentity bool
		want                 *corev1.Secret
		wantErr              string
	}{
		{
			name:                 "generates static CCO secret",
			usesWorkloadIdentity: true,
			want: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ccoSecretNamespace,
					Name:      ccoSecretName,
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"azure_tenant_id": tenantId,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := manager{
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							PlatformWorkloadIdentityProfile: nil,
							ServicePrincipalProfile:         &api.ServicePrincipalProfile{},
						},
					},
				},
				subscriptionDoc: &api.SubscriptionDocument{
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							TenantID: tenantId,
						},
					},
				},
			}

			if tt.usesWorkloadIdentity {
				m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile = &api.PlatformWorkloadIdentityProfile{}
				m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile = nil
			}

			got, err := m.generateCloudCredentialOperatorSecret()
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateAuthenticationConfig(t *testing.T) {
	oidcIssuer := "https://testoidcissuer.local/cluster"

	for _, tt := range []struct {
		name                 string
		usesWorkloadIdentity bool
		oidcIssuer           *api.OIDCIssuer
		want                 *configv1.Authentication
		wantErr              string
	}{
		{
			name:                 "returns error if oidcIssuer is nil in clusterdoc",
			usesWorkloadIdentity: true,
			wantErr:              "oidcIssuer not present in clusterdoc",
		},
		{
			name:                 "generates static Authentication config",
			usesWorkloadIdentity: true,
			oidcIssuer:           (*api.OIDCIssuer)(&oidcIssuer),
			want: &configv1.Authentication{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "config.openshift.io/v1",
					Kind:       "Authentication",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: authenticationConfigName,
				},
				Spec: configv1.AuthenticationSpec{
					ServiceAccountIssuer: oidcIssuer,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := manager{
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								OIDCIssuer: tt.oidcIssuer,
							},
							PlatformWorkloadIdentityProfile: nil,
							ServicePrincipalProfile:         &api.ServicePrincipalProfile{},
						},
					},
				},
			}

			if tt.usesWorkloadIdentity {
				m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile = &api.PlatformWorkloadIdentityProfile{}
				m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile = nil
			}

			got, err := m.generateAuthenticationConfig()
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPlatformWorkloadIdentityFederatedCredName(t *testing.T) {
	docID := "00000000-0000-0000-0000-000000000000"
	subID := "00000000-0000-0000-0000-000000000000"
	clusterRGName := "aro-cluster-rg"
	clusterName := "aro-cluster"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/", subID, clusterName)
	clusterResourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/%s", subID, clusterRGName, clusterName)

	for _, tt := range []struct {
		name           string
		doc            *api.OpenShiftClusterDocument
		identity       api.PlatformWorkloadIdentity
		serviceAccount string
		wantErr        string
		want           string
	}{
		{
			name: "fail - getPlatformWorkloadIdentityFederatedCredName called for a CSP cluster",
			doc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{},
				},
			},
			identity: api.PlatformWorkloadIdentity{},
			wantErr:  "getPlatformWorkloadIdentityFederatedCredName called for a CSP cluster",
			want:     "",
		},
		{
			name: "fail - invalid service account name",
			doc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
						},
					},
				},
			},
			serviceAccount: "   ",
			identity:       api.PlatformWorkloadIdentity{},
			wantErr:        "service account name is required",
			want:           "",
		},
		{
			name: "success - return federated identity name for platform workload identity",
			doc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
						},
					},
				},
			},
			serviceAccount: "openshift-cloud-controller-manager:cloud-controller-manager",
			identity:       api.PlatformWorkloadIdentity{ResourceID: fmt.Sprintf("%s/%s", resourceID, "ccm")},
			wantErr:        "",
			want:           fmt.Sprintf("%s-%s-%s", subID, clusterRGName, clusterName),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := manager{
				doc: tt.doc,
			}
			got, err := m.getPlatformWorkloadIdentityFederatedCredName(tt.serviceAccount, tt.identity)

			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			assert.Contains(t, got, tt.want)
		})
	}
}
