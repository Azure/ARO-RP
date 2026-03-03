package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	ktesting "k8s.io/client-go/testing"

	"sigs.k8s.io/yaml"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	mock_authorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	"github.com/Azure/ARO-RP/test/util/serversideapply"
)

const fakeClusterSPObjectId = "00000000-0000-0000-0000-000000000000"

func TestCreateOrUpdateClusterServicePrincipalRBAC(t *testing.T) {
	ctx := context.Background()
	clusterRGName := "test-cluster"
	resourceGroupID := fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/%s", clusterRGName)
	assignmentName := "assignment"

	m := &manager{
		log: logrus.NewEntry(logrus.StandardLogger()),
		doc: &api.OpenShiftClusterDocument{
			OpenShiftCluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: resourceGroupID,
					},
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						SPObjectID: fakeClusterSPObjectId,
					},
				},
			},
		},
	}

	for _, tt := range []struct {
		name              string
		clusterSPObjectID string
		roleAssignments   []mgmtauthorization.RoleAssignment
		mocksDeployment   func(*mock_features.MockDeploymentsClient)
		mocksAuthz        func(*mock_authorization.MockRoleAssignmentsClient, *mock_authorization.MockRoleDefinitionsClient, interface{})
	}{
		{
			name: "noop",
			roleAssignments: []mgmtauthorization.RoleAssignment{
				{
					Name: pointerutils.ToPtr(assignmentName),
					RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
						RoleDefinitionID: pointerutils.ToPtr(rbac.RoleContributor),
						Scope:            pointerutils.ToPtr(resourceGroupID),
						PrincipalID:      pointerutils.ToPtr(fakeClusterSPObjectId),
					},
				},
			},
			mocksAuthz: func(roleAssignments *mock_authorization.MockRoleAssignmentsClient, roleDefinitions *mock_authorization.MockRoleDefinitionsClient, result interface{}) {
				roleAssignments.EXPECT().ListForResourceGroup(gomock.Any(), gomock.Any(), gomock.Any()).Return(result, nil)
				roleDefinitions.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			},
		},
		{
			name: "needs create",
			mocksDeployment: func(client *mock_features.MockDeploymentsClient) {
				var parameters map[string]interface{}
				client.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRGName, gomock.Any(), mgmtfeatures.Deployment{
					Properties: &mgmtfeatures.DeploymentProperties{
						Template: &arm.Template{
							Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
							ContentVersion: "1.0.0.0",
							Resources:      []*arm.Resource{m.clusterServicePrincipalRBAC()},
						},
						Parameters: parameters,
						Mode:       mgmtfeatures.Incremental,
					},
				}).Return(nil)
			},
			mocksAuthz: func(roleAssignments *mock_authorization.MockRoleAssignmentsClient, roleDefinitions *mock_authorization.MockRoleDefinitionsClient, result interface{}) {
				roleAssignments.EXPECT().ListForResourceGroup(gomock.Any(), gomock.Any(), gomock.Any()).Return(result, nil)
				roleDefinitions.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			},
		},
		{
			name: "needs delete & create",
			roleAssignments: []mgmtauthorization.RoleAssignment{
				{
					Name: pointerutils.ToPtr(assignmentName),
					RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
						RoleDefinitionID: pointerutils.ToPtr(rbac.RoleContributor),
						Scope:            pointerutils.ToPtr(resourceGroupID),
						PrincipalID:      pointerutils.ToPtr("00000000-0000-0000-0000-000000000001"),
					},
				},
			},
			mocksDeployment: func(client *mock_features.MockDeploymentsClient) {
				var parameters map[string]interface{}
				client.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRGName, gomock.Any(), mgmtfeatures.Deployment{
					Properties: &mgmtfeatures.DeploymentProperties{
						Template: &arm.Template{
							Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
							ContentVersion: "1.0.0.0",
							Resources:      []*arm.Resource{m.clusterServicePrincipalRBAC()},
						},
						Parameters: parameters,
						Mode:       mgmtfeatures.Incremental,
					},
				}).Return(nil)
			},
			mocksAuthz: func(roleAssignments *mock_authorization.MockRoleAssignmentsClient, roleDefinitions *mock_authorization.MockRoleDefinitionsClient, result interface{}) {
				roleAssignments.EXPECT().ListForResourceGroup(gomock.Any(), gomock.Any(), gomock.Any()).Return(result, nil)
				roleAssignments.EXPECT().Delete(gomock.Any(), resourceGroupID, assignmentName)
				roleDefinitions.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			roleAssignments := mock_authorization.NewMockRoleAssignmentsClient(controller)
			roleDefinitions := mock_authorization.NewMockRoleDefinitionsClient(controller)
			deployments := mock_features.NewMockDeploymentsClient(controller)

			if tt.mocksDeployment != nil {
				tt.mocksDeployment(deployments)
			}

			if tt.mocksAuthz != nil {
				tt.mocksAuthz(roleAssignments, roleDefinitions, tt.roleAssignments)
			}

			m.roleAssignments = roleAssignments
			m.roleDefinitions = roleDefinitions
			m.deployments = deployments

			err := m.createOrUpdateClusterServicePrincipalRBAC(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func getFakeAROSecret(clientID, secret string) corev1.Secret {
	name := "azure-cloud-provider"
	namespace := "kube-system"
	data := map[string]interface{}{
		"aadClientId":     clientID,
		"aadClientSecret": secret,
	}
	b, err := yaml.Marshal(data)
	if err != nil {
		panic(err)
	}
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"cloud-config": b,
		},
	}
}

func TestCloudConfigSecretFromChanges(t *testing.T) {
	for _, tt := range []struct {
		name       string
		secretIn   func() *corev1.Secret
		cf         map[string]interface{}
		wantSecret func() *corev1.Secret
		wantErrMsg string
	}{
		{
			name: "New CSP (client ID and client secret both changed)",
			secretIn: func() *corev1.Secret {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return &secret
			},
			cf: map[string]interface{}{
				"aadClientId":     "aadClientIdNew",
				"aadClientSecret": "aadClientSecretNew",
			},
			wantSecret: func() *corev1.Secret {
				secret := getFakeAROSecret("aadClientIdNew", "aadClientSecretNew")
				return &secret
			},
		},
		{
			name: "Updated secret (client ID stayed the same, client secret changed)",
			secretIn: func() *corev1.Secret {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return &secret
			},
			cf: map[string]interface{}{
				"aadClientId":     "aadClientId",
				"aadClientSecret": "aadClientSecretNew",
			},
			wantSecret: func() *corev1.Secret {
				secret := getFakeAROSecret("aadClientId", "aadClientSecretNew")
				return &secret
			},
		},
		{
			name: "No errors, nothing changed",
			secretIn: func() *corev1.Secret {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return &secret
			},
			cf: map[string]interface{}{
				"aadClientId":     "aadClientId",
				"aadClientSecret": "aadClientSecret",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			secret, err := cloudConfigSecretFromChanges(tt.secretIn(), tt.cf)
			if tt.wantSecret != nil {
				wantSecret := tt.wantSecret()
				if secret == nil {
					t.Errorf("Did not return a Secret, but expected the following Secret data: %v", string(wantSecret.Data["cloud-config"]))
				}
				if !reflect.DeepEqual(secret, wantSecret) {
					t.Errorf("\n%+v \n!= \n%+v", string(secret.Data["cloud-config"]), string(wantSecret.Data["cloud-config"]))
				}
			} else if tt.wantSecret == nil && secret != nil {
				t.Errorf("Should not have returned a Secret")
			}
			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)
		})
	}
}

func TestServicePrincipalUpdated(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name          string
		kubernetescli func() *fake.Clientset
		spp           *api.ServicePrincipalProfile
		wantSecret    func() *corev1.Secret
		wantErrMsg    string
	}{
		{
			name: "Secret not found",
			kubernetescli: func() *fake.Clientset {
				return fake.NewSimpleClientset()
			},
			spp: &api.ServicePrincipalProfile{
				ClientID:     "aadClientId",
				ClientSecret: "aadClientSecretNew",
			},
			wantErrMsg: "",
		},
		{
			name: "Encounter other error getting Secret",
			kubernetescli: func() *fake.Clientset {
				cli := fake.NewSimpleClientset()
				cli.CoreV1().(*fakecorev1.FakeCoreV1).PrependReactor("get", "secrets", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &corev1.Secret{}, errors.New("Error getting Secret")
				})
				return cli
			},
			spp: &api.ServicePrincipalProfile{
				ClientID:     "aadClientId",
				ClientSecret: "aadClientSecretNew",
			},
			wantErrMsg: "Error getting Secret",
		},
		{
			name: "Unable to unmarshal cloud-config data",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				secret.Data["cloud-config"] = []byte("This is some random data that is not going to unmarshal properly!")
				return fake.NewSimpleClientset(&secret)
			},
			spp: &api.ServicePrincipalProfile{
				ClientID:     "aadClientId",
				ClientSecret: "aadClientSecretNew",
			},
			wantErrMsg: "error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go value of type map[string]interface {}",
		},
		{
			name: "New CSP (client ID and client secret both changed)",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return fake.NewSimpleClientset(&secret)
			},
			spp: &api.ServicePrincipalProfile{
				ClientID:     "aadClientIdNew",
				ClientSecret: "aadClientSecretNew",
			},
			wantSecret: func() *corev1.Secret {
				secret := getFakeAROSecret("aadClientIdNew", "aadClientSecretNew")
				return &secret
			},
		},
		{
			name: "Updated secret (client ID stayed the same, client secret changed)",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return fake.NewSimpleClientset(&secret)
			},
			spp: &api.ServicePrincipalProfile{
				ClientID:     "aadClientId",
				ClientSecret: "aadClientSecretNew",
			},
			wantSecret: func() *corev1.Secret {
				secret := getFakeAROSecret("aadClientId", "aadClientSecretNew")
				return &secret
			},
		},
		{
			name: "No errors, nothing changed",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return fake.NewSimpleClientset(&secret)
			},
			spp: &api.ServicePrincipalProfile{
				ClientID:     "aadClientId",
				ClientSecret: "aadClientSecret",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &manager{
				kubernetescli: tt.kubernetescli(),
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ServicePrincipalProfile: tt.spp,
						},
					},
				},
			}

			secret, err := m.servicePrincipalUpdated(ctx)
			if tt.wantSecret != nil {
				wantSecret := tt.wantSecret()
				if secret == nil {
					t.Errorf("Did not return a Secret, but expected the following Secret data: %v", string(wantSecret.Data["cloud-config"]))
				}
				if !reflect.DeepEqual(secret, wantSecret) {
					t.Errorf("\n%+v \n!= \n%+v", string(secret.Data["cloud-config"]), string(wantSecret.Data["cloud-config"]))
				}
			} else if tt.wantSecret == nil && secret != nil {
				t.Errorf("Should not have returned a Secret")
			}
			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)
		})
	}
}

func TestUpdateAROSecret(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name          string
		kubernetescli func() *fake.Clientset
		operatorcli   func() *operatorfake.Clientset
		doc           api.OpenShiftCluster
		expect        func() corev1.Secret
	}

	for _, tt := range []*test{
		{
			name: "noop",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return fake.NewSimpleClientset(&secret)
			},
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						ClientID:     "aadClientId",
						ClientSecret: "aadClientSecret",
					},
				},
			},
			operatorcli: func() *operatorfake.Clientset {
				return operatorfake.NewSimpleClientset()
			},
			expect: func() corev1.Secret {
				return getFakeAROSecret("aadClientId", "aadClientSecret")
			},
		},
		{
			name: "update",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeAROSecret("aadClientId", "aadClientSecret")
				return fake.NewSimpleClientset(&secret)
			},
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						ClientID:     "new-client-id",
						ClientSecret: "aadClientSecret",
					},
				},
			},
			operatorcli: func() *operatorfake.Clientset {
				return operatorfake.NewSimpleClientset(
					&operatorv1.KubeAPIServer{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
					},
					&operatorv1.KubeControllerManager{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
					},
				)
			},
			expect: func() corev1.Secret {
				return getFakeAROSecret("new-client-id", "aadClientSecret")
			},
		},
		{
			name: "not found - no fail",
			kubernetescli: func() *fake.Clientset {
				return fake.NewSimpleClientset()
			},
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						ClientID:     "clientID",
						ClientSecret: "aadClientSecret",
					},
				},
			},
			operatorcli: func() *operatorfake.Clientset {
				return operatorfake.NewSimpleClientset()
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &manager{
				kubernetescli: tt.kubernetescli(),
				operatorcli:   tt.operatorcli(),
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &tt.doc,
				},
			}

			err := m.updateAROSecret(ctx)
			if err != nil {
				t.Fatal(err)
			}

			secret, _ := m.kubernetescli.CoreV1().Secrets("kube-system").Get(ctx, "azure-cloud-provider", metav1.GetOptions{})

			if secret != nil {
				expect := tt.expect()
				if !reflect.DeepEqual(secret.Data, expect.Data) {
					t.Errorf("\n%+v \n!= \n%+v", string(secret.Data["cloud-config"]), string(expect.Data["cloud-config"]))
				}
			}
		})
	}
}

func getFakeOpenShiftSecret() corev1.Secret {
	name := "azure-credentials"
	namespace := "kube-system"
	data := map[string][]byte{
		"azure_subscription_id": {},
		"azure_resource_prefix": {},
		"azure_resourcegroup":   {},
		"azure_region":          {},
		"azure_client_id":       []byte("azure_client_id_value"),
		"azure_client_secret":   []byte("azure_client_secret_value"),
		"azure_tenant_id":       []byte("azure_tenant_id_value"),
	}
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

func TestUpdateOpenShiftSecret(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name            string
		kubernetescli   func() *fake.Clientset
		doc             api.OpenShiftCluster
		subscriptionDoc api.SubscriptionDocument
		expect          func() corev1.Secret
		wantErr         string
	}

	for _, tt := range []*test{
		{
			name: "noop",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeOpenShiftSecret()
				return serversideapply.CliWithApply([]string{"secrets"}, &secret)
			},
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						ClientID:     "azure_client_id_value",
						ClientSecret: "azure_client_secret_value",
					},
				},
			},
			subscriptionDoc: api.SubscriptionDocument{
				Subscription: &api.Subscription{
					Properties: &api.SubscriptionProperties{
						TenantID: "azure_tenant_id_value",
					},
				},
			},
			expect: func() corev1.Secret {
				return getFakeOpenShiftSecret()
			},
		},
		{
			name: "update secret",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeOpenShiftSecret()
				return serversideapply.CliWithApply([]string{"secrets"}, &secret)
			},
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						ClientID:     "azure_client_id_value",
						ClientSecret: "new_azure_client_secret_value",
					},
				},
			},
			subscriptionDoc: api.SubscriptionDocument{
				Subscription: &api.Subscription{
					Properties: &api.SubscriptionProperties{
						TenantID: "azure_tenant_id_value",
					},
				},
			},
			expect: func() corev1.Secret {
				secret := getFakeOpenShiftSecret()
				secret.Data["azure_client_secret"] = []byte("new_azure_client_secret_value")
				return secret
			},
		},
		{
			name: "update tenant",
			kubernetescli: func() *fake.Clientset {
				secret := getFakeOpenShiftSecret()
				return serversideapply.CliWithApply([]string{"secrets"}, &secret)
			},
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						ClientID:     "azure_client_id_value",
						ClientSecret: "azure_client_secret_value",
					},
				},
			},
			subscriptionDoc: api.SubscriptionDocument{
				Subscription: &api.Subscription{
					Properties: &api.SubscriptionProperties{
						TenantID: "new_azure_tenant_id_value",
					},
				},
			},
			expect: func() corev1.Secret {
				secret := getFakeOpenShiftSecret()
				secret.Data["azure_tenant_id"] = []byte("new_azure_tenant_id_value")
				return secret
			},
		},
		{
			name: "recreate secret when not found",
			kubernetescli: func() *fake.Clientset {
				return serversideapply.CliWithApply([]string{"secrets"})
			},
			doc: api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						ClientID:     "azure_client_id_value",
						ClientSecret: "azure_client_secret_value",
					},
				},
			},
			subscriptionDoc: api.SubscriptionDocument{
				Subscription: &api.Subscription{
					Properties: &api.SubscriptionProperties{
						TenantID: "azure_tenant_id_value",
					},
				},
			},
			expect: func() corev1.Secret {
				return getFakeOpenShiftSecret()
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &manager{
				kubernetescli: tt.kubernetescli(),
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &tt.doc,
				},
				subscriptionDoc: &tt.subscriptionDoc,
			}

			err := m.updateOpenShiftSecret(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			secret, _ := m.kubernetescli.CoreV1().Secrets("kube-system").Get(ctx, "azure-credentials", metav1.GetOptions{})

			if secret != nil {
				expect := tt.expect()
				if !reflect.DeepEqual(secret.Data, expect.Data) {
					t.Errorf("\n%+v \n!= \n%+v", secret.Data, expect.Data)
				}
			}
		})
	}
}
