package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	mock_authorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_platformworkloadidentity "github.com/Azure/ARO-RP/pkg/util/mocks/platformworkloadidentity"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
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
		roles                map[string]api.PlatformWorkloadIdentityRole
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
				pkgoperator.OperatorIdentityName: {
					ClientID: "00d00d00-0d00-0d00-0d00-d00d00d00d00",
				},
			},
			roles: map[string]api.PlatformWorkloadIdentityRole{
				"foo": {
					OperatorName: "foo",
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-foo",
						Name:      "azure-cloud-credentials",
					},
				},
				"bar": {
					OperatorName: "bar",
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-bar",
						Name:      "azure-cloud-credentials",
					},
				},
				pkgoperator.OperatorIdentityName: {
					OperatorName: pkgoperator.OperatorIdentityName,
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-azure-operator",
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

			platformWorkloadIdentityRolesByVersion := mock_platformworkloadidentity.NewMockPlatformWorkloadIdentityRolesByVersion(controller)
			platformWorkloadIdentityRolesByVersion.EXPECT().GetPlatformWorkloadIdentityRolesByRoleName().AnyTimes().Return(tt.roles)

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

				platformWorkloadIdentityRolesByVersion: platformWorkloadIdentityRolesByVersion,
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

func TestDeployPlatformWorkloadIdentitySecrets(t *testing.T) {
	tenantId := "00000000-0000-0000-0000-000000000000"
	subscriptionId := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	location := "eastus"

	for _, tt := range []struct {
		name       string
		identities map[string]api.PlatformWorkloadIdentity
		roles      map[string]api.PlatformWorkloadIdentityRole
		want       []*corev1.Secret
	}{
		{
			name: "updates PWI secrets if a role definition is present",
			identities: map[string]api.PlatformWorkloadIdentity{
				"foo": {
					ClientID: "00f00f00-0f00-0f00-0f00-f00f00f00f00",
				},
				"bar": {
					ClientID: "00ba4ba4-0ba4-0ba4-0ba4-ba4ba4ba4ba4",
				},
				pkgoperator.OperatorIdentityName: {
					ClientID: "00d00d00-0d00-0d00-0d00-d00d00d00d00",
				},
			},
			roles: map[string]api.PlatformWorkloadIdentityRole{
				"foo": {
					OperatorName: "foo",
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-foo",
						Name:      "azure-cloud-credentials",
					},
				},
				"bar": {
					OperatorName: "bar",
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-bar",
						Name:      "azure-cloud-credentials",
					},
				},
				pkgoperator.OperatorIdentityName: {
					OperatorName: pkgoperator.OperatorIdentityName,
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-azure-operator",
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
						Namespace:       "openshift-foo",
						Name:            "azure-cloud-credentials",
						ResourceVersion: "1",
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
						Namespace:       "openshift-bar",
						Name:            "azure-cloud-credentials",
						ResourceVersion: "1",
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
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       "openshift-azure-operator",
						Name:            "azure-cloud-credentials",
						ResourceVersion: "1",
					},
					Type: corev1.SecretTypeOpaque,
					StringData: map[string]string{
						"azure_client_id":            "00d00d00-0d00-0d00-0d00-d00d00d00d00",
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
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()

			clientFake := ctrlfake.NewClientBuilder().Build()

			ch := clienthelper.NewWithClient(logrus.NewEntry(logrus.StandardLogger()), clientFake)

			platformWorkloadIdentityRolesByVersion := mock_platformworkloadidentity.NewMockPlatformWorkloadIdentityRolesByVersion(controller)
			platformWorkloadIdentityRolesByVersion.EXPECT().GetPlatformWorkloadIdentityRolesByRoleName().AnyTimes().Return(tt.roles)

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

				ch: ch,

				platformWorkloadIdentityRolesByVersion: platformWorkloadIdentityRolesByVersion,
			}
			err := m.deployPlatformWorkloadIdentitySecrets(ctx)

			utilerror.AssertErrorMessage(t, err, "")

			for _, wantSecret := range tt.want {
				gotSecret := &corev1.Secret{}
				key := types.NamespacedName{Name: wantSecret.Name, Namespace: wantSecret.Namespace}
				if err := clientFake.Get(ctx, key, gotSecret); err != nil {
					t.Error(err)
				}
				assert.Equal(t, wantSecret, gotSecret)
			}
		})
	}
}

func TestGeneratePlatformWorkloadIdentitySecrets(t *testing.T) {
	tenantId := "00000000-0000-0000-0000-000000000000"
	subscriptionId := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	location := "eastus"

	for _, tt := range []struct {
		name           string
		identities     map[string]api.PlatformWorkloadIdentity
		roles          map[string]api.PlatformWorkloadIdentityRole
		wantSecrets    []*corev1.Secret
		wantNamespaces []*corev1.Namespace
		wantErr        string
	}{
		{
			name:           "no identities, no secrets",
			identities:     map[string]api.PlatformWorkloadIdentity{},
			roles:          map[string]api.PlatformWorkloadIdentityRole{},
			wantSecrets:    []*corev1.Secret{},
			wantNamespaces: []*corev1.Namespace{},
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
			roles: map[string]api.PlatformWorkloadIdentityRole{
				"foo": {
					OperatorName: "foo",
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-foo",
						Name:      "azure-cloud-credentials",
					},
				},
				"bar": {
					OperatorName: "bar",
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-bar",
						Name:      "azure-cloud-credentials",
					},
				},
			},
			wantSecrets: []*corev1.Secret{
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
			wantNamespaces: []*corev1.Namespace{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "openshift-foo",
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "openshift-bar",
					},
				},
			},
		},
		{
			name: "error - identities with unexpected operator name present",
			identities: map[string]api.PlatformWorkloadIdentity{
				"foo": {
					ClientID: "00f00f00-0f00-0f00-0f00-f00f00f00f00",
				},
			},
			roles:          map[string]api.PlatformWorkloadIdentityRole{},
			wantSecrets:    []*corev1.Secret{},
			wantNamespaces: []*corev1.Namespace{},
			wantErr:        fmt.Sprintf("400: %s: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s'. The required platform workload identities are '[]'", api.CloudErrorCodePlatformWorkloadIdentityMismatch, "4.14"),
		},
		{
			name: "skips ARO operator identity",
			identities: map[string]api.PlatformWorkloadIdentity{
				"foo": {
					ClientID: "00f00f00-0f00-0f00-0f00-f00f00f00f00",
				},
				"aro-operator": {
					ClientID: "00ba4ba4-0ba4-0ba4-0ba4-ba4ba4ba4ba4",
				},
			},
			roles: map[string]api.PlatformWorkloadIdentityRole{
				"foo": {
					OperatorName: "foo",
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-foo",
						Name:      "azure-cloud-credentials",
					},
				},
				"aro-operator": {
					OperatorName: "aro-operator",
					SecretLocation: api.SecretLocation{
						Namespace: "openshift-bar",
						Name:      "azure-cloud-credentials",
					},
				},
			},
			wantSecrets: []*corev1.Secret{
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
			wantNamespaces: []*corev1.Namespace{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "openshift-foo",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			platformWorkloadIdentityRolesByVersion := mock_platformworkloadidentity.NewMockPlatformWorkloadIdentityRolesByVersion(controller)
			platformWorkloadIdentityRolesByVersion.EXPECT().GetPlatformWorkloadIdentityRolesByRoleName().AnyTimes().Return(tt.roles)

			m := manager{
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
								PlatformWorkloadIdentities: tt.identities,
							},
							ClusterProfile: api.ClusterProfile{
								Version: "4.14.40",
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

				platformWorkloadIdentityRolesByVersion: platformWorkloadIdentityRolesByVersion,
			}
			gotSecrets, gotNamespaces, err := m.generatePlatformWorkloadIdentitySecretsAndNamespaces(true)

			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			assert.ElementsMatch(t, gotSecrets, tt.wantSecrets)
			assert.ElementsMatch(t, gotNamespaces, tt.wantNamespaces)
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
			serviceAccount: "system:serviceaccount:openshift-cloud-controller-manager:cloud-controller-manager",
			identity:       api.PlatformWorkloadIdentity{ResourceID: fmt.Sprintf("%s/%s", resourceID, "ccm")},
			wantErr:        "",
			want:           fmt.Sprintf("%s_%s", clusterName, "cloud-controller-manager"),
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

func TestEnsurePlatformWorkloadIdentityRBAC(t *testing.T) {
	ctx := context.Background()
	subscriptionId := "10000000-00000000-00000000-00000000"
	clusterRGName := "test-cluster"
	resourceGroupID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscriptionId, clusterRGName)

	objectId1 := "00000000-00000000-00000000-00000001"
	objectId2 := "00000000-00000000-00000000-00000002"

	roleName1 := "00000000-00000000-00000001-00000000"
	roleName2 := "00000000-00000000-00000002-00000000"

	roleDefinitionId1 := fmt.Sprintf("/providers/Microsoft.Authorization/roleDefinitions/%s", "00000000-00000001-00000000-00000000")
	roleDefinitionId2 := fmt.Sprintf("/providers/Microsoft.Authorization/roleDefinitions/%s", "00000000-00000002-00000000-00000000")

	for _, tt := range []struct {
		name                    string
		oc                      *api.OpenShiftCluster
		roles                   map[string]api.PlatformWorkloadIdentityRole
		existingRoleAssignments []mgmtauthorization.RoleAssignment
		wantDeleted             []mgmtauthorization.RoleAssignment
		wantAdded               []*arm.Resource
		wantErr                 string
	}{
		{
			name: "noop",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: resourceGroupID,
					},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{},
					},
				},
			},
		},
		{
			name: "all identities match existing role assignments - noop",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: resourceGroupID,
					},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"1": {
								ObjectID: objectId1,
							},
							"2": {
								ObjectID: objectId2,
							},
						},
					},
				},
			},
			roles: map[string]api.PlatformWorkloadIdentityRole{
				"1": {
					OperatorName:     "1",
					RoleDefinitionID: roleDefinitionId1,
				},
				"2": {
					OperatorName:     "2",
					RoleDefinitionID: roleDefinitionId2,
				},
			},
			existingRoleAssignments: []mgmtauthorization.RoleAssignment{
				{
					Name: &roleName1,
					RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
						Scope:            &resourceGroupID,
						RoleDefinitionID: pointerutils.ToPtr(fmt.Sprintf("/subscriptions/%s%s", subscriptionId, roleDefinitionId1)),
						PrincipalID:      &objectId1,
					},
				},
				{
					Name: &roleName2,
					RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
						Scope:            &resourceGroupID,
						RoleDefinitionID: pointerutils.ToPtr(fmt.Sprintf("/subscriptions/%s%s", subscriptionId, roleDefinitionId2)),
						PrincipalID:      &objectId2,
					},
				},
			},
		},
		{
			name: "new required identity added - creates new roleassignment",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: resourceGroupID,
					},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"1": {
								ObjectID: objectId1,
							},
							"2": {
								ObjectID: objectId2,
							},
						},
					},
				},
			},
			roles: map[string]api.PlatformWorkloadIdentityRole{
				"1": {
					OperatorName:     "1",
					RoleDefinitionID: roleDefinitionId1,
				},
				"2": {
					OperatorName:     "2",
					RoleDefinitionID: roleDefinitionId2,
				},
			},
			existingRoleAssignments: []mgmtauthorization.RoleAssignment{
				{
					Name: &roleName1,
					RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
						Scope:            &resourceGroupID,
						RoleDefinitionID: pointerutils.ToPtr(fmt.Sprintf("/subscriptions/%s%s", subscriptionId, roleDefinitionId1)),
						PrincipalID:      &objectId1,
					},
				},
			},
			wantAdded: []*arm.Resource{
				workloadIdentityResourceGroupRBAC(stringutils.LastTokenByte(roleDefinitionId2, '/'), objectId2),
			},
		},
		{
			name: "identity object id replaced - deletes old roleassignment and creates new",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: resourceGroupID,
					},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
							"1": {
								ObjectID: objectId2,
							},
						},
					},
				},
			},
			roles: map[string]api.PlatformWorkloadIdentityRole{
				"1": {
					OperatorName:     "1",
					RoleDefinitionID: roleDefinitionId1,
				},
			},
			existingRoleAssignments: []mgmtauthorization.RoleAssignment{
				{
					Name: &roleName1,
					RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
						Scope:            &resourceGroupID,
						RoleDefinitionID: pointerutils.ToPtr(fmt.Sprintf("/subscriptions/%s%s", subscriptionId, roleDefinitionId1)),
						PrincipalID:      &objectId1,
					},
				},
			},
			wantDeleted: []mgmtauthorization.RoleAssignment{
				{
					Name: &roleName1,
					RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
						Scope:            &resourceGroupID,
						RoleDefinitionID: pointerutils.ToPtr(fmt.Sprintf("/subscriptions/%s%s", subscriptionId, roleDefinitionId1)),
						PrincipalID:      &objectId1,
					},
				},
			},
			wantAdded: []*arm.Resource{
				workloadIdentityResourceGroupRBAC(stringutils.LastTokenByte(roleDefinitionId1, '/'), objectId2),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			roleAssignments := mock_authorization.NewMockRoleAssignmentsClient(controller)
			deployments := mock_features.NewMockDeploymentsClient(controller)
			platformWorkloadIdentityRolesByVersion := mock_platformworkloadidentity.NewMockPlatformWorkloadIdentityRolesByVersion(controller)

			roleAssignments.EXPECT().ListForResourceGroup(gomock.Eq(ctx), gomock.Eq(clusterRGName), gomock.Eq("atScope()")).Return(tt.existingRoleAssignments, nil)
			if len(tt.wantAdded) > 0 {
				deployments.EXPECT().CreateOrUpdateAndWait(gomock.Any(), gomock.Eq(clusterRGName), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, resourceGroupName string, deploymentName string, parameters mgmtfeatures.Deployment) error {
						template, ok := parameters.Properties.Template.(*arm.Template)
						if !ok {
							t.Errorf("failed to retrieve deployed ARM template, was unexpected type %T", parameters.Properties.Template)
						}
						gotResources := template.Resources

						if diff := cmp.Diff(gotResources, tt.wantAdded); diff != "" {
							t.Errorf("unexpected diff in added roleassignment resources: \ngot: %v\nwant: %v\ndiff: %s", gotResources, tt.wantAdded, diff)
						}
						return nil
					},
				)
			}
			platformWorkloadIdentityRolesByVersion.EXPECT().GetPlatformWorkloadIdentityRolesByRoleName().AnyTimes().Return(tt.roles)
			for _, wantDeleted := range tt.wantDeleted {
				roleAssignments.EXPECT().Delete(gomock.Eq(ctx), gomock.Eq(*wantDeleted.Scope), gomock.Eq(*wantDeleted.Name))
			}

			m := &manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				subscriptionDoc: &api.SubscriptionDocument{
					ID: subscriptionId,
				},
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: tt.oc,
				},
				roleAssignments:                        roleAssignments,
				deployments:                            deployments,
				platformWorkloadIdentityRolesByVersion: platformWorkloadIdentityRolesByVersion,
			}

			err := m.ensurePlatformWorkloadIdentityRBAC(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func workloadIdentityResourceGroupRBAC(roleID, objID string) *arm.Resource {
	return rbac.ResourceGroupRoleAssignmentWithName(
		roleID,
		"'"+objID+"'",
		"guid(resourceGroup().id, '"+roleID+"')",
	)
}
