package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/api/v20240812preview"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	mock_frontend "github.com/Azure/ARO-RP/pkg/util/mocks/frontend"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/version"
	"github.com/Azure/ARO-RP/pkg/api/util/vms"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

const (
	// defaultAPIVersion is the default ARO API version used in tests.
	defaultAPIVersion = "2024-08-12-preview"
	// mockGuid is a mock GUID used in tests.
	mockGuid = "00000000-0000-0000-0000-000000000001"
	// mockLocation is a mock Azure location used in tests.
	mockLocation = "eastus"
	// mockDomain is a mock domain used in tests.
	mockDomain = "example.aroapp.io"
	// mockPodCIDR and mockServiceCIDR are mock CIDRs used in tests.
	mockPodCIDR = "10.0.0.0/16"
	// mockServiceCIDR is a mock service CIDR used in tests.
	mockServiceCIDR = "10.1.0.0/16"
	// mockVMSize is a mock VM size used in tests.
	mockVMSize = "Standard_D32s_v3"
	// mockMiResourceId is a mock managed identity resource ID used in tests.
	mockMiResourceId = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/not-a-real-group/providers/Microsoft.ManagedIdentity/userAssignedIdentities/not-a-real-mi"
	// mockMiResourceId2 is a mock managed identity resource ID used in tests.
	mockMiResourceId2 = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/not-a-real-group/providers/Microsoft.ManagedIdentity/userAssignedIdentities/not-a-real-mi-2"
	// mockIdentityURL is a mock identity URL used in tests for updating cluster identity.
	mockIdentityURL = "https://bogus.identity.azure.net/subscriptions/00000000-0000-0000-0000-000000000001/resourcegroups/rg/providers/Microsoft.ApiManagement/service/test/credentials?tid=00000000-0000-0000-0000-000000000000&oid=00000000-0000-0000-0000-000000000001&aid=00000000-0000-0000-0000-000000000000"
)

var (
	// defaultVersion is the default OpenShift version used in tests.
	defaultVersion = version.DefaultInstallStream.Version.String()
	// defaultMinorVersion is the minor version of the default OpenShift version.
	defaultMinorVersion = version.DefaultInstallStream.Version.MinorVersion()
	// mockResourceGroupID is a mock resource group ID used in tests.
	mockResourceGroupID = fmt.Sprintf("/subscriptions/%s/resourcegroups/clusterresourcegroup", mockGuid)
	// mockMasterSubnetID and mockWorkerSubnetID are mock subnet IDs used in tests.
	mockMasterSubnetID = fmt.Sprintf("/subscriptions/%s/resourcegroups/network/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockGuid)
	mockWorkerSubnetID = fmt.Sprintf("/subscriptions/%s/resourcegroups/network/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker", mockGuid)
	// mockResourceName is a mock resource name used in tests.
	mockResourceName = "resourceName"
	// mockResourceID is a mock resource ID used in tests.
	mockResourceID = testdatabase.GetResourcePath(mockGuid, mockResourceName)
	// mockCurrentTime is a mock current time used in tests.
	mockCurrentTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	// mockSubscriptionDocument is a mock subscription document used in tests.
	mockSubscriptionDocument = &api.SubscriptionDocument{
		ID: mockGuid,
		Subscription: &api.Subscription{
			State: api.SubscriptionStateRegistered,
			Properties: &api.SubscriptionProperties{
				TenantID: mockGuid,
			},
		},
	}
	unexpectedWorkloadIdentitiesError = fmt.Sprintf(`400: PlatformWorkloadIdentityMismatch: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s'. The required platform workload identities are '[aro-operator cloud-controller-manager cloud-network-config disk-csi-driver file-csi-driver image-registry ingress machine-api]'`, defaultMinorVersion)
	mockSystemDataAPI                 = api.SystemData{
		CreatedBy:          "ExampleUser",
		CreatedByType:      api.CreatedByTypeApplication,
		CreatedAt:          &mockCurrentTime,
		LastModifiedBy:     "ExampleUser",
		LastModifiedByType: api.CreatedByTypeApplication,
		LastModifiedAt:     &mockCurrentTime,
	}
	mockSystemData = &v20240812preview.SystemData{
		CreatedAt:          &mockCurrentTime,
		CreatedBy:          "ExampleUser",
		CreatedByType:      v20240812preview.CreatedByTypeApplication,
		LastModifiedAt:     &mockCurrentTime,
		LastModifiedBy:     "ExampleUser",
		LastModifiedByType: v20240812preview.CreatedByTypeApplication,
	}
)

func getPlatformWorkloadIdentityProfile() map[string]v20240812preview.PlatformWorkloadIdentity {
	return map[string]v20240812preview.PlatformWorkloadIdentity{
		"file-csi-driver":          {ResourceID: mockMiResourceId + "0"},
		"cloud-controller-manager": {ResourceID: mockMiResourceId + "1"},
		"ingress":                  {ResourceID: mockMiResourceId + "2"},
		"image-registry":           {ResourceID: mockMiResourceId + "3"},
		"machine-api":              {ResourceID: mockMiResourceId + "4"},
		"cloud-network-config":     {ResourceID: mockMiResourceId + "5"},
		"aro-operator":             {ResourceID: mockMiResourceId + "6"},
		"disk-csi-driver":          {ResourceID: mockMiResourceId + "7"},
	}
}

func getOpenShiftClusterRequest() *v20240812preview.OpenShiftCluster {
	return &v20240812preview.OpenShiftCluster{
		Location: mockLocation,
		Name:     mockResourceName,
		Properties: v20240812preview.OpenShiftClusterProperties{
			ClusterProfile: v20240812preview.ClusterProfile{
				Version:              defaultVersion,
				Domain:               mockDomain,
				ResourceGroupID:      mockResourceGroupID,
				FipsValidatedModules: v20240812preview.FipsValidatedModulesDisabled,
			},
			NetworkProfile: v20240812preview.NetworkProfile{
				PodCIDR:     mockPodCIDR,
				ServiceCIDR: mockServiceCIDR,
			},
			APIServerProfile: v20240812preview.APIServerProfile{
				Visibility: v20240812preview.VisibilityPrivate,
			},
			IngressProfiles: []v20240812preview.IngressProfile{
				{
					Name:       "default",
					Visibility: v20240812preview.VisibilityPublic,
				},
			},
			MasterProfile: v20240812preview.MasterProfile{
				EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
				VMSize:           vms.VMSize(mockVMSize),
				SubnetID:         mockMasterSubnetID,
			},
			WorkerProfiles: []v20240812preview.WorkerProfile{
				{
					Name:             "worker",
					EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
					VMSize:           vms.VMSize(mockVMSize),
					DiskSizeGB:       128,
					Count:            3,
					SubnetID:         mockWorkerSubnetID,
				},
			},
		},
	}
}

func getServicePrincipalOpenShiftClusterRequest() *v20240812preview.OpenShiftCluster {
	cluster := getOpenShiftClusterRequest()
	cluster.Properties.ServicePrincipalProfile = &v20240812preview.ServicePrincipalProfile{
		ClientID:     mockGuid,
		ClientSecret: mockGuid,
	}
	return cluster
}

func getWorkloadIdentityOpenShiftClusterRequest() *v20240812preview.OpenShiftCluster {
	cluster := getOpenShiftClusterRequest()
	cluster.Identity = &v20240812preview.ManagedServiceIdentity{
		Type: "UserAssigned",
		UserAssignedIdentities: map[string]v20240812preview.UserAssignedIdentity{
			mockMiResourceId: {},
		},
	}
	cluster.Properties.PlatformWorkloadIdentityProfile = &v20240812preview.PlatformWorkloadIdentityProfile{
		PlatformWorkloadIdentities: getPlatformWorkloadIdentityProfile(),
	}
	return cluster
}

func getNewOpenShiftClusterResponse() *v20240812preview.OpenShiftCluster {
	return &v20240812preview.OpenShiftCluster{
		ID:         mockResourceID,
		Name:       mockResourceName,
		Type:       "Microsoft.RedHatOpenShift/openShiftClusters",
		Location:   mockLocation,
		SystemData: &v20240812preview.SystemData{},
		Properties: v20240812preview.OpenShiftClusterProperties{
			ProvisioningState: v20240812preview.ProvisioningStateCreating,
			ClusterProfile: v20240812preview.ClusterProfile{
				Version:              defaultVersion,
				Domain:               mockDomain,
				ResourceGroupID:      mockResourceGroupID,
				FipsValidatedModules: v20240812preview.FipsValidatedModulesDisabled,
			},
			NetworkProfile: v20240812preview.NetworkProfile{
				PodCIDR:      mockPodCIDR,
				ServiceCIDR:  mockServiceCIDR,
				OutboundType: v20240812preview.OutboundTypeLoadbalancer,
				LoadBalancerProfile: &v20240812preview.LoadBalancerProfile{
					ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
						Count: 1,
					},
				},
				PreconfiguredNSG: v20240812preview.PreconfiguredNSGDisabled,
			},
			MasterProfile: v20240812preview.MasterProfile{
				EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
				VMSize:           vms.VMSize(mockVMSize),
				SubnetID:         mockMasterSubnetID,
			},
			WorkerProfiles: []v20240812preview.WorkerProfile{
				{
					Name:             "worker",
					EncryptionAtHost: v20240812preview.EncryptionAtHostDisabled,
					VMSize:           vms.VMSize(mockVMSize),
					DiskSizeGB:       128,
					Count:            3,
					SubnetID:         mockWorkerSubnetID,
				},
			},
			APIServerProfile: v20240812preview.APIServerProfile{
				Visibility: v20240812preview.VisibilityPrivate,
			},
			IngressProfiles: []v20240812preview.IngressProfile{
				{
					Name:       "default",
					Visibility: v20240812preview.VisibilityPublic,
				},
			},
		},
	}
}

func getNewServicePrincipalOpenShiftClusterResponse() *v20240812preview.OpenShiftCluster {
	cluster := getNewOpenShiftClusterResponse()
	cluster.Properties.ServicePrincipalProfile = &v20240812preview.ServicePrincipalProfile{
		ClientID: mockGuid,
	}
	return cluster
}

func getNewWorkloadIdentityOpenShiftClusterResponse() *v20240812preview.OpenShiftCluster {
	cluster := getNewOpenShiftClusterResponse()
	cluster.Identity = &v20240812preview.ManagedServiceIdentity{
		Type: "UserAssigned",
		UserAssignedIdentities: map[string]v20240812preview.UserAssignedIdentity{
			mockMiResourceId: {},
		},
		TenantID: mockGuid,
	}
	cluster.Properties.PlatformWorkloadIdentityProfile = &v20240812preview.PlatformWorkloadIdentityProfile{
		PlatformWorkloadIdentities: getPlatformWorkloadIdentityProfile(),
	}
	return cluster
}

func getExistingServicePrincipalOpenShiftClusterResponse() *v20240812preview.OpenShiftCluster {
	return getNewServicePrincipalOpenShiftClusterResponse()
}

func getExistingWorkloadIdentityOpenShiftClusterResponse() *v20240812preview.OpenShiftCluster {
	cluster := getNewWorkloadIdentityOpenShiftClusterResponse()
	// Since it is an existing cluster, populate the values updated by the backend for Workload Identity clusters
	cluster.Properties.ClusterProfile.OIDCIssuer = (*v20240812preview.OIDCIssuer)(pointerutils.ToPtr(mockGuid))
	for roleName, identity := range cluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
		identity.ObjectID = mockGuid
		identity.ClientID = mockGuid
		cluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[roleName] = identity
	}
	return cluster
}

// getAsynchronousOperationDocument creates an AsyncOperationDocument with the specified initial and current provisioning states.
func getAsynchronousOperationDocument(initialProvisioningState, provisioningState api.ProvisioningState) *api.AsyncOperationDocument {
	return &api.AsyncOperationDocument{
		OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockGuid, "resourceName")),
		AsyncOperation: &api.AsyncOperation{
			InitialProvisioningState: initialProvisioningState,
			ProvisioningState:        provisioningState,
		},
	}
}

// getOpenShiftClusterDocument creates an OpenShiftClusterDocument with the specified provisioning state.
func getOpenShiftClusterDocument(provisioningState, lastProvisioningState, failedProvisioningState api.ProvisioningState) *api.OpenShiftClusterDocument {
	return &api.OpenShiftClusterDocument{
		Key:                       strings.ToLower(mockResourceID),
		ClusterResourceGroupIDKey: strings.ToLower(fmt.Sprintf("/subscriptions/%s/resourcegroups/clusterresourcegroup", mockGuid)),
		Bucket:                    1,
		OpenShiftCluster: &api.OpenShiftCluster{
			ID:       mockResourceID,
			Name:     mockResourceName,
			Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
			Location: mockLocation,
			Properties: api.OpenShiftClusterProperties{
				ArchitectureVersion:     version.InstallArchitectureVersion,
				ProvisioningState:       provisioningState,
				LastProvisioningState:   lastProvisioningState,
				FailedProvisioningState: failedProvisioningState,
				ProvisionedBy:           version.GitCommit,
				CreatedAt:               mockCurrentTime,
				CreatedBy:               version.GitCommit,
				ClusterProfile: api.ClusterProfile{
					Version:              defaultVersion,
					Domain:               mockDomain,
					ResourceGroupID:      mockResourceGroupID,
					FipsValidatedModules: api.FipsValidatedModulesDisabled,
				},
				NetworkProfile: api.NetworkProfile{
					PodCIDR:      mockPodCIDR,
					ServiceCIDR:  mockServiceCIDR,
					OutboundType: api.OutboundTypeLoadbalancer,
					LoadBalancerProfile: &api.LoadBalancerProfile{
						ManagedOutboundIPs: &api.ManagedOutboundIPs{
							Count: 1,
						},
					},
					PreconfiguredNSG: api.PreconfiguredNSGDisabled,
				},
				MasterProfile: api.MasterProfile{
					EncryptionAtHost: api.EncryptionAtHostDisabled,
					VMSize:           vms.VMSize(mockVMSize),
					SubnetID:         mockMasterSubnetID,
				},
				WorkerProfiles: []api.WorkerProfile{
					{
						Name:             "worker",
						EncryptionAtHost: api.EncryptionAtHostDisabled,
						VMSize:           vms.VMSize(mockVMSize),
						DiskSizeGB:       128,
						Count:            3,
						SubnetID:         mockWorkerSubnetID,
					},
				},
				APIServerProfile: api.APIServerProfile{
					Visibility: api.VisibilityPrivate,
				},
				IngressProfiles: []api.IngressProfile{
					{
						Name:       "default",
						Visibility: api.VisibilityPublic,
					},
				},
				FeatureProfile: api.FeatureProfile{
					GatewayEnabled: true,
				},
				OperatorFlags: operator.DefaultOperatorFlags(),
			},
		},
	}
}

func getServicePrincipalOpenShiftClusterDocument(provisioningState, lastProvisioningState, failedProvisioningState api.ProvisioningState) *api.OpenShiftClusterDocument {
	doc := getOpenShiftClusterDocument(provisioningState, lastProvisioningState, failedProvisioningState)
	doc.ClientIDKey = mockGuid
	doc.OpenShiftCluster.Properties.ServicePrincipalProfile = &api.ServicePrincipalProfile{
		ClientID:     mockGuid,
		ClientSecret: mockGuid,
	}
	return doc
}

func getExistingServicePrincipalOpenShiftClusterDocument(provisioningState, lastProvisioningState, failedProvisioningState api.ProvisioningState) *api.OpenShiftClusterDocument {
	doc := getServicePrincipalOpenShiftClusterDocument(provisioningState, lastProvisioningState, failedProvisioningState)
	// FakeClusterConflictChecker will not allow the update to proceed if the ClientIDKey & ClusterResourceGroupIDKey is set.
	doc.ClientIDKey = ""
	doc.ClusterResourceGroupIDKey = ""
	return doc
}

func getWorkloadIdentityOpenShiftClusterDocument(provisioningState, lastProvisioningState, failedProvisioningState api.ProvisioningState) *api.OpenShiftClusterDocument {
	doc := getOpenShiftClusterDocument(provisioningState, lastProvisioningState, failedProvisioningState)
	doc.ClientIDKey = strings.ToLower(mockMiResourceId)
	doc.OpenShiftCluster.Identity = &api.ManagedServiceIdentity{
		Type: "UserAssigned",
		UserAssignedIdentities: map[string]api.UserAssignedIdentity{
			mockMiResourceId: {},
		},
		IdentityURL: middleware.MockIdentityURL,
		TenantID:    mockGuid,
	}
	doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile = &api.PlatformWorkloadIdentityProfile{
		PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
			"file-csi-driver": {
				ResourceID: mockMiResourceId + "0",
			},
			"cloud-controller-manager": {
				ResourceID: mockMiResourceId + "1",
			},
			"ingress": {
				ResourceID: mockMiResourceId + "2",
			},
			"image-registry": {
				ResourceID: mockMiResourceId + "3",
			},
			"machine-api": {
				ResourceID: mockMiResourceId + "4",
			},
			"cloud-network-config": {
				ResourceID: mockMiResourceId + "5",
			},
			"aro-operator": {
				ResourceID: mockMiResourceId + "6",
			},
			"disk-csi-driver": {
				ResourceID: mockMiResourceId + "7",
			},
		},
	}
	return doc
}

func getExistingWorkloadIdentityOpenShiftClusterDocument(provisioningState, lastProvisioningState, failedProvisioningState api.ProvisioningState) *api.OpenShiftClusterDocument {
	doc := getWorkloadIdentityOpenShiftClusterDocument(provisioningState, lastProvisioningState, failedProvisioningState)
	// FakeClusterConflictChecker will not allow the update to proceed if the ClientIDKey & ClusterResourceGroupIDKey is set.
	doc.ClientIDKey = ""
	doc.ClusterResourceGroupIDKey = ""
	// Populate the values updated by the backend for Workload Identity clusters
	doc.OpenShiftCluster.Properties.ClusterProfile.OIDCIssuer = (*api.OIDCIssuer)(pointerutils.ToPtr(mockGuid))
	doc.OpenShiftCluster.Properties.ClusterProfile.BoundServiceAccountSigningKey = (*api.SecureString)(pointerutils.ToPtr(mockGuid))
	for roleName, identity := range doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
		identity.ObjectID = mockGuid
		identity.ClientID = mockGuid
		doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[roleName] = identity
	}
	return doc
}

// getMIWIUpgradeableToVersion returns the next version of MIWI that is upgradeable from the default version.
// This function simulates the logic to determine the next version based on the default version.
// It increments the minor version of the default version by 1.
func getMIWIUpgradeableToVersion() version.Version {
	ver, _ := version.DefaultInstallStream.Version.Components()
	return version.NewVersion(ver[0], ver[1]+1, ver[2])
}

// getOCPVersionsChangeFeed returns a map of OpenShift versions for testing purposes.
func getOCPVersionsChangeFeed() map[string]*api.OpenShiftVersion {
	return map[string]*api.OpenShiftVersion{
		defaultVersion: {
			Properties: api.OpenShiftVersionProperties{
				Version: defaultVersion,
				Enabled: true,
				Default: true,
			},
		},
		getMIWIUpgradeableToVersion().String(): {
			Properties: api.OpenShiftVersionProperties{
				Version: getMIWIUpgradeableToVersion().String(),
				Enabled: true,
				Default: false,
			},
		},
	}
}

// getPlatformWorkloadIdentityRolesChangeFeed returns a map of platform workload identity roles for testing purposes.
func getPlatformWorkloadIdentityRolesChangeFeed() map[string]*api.PlatformWorkloadIdentityRoleSet {
	return map[string]*api.PlatformWorkloadIdentityRoleSet{
		defaultMinorVersion: {
			Properties: api.PlatformWorkloadIdentityRoleSetProperties{
				OpenShiftVersion: defaultMinorVersion,
				PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
					{OperatorName: "cloud-controller-manager"},
					{OperatorName: "ingress"},
					{OperatorName: "machine-api"},
					{OperatorName: "disk-csi-driver"},
					{OperatorName: "cloud-network-config"},
					{OperatorName: "image-registry"},
					{OperatorName: "file-csi-driver"},
					{OperatorName: "aro-operator"},
				},
			},
		},
		getMIWIUpgradeableToVersion().MinorVersion(): {
			Properties: api.PlatformWorkloadIdentityRoleSetProperties{
				OpenShiftVersion: getMIWIUpgradeableToVersion().MinorVersion(),
				PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
					{OperatorName: "cloud-controller-manager"},
					{OperatorName: "ingress"},
					{OperatorName: "machine-api"},
					{OperatorName: "disk-csi-driver"},
					{OperatorName: "cloud-network-config"},
					{OperatorName: "image-registry"},
					{OperatorName: "file-csi-driver"},
					{OperatorName: "aro-operator"},
					{OperatorName: "extra-new-operator"},
				},
			},
		},
	}
}

// TestPutorPatchOpenShiftClusterCreate contains the logic to test the operations for creating an OpenShift cluster.
// The test should validate that the operation behaves as expected, including error handling and response validation.
func TestPutorPatchOpenShiftClusterCreate(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name                    string
		request                 func() *v20240812preview.OpenShiftCluster
		fixture                 func(*testdatabase.Fixture)
		quotaValidatorError     error
		skuValidatorError       error
		providersValidatorError error
		wantSystemDataEnriched  bool
		wantDocuments           func(*testdatabase.Checker)
		wantStatusCode          int
		wantResponse            *v20240812preview.OpenShiftCluster
		wantAsync               bool
		wantError               string
	}{
		{
			name: "create a new OpenShift Service Principal cluster",
			request: func() *v20240812preview.OpenShiftCluster {
				return getServicePrincipalOpenShiftClusterRequest()
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
			},
			wantAsync:              true,
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateCreating, api.ProvisioningStateCreating))
				checker.AddOpenShiftClusterDocuments(getServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateCreating, "", ""))
			},
			wantStatusCode: http.StatusCreated,
			wantResponse:   getNewServicePrincipalOpenShiftClusterResponse(),
		},
		{
			name: "create a new OpenShift Workload Identity cluster",
			request: func() *v20240812preview.OpenShiftCluster {
				return getWorkloadIdentityOpenShiftClusterRequest()
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
			},
			wantAsync:              true,
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateCreating, api.ProvisioningStateCreating))
				checker.AddOpenShiftClusterDocuments(getWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateCreating, "", ""))
			},
			wantStatusCode: http.StatusCreated,
			wantResponse:   getNewWorkloadIdentityOpenShiftClusterResponse(),
		},
		{
			name: "create a new OpenShift Workload Identity cluster - unexpected workload identity provided",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := getWorkloadIdentityOpenShiftClusterRequest()
				delete(cluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities, "aro-operator")
				cluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["unexpected-operator"] = v20240812preview.PlatformWorkloadIdentity{ResourceID: mockMiResourceId}
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
			},
			wantSystemDataEnriched: true,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              unexpectedWorkloadIdentitiesError,
		},
		{
			name: "create a new OpenShift Workload Identity cluster - missing workload identity provided",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := getWorkloadIdentityOpenShiftClusterRequest()
				delete(cluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities, "aro-operator")
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
			},
			wantSystemDataEnriched: true,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              unexpectedWorkloadIdentitiesError,
		},
		{
			name: "create a new OpenShift Workload Identity cluster - extra workload identity provided",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := getWorkloadIdentityOpenShiftClusterRequest()
				cluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["extra-operator"] = v20240812preview.PlatformWorkloadIdentity{ResourceID: mockMiResourceId}
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
			},
			wantSystemDataEnriched: true,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              unexpectedWorkloadIdentitiesError,
		},
		{
			name: "create a new OpenShift cluster - vm not supported",
			request: func() *v20240812preview.OpenShiftCluster {
				return getServicePrincipalOpenShiftClusterRequest()
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
			},
			quotaValidatorError: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("The provided VM SKU %s is not supported.", "something")),
			wantStatusCode:      http.StatusBadRequest,
			wantError:           "400: InvalidParameter: : The provided VM SKU something is not supported.",
		},
		{
			name: "create a new OpenShift cluster - quota fails",
			request: func() *v20240812preview.OpenShiftCluster {
				return getServicePrincipalOpenShiftClusterRequest()
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
			},
			quotaValidatorError: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeQuotaExceeded, "", "Resource quota of vm exceeded. Maximum allowed: 0, Current in use: 0, Additional requested: 1."),
			wantStatusCode:      http.StatusBadRequest,
			wantError:           "400: QuotaExceeded: : Resource quota of vm exceeded. Maximum allowed: 0, Current in use: 0, Additional requested: 1.",
		},
		{
			name: "create a new OpenShift cluster - sku unavailable",
			request: func() *v20240812preview.OpenShiftCluster {
				return getServicePrincipalOpenShiftClusterRequest()
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
			},
			skuValidatorError: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("The selected SKU '%v' is unavailable in region '%v'", "Standard_Sku", "somewhere")),
			wantStatusCode:    http.StatusBadRequest,
			wantError:         "400: InvalidParameter: : The selected SKU 'Standard_Sku' is unavailable in region 'somewhere'",
		},
		{
			name: "create a new OpenShift cluster - sku restricted",
			request: func() *v20240812preview.OpenShiftCluster {
				return getServicePrincipalOpenShiftClusterRequest()
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
			},
			skuValidatorError: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("The selected SKU '%v' is restricted in region '%v' for selected subscription", "Standard_Sku", "somewhere")),
			wantStatusCode:    http.StatusBadRequest,
			wantError:         "400: InvalidParameter: : The selected SKU 'Standard_Sku' is restricted in region 'somewhere' for selected subscription",
		},
		{
			name: "create a new OpenShift cluster - Microsoft.Authorization provider not registered",
			request: func() *v20240812preview.OpenShiftCluster {
				return getServicePrincipalOpenShiftClusterRequest()
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
			},
			providersValidatorError: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceProviderNotRegistered, "", fmt.Sprintf("The resource provider '%s' is not registered.", "Microsoft.Authorization")),
			wantStatusCode:          http.StatusBadRequest,
			wantError:               "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Authorization' is not registered.",
		},
		{
			name: "create a new OpenShift cluster - Microsoft.Compute provider not registered",
			request: func() *v20240812preview.OpenShiftCluster {
				return getServicePrincipalOpenShiftClusterRequest()
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
			},
			providersValidatorError: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceProviderNotRegistered, "", fmt.Sprintf("The resource provider '%s' is not registered.", "Microsoft.Compute")),
			wantStatusCode:          http.StatusBadRequest,
			wantError:               "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Compute' is not registered.",
		},
		{
			name: "create a new OpenShift cluster - Microsoft.Network provider not registered",
			request: func() *v20240812preview.OpenShiftCluster {
				return getServicePrincipalOpenShiftClusterRequest()
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
			},
			providersValidatorError: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceProviderNotRegistered, "", fmt.Sprintf("The resource provider '%s' is not registered.", "Microsoft.Network")),
			wantStatusCode:          http.StatusBadRequest,
			wantError:               "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Network' is not registered.",
		},
		{
			name: "create a new OpenShift Service Principal cluster - fail as provided cluster resource group already contains a cluster",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := getServicePrincipalOpenShiftClusterRequest()
				cluster.Properties.ServicePrincipalProfile.ClientID = "11111111-1111-1111-1111-111111111111"
				cluster.Name = "otherresourcename" // Different name to avoid conflict with the fixture
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              fmt.Sprintf("400: DuplicateResourceGroup: : The provided resource group '/subscriptions/%s/resourcegroups/clusterresourcegroup' already contains a cluster.", mockGuid),
		},
		{
			name: "create a new OpenShift Service Principal cluster - fail as provided clientID is not unique",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := getServicePrincipalOpenShiftClusterRequest()
				cluster.Name = "otherresourcename" // Different name to avoid conflict with the fixture
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              fmt.Sprintf("400: DuplicateClientID: : The provided service principal with client ID '%s' is already in use by a cluster.", mockGuid),
		},
		{
			name: "create a new OpenShift Workload Identity cluster - fail as provided clientID is not unique",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := getWorkloadIdentityOpenShiftClusterRequest()
				cluster.Identity = &v20240812preview.ManagedServiceIdentity{
					Type: "UserAssigned",
					UserAssignedIdentities: map[string]v20240812preview.UserAssignedIdentity{
						mockMiResourceId: {},
					},
				}
				cluster.Name = "otherresourcename" // Different name to avoid conflict with the fixture
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantAsync:              true,
			wantSystemDataEnriched: true,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              fmt.Sprintf("400: DuplicateClientID: : The provided user assigned identity '%s' is already in use by a cluster.", mockMiResourceId),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).
				WithOpenShiftClusters().
				WithSubscriptions().
				WithAsyncOperations().
				WithOpenShiftVersions()
			defer ti.done()

			controller := gomock.NewController(t)
			defer controller.Finish()

			mockQuotaValidator := mock_frontend.NewMockQuotaValidator(controller)
			mockQuotaValidator.EXPECT().ValidateQuota(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.quotaValidatorError).AnyTimes()

			mockSkuValidator := mock_frontend.NewMockSkuValidator(controller)
			mockSkuValidator.EXPECT().ValidateVMSku(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.skuValidatorError).AnyTimes()

			mockProvidersValidator := mock_frontend.NewMockProvidersValidator(controller)
			mockProvidersValidator.EXPECT().ValidateProviders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.providersValidatorError).AnyTimes()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, ti.enricher)
			if err != nil {
				t.Fatal(err)
			}
			f.bucketAllocator = bucket.Fixed(1)
			f.now = func() time.Time { return mockCurrentTime }

			f.quotaValidator = mockQuotaValidator
			f.skuValidator = mockSkuValidator
			f.providersValidator = mockProvidersValidator

			var systemDataClusterDocEnricherCalled bool
			f.systemDataClusterDocEnricher = func(doc *api.OpenShiftClusterDocument, systemData *api.SystemData) {
				systemDataClusterDocEnricherCalled = true
			}

			go f.Run(ctx, nil, nil)
			f.ocpVersionsMu.Lock()
			f.enabledOcpVersions = getOCPVersionsChangeFeed()
			for key, doc := range getOCPVersionsChangeFeed() {
				if doc.Properties.Default {
					f.defaultOcpVersion = key
				}
			}
			f.ocpVersionsMu.Unlock()

			f.platformWorkloadIdentityRoleSetsMu.Lock()
			f.availablePlatformWorkloadIdentityRoleSets = getPlatformWorkloadIdentityRolesChangeFeed()
			f.platformWorkloadIdentityRoleSetsMu.Unlock()

			oc := tt.request()
			requestHeaders := http.Header{
				"Content-Type": []string{"application/json"},
			}

			var internal api.OpenShiftCluster
			f.apis[defaultAPIVersion].OpenShiftClusterConverter.ToInternal(oc, &internal)
			if internal.UsesWorkloadIdentity() {
				requestHeaders.Add(middleware.MsiIdentityURLHeader, middleware.MockIdentityURL)
				requestHeaders.Add(middleware.MsiTenantHeader, mockGuid)
			}

			resp, b, err := ti.request(http.MethodPut,
				"https://server"+testdatabase.GetResourcePath(mockGuid, oc.Name)+"?api-version=2024-08-12-preview",
				requestHeaders,
				oc,
			)
			if err != nil {
				t.Error(err, b)
			}

			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("https://localhost:8443/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockGuid, ti.env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}

			if tt.wantDocuments != nil {
				tt.wantDocuments(ti.checker)
				errs := ti.checker.CheckOpenShiftClusters(ti.openShiftClustersClient)
				errs = append(errs, ti.checker.CheckAsyncOperations(ti.asyncOperationsClient)...)
				for _, err := range errs {
					t.Error(err)
				}
			}

			if tt.wantSystemDataEnriched != systemDataClusterDocEnricherCalled {
				t.Error(systemDataClusterDocEnricherCalled)
			}
		})
	}
}

// TestPutorPatchOpenShiftClusterUpdatePut contains the logic to test the operations for updating an OpenShift cluster using PUT.
// The test should validate that the operation behaves as expected, including error handling and response validation.
func TestPutorPatchOpenShiftClusterUpdatePut(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name                    string
		request                 func() *v20240812preview.OpenShiftCluster
		fixture                 func(*testdatabase.Fixture)
		quotaValidatorError     error
		skuValidatorError       error
		providersValidatorError error
		wantSystemDataEnriched  bool
		wantDocuments           func(*testdatabase.Checker)
		wantStatusCode          int
		wantResponse            func() *v20240812preview.OpenShiftCluster
		wantAsync               bool
		wantError               string
	}{
		{
			name: "update a cluster from succeeded",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := getServicePrincipalOpenShiftClusterRequest()
				// OutboundType is set to the current value
				cluster.Properties.NetworkProfile.OutboundType = v20240812preview.OutboundTypeLoadbalancer
				// PreconfiguredNSG is set to the current value
				cluster.Properties.NetworkProfile.PreconfiguredNSG = v20240812preview.PreconfiguredNSGDisabled
				// Update the LoadBalancerProfile to have 2 ManagedOutboundIPs
				cluster.Properties.NetworkProfile.LoadBalancerProfile = &v20240812preview.LoadBalancerProfile{
					ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
						Count: 2,
					},
				}
				// Update the Tags to ensure they are changed
				cluster.Tags = map[string]string{"tag": "tag"}
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getExistingServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", "")
				doc.OpenShiftCluster.Tags = map[string]string{"tag": "will-not-be-kept"}
				doc.OpenShiftCluster.SystemData = mockSystemDataAPI
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusOK,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateUpdating, api.ProvisioningStateUpdating))
				doc := getExistingServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.SystemData = mockSystemDataAPI
				doc.OpenShiftCluster.Tags = map[string]string{"tag": "tag"}
				// LoadBalancerProfile is updated with 2 ManagedOutboundIPs
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile = &api.LoadBalancerProfile{
					ManagedOutboundIPs: &api.ManagedOutboundIPs{
						Count: 2,
					},
				}
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				response := getExistingServicePrincipalOpenShiftClusterResponse()
				// SystemData won't be changed by the update operation, so we set it to the mockSystemDataAPI.
				response.SystemData = mockSystemData
				// Tags and ManagedOutboundIPs count are updated.
				response.Tags = map[string]string{"tag": "tag"}
				response.Properties.NetworkProfile.LoadBalancerProfile = &v20240812preview.LoadBalancerProfile{
					ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
						Count: 2,
					},
				}
				// ProvisioningState is set to Updating
				response.Properties.ProvisioningState = v20240812preview.ProvisioningStateUpdating
				return response
			},
		},
		{
			name: "update a cluster from a failed during update",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := getServicePrincipalOpenShiftClusterRequest()
				// OutboundType is set to the current value
				cluster.Properties.NetworkProfile.OutboundType = v20240812preview.OutboundTypeLoadbalancer
				// PreconfiguredNSG is set to the current value
				cluster.Properties.NetworkProfile.PreconfiguredNSG = v20240812preview.PreconfiguredNSGDisabled
				// LoadBalancerProfile to have 1 ManagedOutboundIPs, i.e. current value
				cluster.Properties.NetworkProfile.LoadBalancerProfile = &v20240812preview.LoadBalancerProfile{
					ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
						Count: 1,
					},
				}
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateFailed, "", api.ProvisioningStateUpdating))
			},
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusOK,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateUpdating, api.ProvisioningStateUpdating))
				doc := getExistingServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateUpdating, "", api.ProvisioningStateUpdating)
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				doc.OpenShiftCluster.Properties.LastProvisioningState = api.ProvisioningStateFailed
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				response := getExistingServicePrincipalOpenShiftClusterResponse()
				// ProvisioningState is set to Updating
				response.Properties.ProvisioningState = v20240812preview.ProvisioningStateUpdating
				return response
			},
		},
		{
			name: "update a cluster from failed during creation",
			request: func() *v20240812preview.OpenShiftCluster {
				return getServicePrincipalOpenShiftClusterRequest()
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateFailed, "", api.ProvisioningStateCreating))
			},
			wantStatusCode: http.StatusBadRequest,
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				return nil
			},
			wantError: "400: RequestNotAllowed: : Request is not allowed on cluster whose creation failed. Delete the cluster.",
		},
		{
			name: "update a cluster from failed during deletion",
			request: func() *v20240812preview.OpenShiftCluster {
				return getServicePrincipalOpenShiftClusterRequest()
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateFailed, "", api.ProvisioningStateDeleting))
			},
			wantStatusCode: http.StatusBadRequest,
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				return nil
			},
			wantError: "400: RequestNotAllowed: : Request is not allowed on cluster whose deletion failed. Delete the cluster.",
		},
		{
			name: "update a Workload Identity cluster from succeeded",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := getWorkloadIdentityOpenShiftClusterRequest()
				// OutboundType is set to the current value
				cluster.Properties.NetworkProfile.OutboundType = v20240812preview.OutboundTypeLoadbalancer
				// PreconfiguredNSG is set to the current value
				cluster.Properties.NetworkProfile.PreconfiguredNSG = v20240812preview.PreconfiguredNSGDisabled
				// LoadBalancerProfile is set to the current value
				cluster.Properties.NetworkProfile.LoadBalancerProfile = &v20240812preview.LoadBalancerProfile{
					ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
						Count: 1,
					},
				}
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusOK,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateUpdating, api.ProvisioningStateUpdating))
				doc := getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				response := getExistingWorkloadIdentityOpenShiftClusterResponse()
				// ProvisioningState is set to Updating
				response.Properties.ProvisioningState = v20240812preview.ProvisioningStateUpdating
				return response
			},
		},
		{
			name: "update a Workload Identity cluster from succeeded - set UpgradeableTo with new identity",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := getWorkloadIdentityOpenShiftClusterRequest()
				// OutboundType is set to the current value
				cluster.Properties.NetworkProfile.OutboundType = v20240812preview.OutboundTypeLoadbalancer
				// PreconfiguredNSG is set to the current value
				cluster.Properties.NetworkProfile.PreconfiguredNSG = v20240812preview.PreconfiguredNSGDisabled
				// LoadBalancerProfile is set to the current value
				cluster.Properties.NetworkProfile.LoadBalancerProfile = &v20240812preview.LoadBalancerProfile{
					ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
						Count: 1,
					},
				}
				// Set UpgradeableTo with new identity
				cluster.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo = pointerutils.ToPtr(v20240812preview.UpgradeableTo(getMIWIUpgradeableToVersion().String()))
				cluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["extra-new-operator"] = v20240812preview.PlatformWorkloadIdentity{
					ResourceID: mockMiResourceId,
				}
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusOK,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateUpdating, api.ProvisioningStateUpdating))
				doc := getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				// Set UpgradeableTo with new identity
				doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo = pointerutils.ToPtr(api.UpgradeableTo(getMIWIUpgradeableToVersion().String()))
				doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["extra-new-operator"] = api.PlatformWorkloadIdentity{
					ResourceID: mockMiResourceId,
				}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				response := getExistingWorkloadIdentityOpenShiftClusterResponse()
				// Set UpgradeableTo with new identity
				response.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo = pointerutils.ToPtr(v20240812preview.UpgradeableTo(getMIWIUpgradeableToVersion().String()))
				response.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["extra-new-operator"] = v20240812preview.PlatformWorkloadIdentity{
					ResourceID: mockMiResourceId,
				}
				// ProvisioningState is set to Updating
				response.Properties.ProvisioningState = v20240812preview.ProvisioningStateUpdating
				return response
			},
		},
		{
			name: "Fail - update a Workload Identity cluster from succeeded - pass existing issuerURL in the body",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := getWorkloadIdentityOpenShiftClusterRequest()
				// OutboundType is set to the current value
				cluster.Properties.NetworkProfile.OutboundType = v20240812preview.OutboundTypeLoadbalancer
				// PreconfiguredNSG is set to the current value
				cluster.Properties.NetworkProfile.PreconfiguredNSG = v20240812preview.PreconfiguredNSGDisabled
				// LoadBalancerProfile is set to the current value
				cluster.Properties.NetworkProfile.LoadBalancerProfile = &v20240812preview.LoadBalancerProfile{
					ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
						Count: 1,
					},
				}
				// Set UpgradeableTo with new identity
				cluster.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo = pointerutils.ToPtr(v20240812preview.UpgradeableTo(getMIWIUpgradeableToVersion().String()))
				cluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["extra-new-operator"] = v20240812preview.PlatformWorkloadIdentity{
					ResourceID: mockMiResourceId,
				}
				cluster.Properties.ClusterProfile.OIDCIssuer = (*v20240812preview.OIDCIssuer)(pointerutils.ToPtr(mockGuid))
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				return nil
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: PropertyChangeNotAllowed: properties.clusterProfile.oidcIssuer: Changing property 'properties.clusterProfile.oidcIssuer' is not allowed.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).
				WithOpenShiftClusters().
				WithSubscriptions().
				WithAsyncOperations().
				WithOpenShiftVersions()
			defer ti.done()

			controller := gomock.NewController(t)
			defer controller.Finish()

			mockQuotaValidator := mock_frontend.NewMockQuotaValidator(controller)
			mockQuotaValidator.EXPECT().ValidateQuota(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.quotaValidatorError).AnyTimes()

			mockSkuValidator := mock_frontend.NewMockSkuValidator(controller)
			mockSkuValidator.EXPECT().ValidateVMSku(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.skuValidatorError).AnyTimes()

			mockProvidersValidator := mock_frontend.NewMockProvidersValidator(controller)
			mockProvidersValidator.EXPECT().ValidateProviders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.providersValidatorError).AnyTimes()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, ti.enricher)
			if err != nil {
				t.Fatal(err)
			}
			f.bucketAllocator = bucket.Fixed(1)
			f.now = func() time.Time { return mockCurrentTime }

			f.quotaValidator = mockQuotaValidator
			f.skuValidator = mockSkuValidator
			f.providersValidator = mockProvidersValidator

			var systemDataClusterDocEnricherCalled bool
			f.systemDataClusterDocEnricher = func(doc *api.OpenShiftClusterDocument, systemData *api.SystemData) {
				enrichClusterSystemData(doc, systemData)
				systemDataClusterDocEnricherCalled = true
			}

			go f.Run(ctx, nil, nil)
			f.ocpVersionsMu.Lock()
			f.enabledOcpVersions = getOCPVersionsChangeFeed()
			for key, doc := range getOCPVersionsChangeFeed() {
				if doc.Properties.Default {
					f.defaultOcpVersion = key
				}
			}
			f.ocpVersionsMu.Unlock()

			f.platformWorkloadIdentityRoleSetsMu.Lock()
			f.availablePlatformWorkloadIdentityRoleSets = getPlatformWorkloadIdentityRolesChangeFeed()
			f.platformWorkloadIdentityRoleSetsMu.Unlock()

			oc := tt.request()
			requestHeaders := http.Header{
				"Content-Type": []string{"application/json"},
			}

			var internal api.OpenShiftCluster
			f.apis[defaultAPIVersion].OpenShiftClusterConverter.ToInternal(oc, &internal)
			if internal.UsesWorkloadIdentity() {
				requestHeaders.Add(middleware.MsiIdentityURLHeader, middleware.MockIdentityURL)
				requestHeaders.Add(middleware.MsiTenantHeader, mockGuid)
			}

			resp, b, err := ti.request(http.MethodPut,
				"https://server"+testdatabase.GetResourcePath(mockGuid, oc.Name)+"?api-version=2024-08-12-preview",
				requestHeaders,
				oc,
			)
			if err != nil {
				t.Error(err, b)
			}

			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("https://localhost:8443/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockGuid, ti.env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse())
			if err != nil {
				t.Error(err)
			}

			if tt.wantDocuments != nil {
				tt.wantDocuments(ti.checker)
				errs := ti.checker.CheckOpenShiftClusters(ti.openShiftClustersClient)
				errs = append(errs, ti.checker.CheckAsyncOperations(ti.asyncOperationsClient)...)
				for _, err := range errs {
					t.Error(err)
				}
			}

			if tt.wantSystemDataEnriched != systemDataClusterDocEnricherCalled {
				t.Error(systemDataClusterDocEnricherCalled)
			}
		})
	}
}

// TestPutorPatchOpenShiftClusterUpdatePatch contains the logic to test the operations for updating an OpenShift cluster using PATCH.
// The test should validate that the operation behaves as expected, including error handling and response validation.
func TestPutorPatchOpenShiftClusterUpdatePatch(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name                    string
		request                 func() *v20240812preview.OpenShiftCluster
		fixture                 func(*testdatabase.Fixture)
		headers                 map[string]string
		quotaValidatorError     error
		skuValidatorError       error
		providersValidatorError error
		wantSystemDataEnriched  bool
		wantDocuments           func(*testdatabase.Checker)
		wantStatusCode          int
		wantResponse            func() *v20240812preview.OpenShiftCluster
		wantAsync               bool
		wantError               string
	}{
		{
			name: "patch a cluster from succeeded",
			request: func() *v20240812preview.OpenShiftCluster {
				return &v20240812preview.OpenShiftCluster{
					// Update the LoadBalancerProfile to have 2 ManagedOutboundIPs
					Properties: v20240812preview.OpenShiftClusterProperties{
						NetworkProfile: v20240812preview.NetworkProfile{
							LoadBalancerProfile: &v20240812preview.LoadBalancerProfile{
								ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
									Count: 2,
								},
							},
						},
					},
					// Update the Tags to ensure they are changed
					Tags: map[string]string{"tag": "tag"},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getExistingServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", "")
				doc.OpenShiftCluster.Tags = map[string]string{"tag": "will-not-be-kept"}
				doc.OpenShiftCluster.SystemData = mockSystemDataAPI
				f.AddOpenShiftClusterDocuments(doc)
			},

			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateUpdating, api.ProvisioningStateUpdating))
				doc := getExistingServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.SystemData = mockSystemDataAPI
				doc.OpenShiftCluster.Tags = map[string]string{"tag": "tag"}
				// LoadBalancerProfile is updated with 2 ManagedOutboundIPs
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile = &api.LoadBalancerProfile{
					ManagedOutboundIPs: &api.ManagedOutboundIPs{
						Count: 2,
					},
				}
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				response := getExistingServicePrincipalOpenShiftClusterResponse()
				// SystemData won't be changed by the update operation, so we set it to the mockSystemDataAPI.
				response.SystemData = mockSystemData
				// Tags and ManagedOutboundIPs count are updated.
				response.Tags = map[string]string{"tag": "tag"}
				response.Properties.NetworkProfile.LoadBalancerProfile = &v20240812preview.LoadBalancerProfile{
					ManagedOutboundIPs: &v20240812preview.ManagedOutboundIPs{
						Count: 2,
					},
				}
				// ProvisioningState is set to Updating
				response.Properties.ProvisioningState = v20240812preview.ProvisioningStateUpdating
				return response
			},
		},
		{
			name: "patch a workload identity cluster from succeeded - set UpgradeableTo with new identity",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := &v20240812preview.OpenShiftCluster{
					Properties: v20240812preview.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &v20240812preview.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
								"extra-new-operator": {
									ResourceID: mockMiResourceId,
								},
							},
							UpgradeableTo: pointerutils.ToPtr(v20240812preview.UpgradeableTo(getMIWIUpgradeableToVersion().String())),
						},
					},
				}
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateUpdating, api.ProvisioningStateUpdating))
				doc := getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				// Set UpgradeableTo with new identity
				doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo = pointerutils.ToPtr(api.UpgradeableTo(getMIWIUpgradeableToVersion().String()))
				doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["extra-new-operator"] = api.PlatformWorkloadIdentity{
					ResourceID: mockMiResourceId,
				}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				response := getExistingWorkloadIdentityOpenShiftClusterResponse()
				// Set UpgradeableTo with new identity
				response.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo = pointerutils.ToPtr(v20240812preview.UpgradeableTo(getMIWIUpgradeableToVersion().String()))
				response.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["extra-new-operator"] = v20240812preview.PlatformWorkloadIdentity{
					ResourceID: mockMiResourceId,
				}
				// ProvisioningState is set to Updating
				response.Properties.ProvisioningState = v20240812preview.ProvisioningStateUpdating
				return response
			},
		},
		{
			name: "patch a workload identity cluster from succeeded - can replace platform workload identities and existing clientIDs+objectIDs are removed",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := &v20240812preview.OpenShiftCluster{
					Properties: v20240812preview.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &v20240812preview.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
								"aro-operator": {
									ResourceID: mockMiResourceId,
								},
							},
						},
					},
				}
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateUpdating, api.ProvisioningStateUpdating))
				doc := getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["aro-operator"] = api.PlatformWorkloadIdentity{
					ResourceID: mockMiResourceId,
				}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				response := getExistingWorkloadIdentityOpenShiftClusterResponse()
				response.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["aro-operator"] = v20240812preview.PlatformWorkloadIdentity{
					ResourceID: mockMiResourceId,
				}
				// ProvisioningState is set to Updating
				response.Properties.ProvisioningState = v20240812preview.ProvisioningStateUpdating
				return response
			},
		},
		{
			name: "Fail - patch a workload identity cluster from succeeded - pass same issuerURL in the body",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := &v20240812preview.OpenShiftCluster{
					Properties: v20240812preview.OpenShiftClusterProperties{
						ClusterProfile: v20240812preview.ClusterProfile{
							OIDCIssuer: (*v20240812preview.OIDCIssuer)(pointerutils.ToPtr(mockGuid)),
						},
					},
				}
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				return nil
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: PropertyChangeNotAllowed: properties.clusterProfile.oidcIssuer: Changing property 'properties.clusterProfile.oidcIssuer' is not allowed.",
		},
		{
			name: "patch a workload identity cluster from succeeded - unexpected identity provided",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := &v20240812preview.OpenShiftCluster{
					Properties: v20240812preview.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &v20240812preview.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
								"unexpected-operator": {
									ResourceID: mockMiResourceId,
								},
							},
							UpgradeableTo: pointerutils.ToPtr(v20240812preview.UpgradeableTo(getMIWIUpgradeableToVersion().String())),
						},
					},
				}
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantSystemDataEnriched: true,
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				return nil
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      fmt.Sprintf(`400: PlatformWorkloadIdentityMismatch: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s or %s'. The required platform workload identities are '[aro-operator cloud-controller-manager cloud-network-config disk-csi-driver extra-new-operator file-csi-driver image-registry ingress machine-api]'`, defaultMinorVersion, getMIWIUpgradeableToVersion().MinorVersion()),
		},
		{
			name: "patch a workload identity cluster from succeeded - unexpected identity provided",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := &v20240812preview.OpenShiftCluster{
					Properties: v20240812preview.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &v20240812preview.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]v20240812preview.PlatformWorkloadIdentity{
								"unexpected-operator": {
									ResourceID: mockMiResourceId,
								},
							},
							UpgradeableTo: pointerutils.ToPtr(v20240812preview.UpgradeableTo(getMIWIUpgradeableToVersion().String())),
						},
					},
				}
				return cluster
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantSystemDataEnriched: true,
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				return nil
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      fmt.Sprintf(`400: PlatformWorkloadIdentityMismatch: properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities: There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s or %s'. The required platform workload identities are '[aro-operator cloud-controller-manager cloud-network-config disk-csi-driver extra-new-operator file-csi-driver image-registry ingress machine-api]'`, defaultMinorVersion, getMIWIUpgradeableToVersion().MinorVersion()),
		},
		{
			name: "replace cluster identity",
			request: func() *v20240812preview.OpenShiftCluster {
				cluster := &v20240812preview.OpenShiftCluster{
					Identity: &v20240812preview.ManagedServiceIdentity{
						Type: v20240812preview.ManagedServiceIdentityUserAssigned,
						UserAssignedIdentities: map[string]v20240812preview.UserAssignedIdentity{
							mockMiResourceId2: {
								ClientID:    mockGuid,
								PrincipalID: mockGuid,
							},
						},
					},
				}
				return cluster
			},
			headers: map[string]string{
				middleware.MsiIdentityURLHeader: mockIdentityURL,
				middleware.MsiTenantHeader:      "qwer",
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusOK,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateUpdating, api.ProvisioningStateUpdating))
				doc := getExistingWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				// Expect the Identity to be set to new value
				doc.OpenShiftCluster.Identity = &api.ManagedServiceIdentity{
					TenantID: "qwer",
					Type:     api.ManagedServiceIdentityUserAssigned,
					UserAssignedIdentities: map[string]api.UserAssignedIdentity{
						mockMiResourceId2: {
							ClientID:    mockGuid,
							PrincipalID: mockGuid,
						},
					},
					IdentityURL: mockIdentityURL,
				}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				response := getExistingWorkloadIdentityOpenShiftClusterResponse()
				// Expect the Identity to be updated to new value
				response.Identity = &v20240812preview.ManagedServiceIdentity{
					TenantID: "qwer",
					Type:     v20240812preview.ManagedServiceIdentityUserAssigned,
					UserAssignedIdentities: map[string]v20240812preview.UserAssignedIdentity{
						mockMiResourceId2: {
							ClientID:    mockGuid,
							PrincipalID: mockGuid,
						},
					},
				}
				// ProvisioningState is set to Updating
				response.Properties.ProvisioningState = v20240812preview.ProvisioningStateUpdating
				return response
			},
		},
		{
			name: "update a cluster from a failed during update",
			request: func() *v20240812preview.OpenShiftCluster {
				return &v20240812preview.OpenShiftCluster{}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateFailed, "", api.ProvisioningStateUpdating))
			},
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusOK,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateUpdating, api.ProvisioningStateUpdating))
				doc := getExistingServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateUpdating, "", api.ProvisioningStateUpdating)
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				doc.OpenShiftCluster.Properties.LastProvisioningState = api.ProvisioningStateFailed
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				response := getExistingServicePrincipalOpenShiftClusterResponse()
				// ProvisioningState is set to Updating
				response.Properties.ProvisioningState = v20240812preview.ProvisioningStateUpdating
				return response
			},
		},
		{
			name: "update a cluster from failed during creation",
			request: func() *v20240812preview.OpenShiftCluster {
				return &v20240812preview.OpenShiftCluster{}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateFailed, "", api.ProvisioningStateCreating))
			},
			wantStatusCode: http.StatusBadRequest,
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				return nil
			},
			wantError: "400: RequestNotAllowed: : Request is not allowed on cluster whose creation failed. Delete the cluster.",
		},
		{
			name: "update a cluster from failed during deletion",
			request: func() *v20240812preview.OpenShiftCluster {
				return &v20240812preview.OpenShiftCluster{}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getExistingServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateFailed, "", api.ProvisioningStateDeleting))
			},
			wantStatusCode: http.StatusBadRequest,
			wantResponse: func() *v20240812preview.OpenShiftCluster {
				return nil
			},
			wantError: "400: RequestNotAllowed: : Request is not allowed on cluster whose deletion failed. Delete the cluster.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).
				WithOpenShiftClusters().
				WithSubscriptions().
				WithAsyncOperations().
				WithOpenShiftVersions()
			defer ti.done()

			controller := gomock.NewController(t)
			defer controller.Finish()

			mockQuotaValidator := mock_frontend.NewMockQuotaValidator(controller)
			mockQuotaValidator.EXPECT().ValidateQuota(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.quotaValidatorError).AnyTimes()

			mockSkuValidator := mock_frontend.NewMockSkuValidator(controller)
			mockSkuValidator.EXPECT().ValidateVMSku(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.skuValidatorError).AnyTimes()

			mockProvidersValidator := mock_frontend.NewMockProvidersValidator(controller)
			mockProvidersValidator.EXPECT().ValidateProviders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.providersValidatorError).AnyTimes()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, ti.enricher)
			if err != nil {
				t.Fatal(err)
			}
			f.bucketAllocator = bucket.Fixed(1)
			f.now = func() time.Time { return mockCurrentTime }

			f.quotaValidator = mockQuotaValidator
			f.skuValidator = mockSkuValidator
			f.providersValidator = mockProvidersValidator

			var systemDataClusterDocEnricherCalled bool
			f.systemDataClusterDocEnricher = func(doc *api.OpenShiftClusterDocument, systemData *api.SystemData) {
				systemDataClusterDocEnricherCalled = true
			}

			go f.Run(ctx, nil, nil)
			f.ocpVersionsMu.Lock()
			f.enabledOcpVersions = getOCPVersionsChangeFeed()
			for key, doc := range getOCPVersionsChangeFeed() {
				if doc.Properties.Default {
					f.defaultOcpVersion = key
				}
			}
			f.ocpVersionsMu.Unlock()

			f.platformWorkloadIdentityRoleSetsMu.Lock()
			f.availablePlatformWorkloadIdentityRoleSets = getPlatformWorkloadIdentityRolesChangeFeed()
			f.platformWorkloadIdentityRoleSetsMu.Unlock()

			oc := tt.request()
			requestHeaders := http.Header{
				"Content-Type": []string{"application/json"},
			}
			for k, v := range tt.headers {
				requestHeaders[k] = []string{v}
			}

			var internal api.OpenShiftCluster
			f.apis[defaultAPIVersion].OpenShiftClusterConverter.ToInternal(oc, &internal)
			if internal.UsesWorkloadIdentity() {
				requestHeaders.Add(middleware.MsiIdentityURLHeader, middleware.MockIdentityURL)
				requestHeaders.Add(middleware.MsiTenantHeader, mockGuid)
			}

			resp, b, err := ti.request(http.MethodPatch,
				"https://server"+testdatabase.GetResourcePath(mockGuid, "resourceName")+"?api-version=2024-08-12-preview",
				requestHeaders,
				oc,
			)
			if err != nil {
				t.Error(err, b)
			}

			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("https://localhost:8443/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockGuid, ti.env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse())
			if err != nil {
				t.Error(err)
			}

			if tt.wantDocuments != nil {
				tt.wantDocuments(ti.checker)
				errs := ti.checker.CheckOpenShiftClusters(ti.openShiftClustersClient)
				errs = append(errs, ti.checker.CheckAsyncOperations(ti.asyncOperationsClient)...)
				for _, err := range errs {
					t.Error(err)
				}
			}

			if tt.wantSystemDataEnriched != systemDataClusterDocEnricherCalled {
				t.Error(systemDataClusterDocEnricherCalled)
			}
		})
	}
}

func TestPutorPatchOpenShiftClusterAdminAPI(t *testing.T) {
	ctx := context.Background()
	getAdminPlatformWorkloadIdentityProfile := func() map[string]admin.PlatformWorkloadIdentity {
		return map[string]admin.PlatformWorkloadIdentity{
			"file-csi-driver":          {ResourceID: mockMiResourceId + "0"},
			"cloud-controller-manager": {ResourceID: mockMiResourceId + "1"},
			"ingress":                  {ResourceID: mockMiResourceId + "2"},
			"image-registry":           {ResourceID: mockMiResourceId + "3"},
			"machine-api":              {ResourceID: mockMiResourceId + "4"},
			"cloud-network-config":     {ResourceID: mockMiResourceId + "5"},
			"aro-operator":             {ResourceID: mockMiResourceId + "6"},
			"disk-csi-driver":          {ResourceID: mockMiResourceId + "7"},
		}
	}
	getAdminOpenshiftClusterResponse := func() *admin.OpenShiftCluster {
		return &admin.OpenShiftCluster{
			ID:       mockResourceID,
			Name:     mockResourceName,
			Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
			Location: mockLocation,
			Tags:     map[string]string{"tag": "will-be-kept"},
			Properties: admin.OpenShiftClusterProperties{
				ArchitectureVersion:   admin.ArchitectureVersionV2,
				ProvisioningState:     admin.ProvisioningStateAdminUpdating,
				LastProvisioningState: admin.ProvisioningStateSucceeded,
				MaintenanceTask:       admin.MaintenanceTaskEverything,
				MaintenanceState:      admin.MaintenanceStateUnplanned,
				OperatorFlags:         admin.OperatorFlags{"testFlag": "true"},
				CreatedBy:             "unknown",
				ProvisionedBy:         "unknown",
				ClusterProfile: admin.ClusterProfile{
					Version:              defaultVersion,
					Domain:               mockDomain,
					ResourceGroupID:      mockResourceGroupID,
					FipsValidatedModules: admin.FipsValidatedModulesDisabled,
				},
				NetworkProfile: admin.NetworkProfile{
					PodCIDR:      mockPodCIDR,
					ServiceCIDR:  mockServiceCIDR,
					OutboundType: admin.OutboundTypeLoadbalancer,
					LoadBalancerProfile: &admin.LoadBalancerProfile{
						ManagedOutboundIPs: &admin.ManagedOutboundIPs{
							Count: 1,
						},
					},
					PreconfiguredNSG: admin.PreconfiguredNSGDisabled,
				},
				MasterProfile: admin.MasterProfile{
					EncryptionAtHost: admin.EncryptionAtHostDisabled,
					VMSize:           vms.VMSize(mockVMSize),
					SubnetID:         mockMasterSubnetID,
				},
				WorkerProfiles: []admin.WorkerProfile{
					{
						Name:             "worker",
						EncryptionAtHost: admin.EncryptionAtHostDisabled,
						VMSize:           vms.VMSize(mockVMSize),
						DiskSizeGB:       128,
						Count:            3,
						SubnetID:         mockWorkerSubnetID,
					},
				},
				APIServerProfile: admin.APIServerProfile{
					Visibility: admin.VisibilityPrivate,
				},
				IngressProfiles: []admin.IngressProfile{
					{
						Name:       "default",
						Visibility: admin.VisibilityPublic,
					},
				},
				FeatureProfile: admin.FeatureProfile{
					GatewayEnabled: true,
				},
			},
		}
	}
	getAdminServicePrincipalOpenshiftClusterResponse := func() *admin.OpenShiftCluster {
		response := getAdminOpenshiftClusterResponse()
		response.Properties.ServicePrincipalProfile = &admin.ServicePrincipalProfile{
			ClientID: mockGuid,
		}
		return response
	}
	getAdminWorkloadIdentityOpenshiftClusterResponse := func() *admin.OpenShiftCluster {
		response := getAdminOpenshiftClusterResponse()
		response.Identity = &admin.ManagedServiceIdentity{
			Type: "UserAssigned",
			UserAssignedIdentities: map[string]admin.UserAssignedIdentity{
				mockMiResourceId: {},
			},
		}
		response.Properties.PlatformWorkloadIdentityProfile = &admin.PlatformWorkloadIdentityProfile{
			PlatformWorkloadIdentities: getAdminPlatformWorkloadIdentityProfile(),
		}
		return response
	}
	getAdminServicePrincipalOpenShiftClusterDocument := func(provisioningState, lastProvisioningState, failedProvisioningState api.ProvisioningState) *api.OpenShiftClusterDocument {
		doc := getServicePrincipalOpenShiftClusterDocument(provisioningState, lastProvisioningState, failedProvisioningState)
		doc.OpenShiftCluster.Tags = map[string]string{"tag": "will-be-kept"}
		doc.OpenShiftCluster.Properties.OperatorFlags = api.OperatorFlags{"testFlag": "true"}
		doc.OpenShiftCluster.Properties.CreatedAt = time.Time{}
		// FakeClusterConflictChecker will not allow the update to proceed if the ClientIDKey & ClusterResourceGroupIDKey is set.
		doc.ClientIDKey = ""
		doc.ClusterResourceGroupIDKey = ""
		return doc
	}
	getAdminWorkloadIdentityOpenShiftClusterDocument := func(provisioningState, lastProvisioningState, failedProvisioningState api.ProvisioningState) *api.OpenShiftClusterDocument {
		doc := getWorkloadIdentityOpenShiftClusterDocument(provisioningState, lastProvisioningState, failedProvisioningState)
		doc.OpenShiftCluster.Tags = map[string]string{"tag": "will-be-kept"}
		doc.OpenShiftCluster.Properties.OperatorFlags = api.OperatorFlags{"testFlag": "true"}
		doc.OpenShiftCluster.Properties.CreatedAt = time.Time{}
		// FakeClusterConflictChecker will not allow the update to proceed if the ClientIDKey & ClusterResourceGroupIDKey is set.
		doc.ClientIDKey = ""
		doc.ClusterResourceGroupIDKey = ""
		return doc
	}

	for _, tt := range []struct {
		name                    string
		request                 func() *admin.OpenShiftCluster
		fixture                 func(*testdatabase.Fixture)
		quotaValidatorError     error
		skuValidatorError       error
		providersValidatorError error
		wantSystemDataEnriched  bool
		wantDocuments           func(*testdatabase.Checker)
		wantStatusCode          int
		wantResponse            func() *admin.OpenShiftCluster
		wantAsync               bool
		wantError               string
	}{
		{
			name: "patch with empty request",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateAdminUpdating))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateUnplanned
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				return getAdminServicePrincipalOpenshiftClusterResponse()
			},
		},
		{
			name: "patch with flags merges the flags together",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						MaintenanceTask:     admin.MaintenanceTaskOperator,
						OperatorFlags:       admin.OperatorFlags{"exploding-flag": "true", "overwrittenFlag": "true"},
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateAdminUpdating))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.OperatorFlags = api.OperatorFlags{"exploding-flag": "true", "overwrittenFlag": "true", "testFlag": "true"}
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskOperator
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateUnplanned
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.OperatorFlags = admin.OperatorFlags{"exploding-flag": "true", "overwrittenFlag": "true", "testFlag": "true"}
				response.Properties.MaintenanceTask = admin.MaintenanceTaskOperator
				return response
			},
		},
		{
			name: "patch an existing cluster with no flags in db will use defaults",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", "")
				doc.OpenShiftCluster.Properties.OperatorFlags = operator.DefaultOperatorFlags()
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateAdminUpdating))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.OperatorFlags = operator.DefaultOperatorFlags()
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateUnplanned
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.OperatorFlags = operator.DefaultOperatorFlags()
				return response
			},
		},
		{
			name: "patch with OperatorFlagsMergeStrategy=reset will reset flags to defaults and merge in request flags",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					OperatorFlagsMergeStrategy: admin.OperatorFlagsMergeStrategyReset,
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						OperatorFlags:       admin.OperatorFlags{"exploding-flag": "true", "overwrittenFlag": "true"},
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", "")
				doc.OpenShiftCluster.Properties.OperatorFlags = api.OperatorFlags{"testFlag": "true", "overwrittenFlag": "false"}
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateAdminUpdating))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateSucceeded, "")
				expectedFlags := operator.DefaultOperatorFlags()
				expectedFlags["exploding-flag"] = "true"
				expectedFlags["overwrittenFlag"] = "true"
				doc.OpenShiftCluster.Properties.OperatorFlags = expectedFlags
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateUnplanned
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				expectedFlags := operator.DefaultOperatorFlags()
				expectedFlags["exploding-flag"] = "true"
				expectedFlags["overwrittenFlag"] = "true"
				response.Properties.OperatorFlags = expectedFlags
				return response
			},
		},
		{
			name: "patch with operator update request -- existing maintenance task",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						MaintenanceTask:     admin.MaintenanceTaskOperator,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateNone
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateAdminUpdating))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskOperator
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateUnplanned
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.MaintenanceTask = admin.MaintenanceTaskOperator
				return response
			},
		},
		{
			name: "patch a cluster with registry profile should fail",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						RegistryProfiles: []admin.RegistryProfile{
							{
								Name:     "TestUser",
								Username: "TestUserName",
							},
						},
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateNone
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: false,
			wantDocuments: func(checker *testdatabase.Checker) {
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateNone
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      false,
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: PropertyChangeNotAllowed: properties.registryProfiles: Changing property 'properties.registryProfiles' is not allowed.`,
			wantResponse:   func() *admin.OpenShiftCluster { return nil },
		},
		{
			name: "patch an empty maintenance state cluster with maintenance pending request",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						MaintenanceTask:     admin.MaintenanceTaskPending,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.MaintenanceState = ""
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", "")
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStatePending
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.ProvisioningState = admin.ProvisioningStateSucceeded
				response.Properties.LastProvisioningState = ""
				response.Properties.MaintenanceTask = ""
				response.Properties.MaintenanceState = admin.MaintenanceStatePending
				return response
			},
		},
		{
			name: "patch a none maintenance state cluster with maintenance pending request",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						MaintenanceTask:     admin.MaintenanceTaskPending,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateNone
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", "")
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStatePending
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.ProvisioningState = admin.ProvisioningStateSucceeded
				response.Properties.LastProvisioningState = ""
				response.Properties.MaintenanceTask = ""
				response.Properties.MaintenanceState = admin.MaintenanceStatePending
				return response
			},
		},
		{
			name: "patch a maintenance state pending cluster with planned maintenance",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						MaintenanceTask:     admin.MaintenanceTaskEverything,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStatePending
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateAdminUpdating))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStatePlanned
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.MaintenanceState = admin.MaintenanceStatePlanned
				return response
			},
		},
		{
			name: "patch a planned maintenance ongoing cluster with maintenance none request",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						MaintenanceTask:     admin.MaintenanceTaskNone,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStatePlanned
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateNone
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.ProvisioningState = admin.ProvisioningStateSucceeded
				response.Properties.MaintenanceState = admin.MaintenanceStateNone
				response.Properties.MaintenanceTask = ""
				return response
			},
		},
		{
			name: "patch an unplanned maintenance ongoing cluster with maintenance none request",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						MaintenanceTask:     admin.MaintenanceTaskNone,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateUnplanned
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateNone
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.ProvisioningState = admin.ProvisioningStateSucceeded
				response.Properties.MaintenanceState = admin.MaintenanceStateNone
				response.Properties.MaintenanceTask = ""
				return response
			},
		},
		{
			name: "patch a none maintenance state cluster with maintenance unplanned request",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateNone
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateAdminUpdating))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateUnplanned
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.MaintenanceState = admin.MaintenanceStateUnplanned
				return response
			},
		},
		{
			name: "patch a failed planned maintenance cluster with customer action needed",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						MaintenanceTask:     admin.MaintenanceTaskCustomerActionNeeded,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStatePlanned
				doc.OpenShiftCluster.Properties.LastAdminUpdateError = "error"
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateCustomerActionNeeded
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				doc.OpenShiftCluster.Properties.LastAdminUpdateError = "error"
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.ProvisioningState = admin.ProvisioningStateSucceeded
				response.Properties.LastAdminUpdateError = "error"
				response.Properties.MaintenanceTask = ""
				response.Properties.MaintenanceState = admin.MaintenanceStateCustomerActionNeeded
				return response
			},
		},
		{
			name: "patch a failed unplanned maintenance cluster with customer action needed",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						MaintenanceTask:     admin.MaintenanceTaskCustomerActionNeeded,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", api.ProvisioningStateUpdating)
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateUnplanned
				doc.OpenShiftCluster.Properties.LastAdminUpdateError = "error"
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", api.ProvisioningStateUpdating)
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateCustomerActionNeeded
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				doc.OpenShiftCluster.Properties.LastAdminUpdateError = "error"
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.ProvisioningState = admin.ProvisioningStateSucceeded
				response.Properties.LastProvisioningState = ""
				response.Properties.FailedProvisioningState = admin.ProvisioningStateUpdating
				response.Properties.LastAdminUpdateError = "error"
				response.Properties.MaintenanceTask = ""
				response.Properties.MaintenanceState = admin.MaintenanceStateCustomerActionNeeded
				return response
			},
		},
		{
			name: "patch a customer action needed cluster with maintenance state none",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						MaintenanceTask:     admin.MaintenanceTaskNone,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", api.ProvisioningStateUpdating)
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateCustomerActionNeeded
				doc.OpenShiftCluster.Properties.LastAdminUpdateError = "error"
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateSucceeded, api.ProvisioningStateSucceeded))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", api.ProvisioningStateUpdating)
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateNone
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				doc.OpenShiftCluster.Properties.LastAdminUpdateError = "error"
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.ProvisioningState = admin.ProvisioningStateSucceeded
				response.Properties.LastProvisioningState = ""
				response.Properties.FailedProvisioningState = admin.ProvisioningStateUpdating
				response.Properties.LastAdminUpdateError = "error"
				response.Properties.MaintenanceTask = ""
				response.Properties.MaintenanceState = admin.MaintenanceStateNone
				return response
			},
		},
		{
			name: "patch a customer action needed cluster with maintenance state unplanned",
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
						MaintenanceTask:     admin.MaintenanceTaskEverything,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", api.ProvisioningStateUpdating)
				doc.OpenShiftCluster.Properties.MaintenanceTask = ""
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateCustomerActionNeeded
				doc.OpenShiftCluster.Properties.LastAdminUpdateError = "error"
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateAdminUpdating))
				doc := getAdminServicePrincipalOpenShiftClusterDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateSucceeded, api.ProvisioningStateUpdating)
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateUnplanned
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminServicePrincipalOpenshiftClusterResponse()
				response.Properties.FailedProvisioningState = admin.ProvisioningStateUpdating
				response.Properties.MaintenanceState = admin.MaintenanceStateUnplanned
				return response
			},
		},
		{
			name: "patch workload identity cluster with empty request - WI-related fields are still present in the cluster doc afterward",
			// Several workload identity-related fields are removed from the cluster doc by the converter's
			// ExternalNoReadOnly. Since the `wantDocuments` and `wantResponse` fields in these unit test cases
			// only reflect what the frontend does before the backend acts on the async operation, these fields,
			// which were included in the fixture, are notably missing in `wantDocuments` and `wantResponse`. In an
			// end-to-end admin PATCH, the fields would be repopulated by designated steps in the admin update.
			request: func() *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					Properties: admin.OpenShiftClusterProperties{
						ArchitectureVersion: admin.ArchitectureVersionV2,
					},
				}
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(mockSubscriptionDocument)
				f.AddOpenShiftClusterDocuments(getAdminWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateSucceeded, "", ""))
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(checker *testdatabase.Checker) {
				checker.AddAsyncOperationDocuments(getAsynchronousOperationDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateAdminUpdating))
				doc := getAdminWorkloadIdentityOpenShiftClusterDocument(api.ProvisioningStateAdminUpdating, api.ProvisioningStateSucceeded, "")
				doc.OpenShiftCluster.Identity.TenantID = ""
				doc.OpenShiftCluster.Properties.MaintenanceTask = api.MaintenanceTaskEverything
				doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateUnplanned
				doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []api.EffectiveOutboundIP{}
				checker.AddOpenShiftClusterDocuments(doc)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *admin.OpenShiftCluster {
				response := getAdminWorkloadIdentityOpenshiftClusterResponse()
				response.Properties.MaintenanceState = admin.MaintenanceStateUnplanned
				return response
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).
				WithOpenShiftClusters().
				WithSubscriptions().
				WithAsyncOperations()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, ti.enricher)
			if err != nil {
				t.Fatal(err)
			}
			f.bucketAllocator = bucket.Fixed(1)
			f.now = func() time.Time { return mockCurrentTime }

			var systemDataClusterDocEnricherCalled bool
			f.systemDataClusterDocEnricher = func(doc *api.OpenShiftClusterDocument, systemData *api.SystemData) {
				systemDataClusterDocEnricherCalled = true
			}

			go f.Run(ctx, nil, nil)

			f.platformWorkloadIdentityRoleSetsMu.Lock()
			f.availablePlatformWorkloadIdentityRoleSets = getPlatformWorkloadIdentityRolesChangeFeed()
			f.platformWorkloadIdentityRoleSetsMu.Unlock()

			oc := tt.request()
			requestHeaders := http.Header{
				"Content-Type": []string{"application/json"},
			}

			var internal api.OpenShiftCluster
			f.apis["admin"].OpenShiftClusterConverter.ToInternal(oc, &internal)

			resp, b, err := ti.request(http.MethodPatch,
				"https://server"+testdatabase.GetResourcePath(mockGuid, "resourceName")+"?api-version=admin",
				requestHeaders,
				oc,
			)
			if err != nil {
				t.Error(err, b)
			}

			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("https://localhost:8443/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockGuid, ti.env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse())
			if err != nil {
				t.Error(err)
			}

			if tt.wantDocuments != nil {
				tt.wantDocuments(ti.checker)
				errs := ti.checker.CheckOpenShiftClusters(ti.openShiftClustersClient)
				errs = append(errs, ti.checker.CheckAsyncOperations(ti.asyncOperationsClient)...)
				for _, err := range errs {
					t.Error(err)
				}
			}

			if tt.wantSystemDataEnriched != systemDataClusterDocEnricherCalled {
				t.Error(systemDataClusterDocEnricherCalled)
			}
		})
	}
}

func TestEnrichClusterSystemData(t *testing.T) {
	accountID1 := "00000000-0000-0000-0000-000000000001"
	accountID2 := "00000000-0000-0000-0000-000000000002"
	timestampString := "2021-01-23T12:34:54.0000000Z"
	timestamp, err := time.Parse(time.RFC3339, timestampString)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name       string
		systemData *api.SystemData
		expected   *api.OpenShiftClusterDocument
	}{
		{
			name:       "new systemData is nil",
			systemData: nil,
			expected: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{},
			},
		},
		{
			name: "new systemData has all fields",
			systemData: &api.SystemData{
				CreatedBy:          accountID1,
				CreatedByType:      api.CreatedByTypeApplication,
				CreatedAt:          &timestamp,
				LastModifiedBy:     accountID1,
				LastModifiedByType: api.CreatedByTypeApplication,
				LastModifiedAt:     &timestamp,
			},
			expected: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					SystemData: api.SystemData{
						CreatedBy:          accountID1,
						CreatedByType:      api.CreatedByTypeApplication,
						CreatedAt:          &timestamp,
						LastModifiedBy:     accountID1,
						LastModifiedByType: api.CreatedByTypeApplication,
						LastModifiedAt:     &timestamp,
					},
				},
			},
		},
		{
			name: "update object",
			systemData: &api.SystemData{
				CreatedBy:          accountID1,
				CreatedByType:      api.CreatedByTypeApplication,
				CreatedAt:          &timestamp,
				LastModifiedBy:     accountID2,
				LastModifiedByType: api.CreatedByTypeApplication,
				LastModifiedAt:     &timestamp,
			},
			expected: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					SystemData: api.SystemData{
						CreatedBy:          accountID1,
						CreatedByType:      api.CreatedByTypeApplication,
						CreatedAt:          &timestamp,
						LastModifiedBy:     accountID2,
						LastModifiedByType: api.CreatedByTypeApplication,
						LastModifiedAt:     &timestamp,
					},
				},
			},
		},
		{
			name: "old cluster update. Creation unknown",
			systemData: &api.SystemData{
				LastModifiedBy:     accountID2,
				LastModifiedByType: api.CreatedByTypeApplication,
				LastModifiedAt:     &timestamp,
			},
			expected: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					SystemData: api.SystemData{
						LastModifiedBy:     accountID2,
						LastModifiedByType: api.CreatedByTypeApplication,
						LastModifiedAt:     &timestamp,
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc := &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{},
			}
			enrichClusterSystemData(doc, tt.systemData)

			if !reflect.DeepEqual(doc, tt.expected) {
				t.Error(cmp.Diff(doc, tt.expected))
			}
		})
	}
}

func TestValidateIdentityUrl(t *testing.T) {
	for _, tt := range []struct {
		name        string
		identityURL string
		cluster     *api.OpenShiftCluster
		expected    *api.OpenShiftCluster
		wantError   error
	}{
		{
			name:        "identity URL is empty",
			identityURL: "",
			cluster:     &api.OpenShiftCluster{},
			expected:    &api.OpenShiftCluster{},
			wantError:   errMissingIdentityParameter,
		},
		{
			name: "pass - identity URL passed",
			cluster: &api.OpenShiftCluster{
				Identity: &api.ManagedServiceIdentity{},
			},
			identityURL: "http://foo.bar",
			expected: &api.OpenShiftCluster{
				Identity: &api.ManagedServiceIdentity{
					IdentityURL: "http://foo.bar",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentityUrl(tt.cluster, tt.identityURL)
			if !errors.Is(err, tt.wantError) {
				t.Error(cmp.Diff(err, tt.wantError))
			}

			if !reflect.DeepEqual(tt.cluster, tt.expected) {
				t.Error(cmp.Diff(tt.cluster, tt.expected))
			}
		})
	}
}

func TestValidateIdentityTenantID(t *testing.T) {
	for _, tt := range []struct {
		name      string
		tenantID  string
		cluster   *api.OpenShiftCluster
		expected  *api.OpenShiftCluster
		wantError error
	}{
		{
			name:      "tenantID is empty",
			tenantID:  "",
			cluster:   &api.OpenShiftCluster{},
			expected:  &api.OpenShiftCluster{},
			wantError: errMissingIdentityParameter,
		},
		{
			name: "pass - tenantID passed",
			cluster: &api.OpenShiftCluster{
				Identity: &api.ManagedServiceIdentity{},
			},
			tenantID: "bogus",
			expected: &api.OpenShiftCluster{
				Identity: &api.ManagedServiceIdentity{
					TenantID: "bogus",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentityTenantID(tt.cluster, tt.tenantID)
			if !errors.Is(err, tt.wantError) {
				t.Error(cmp.Diff(err, tt.wantError))
			}

			if !reflect.DeepEqual(tt.cluster, tt.expected) {
				t.Error(cmp.Diff(tt.cluster, tt.expected))
			}
		})
	}
}

// TestConversion_PreconfiguredNSG checks that converting from the internal
// api.OpenShiftCluster to the external admin.OpenShiftCluster preserves or
// defaults the PreconfiguredNSG field correctly.
func TestConversion_PreconfiguredNSG(t *testing.T) {
	// Grab the existing converter from the "admin" API version.
	// This is what actually does the internal->external conversion.
	converter := api.APIs["admin"].OpenShiftClusterConverter

	tests := []struct {
		name     string
		input    *api.OpenShiftCluster
		expected admin.PreconfiguredNSG
	}{
		{
			name: "preconfiguredNSG is correctly preserved as Enabled",
			input: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					NetworkProfile: api.NetworkProfile{
						PreconfiguredNSG: api.PreconfiguredNSGEnabled,
					},
				},
			},
			expected: admin.PreconfiguredNSGEnabled,
		},
		{
			name: "preconfiguredNSG defaults to Disabled when missing",
			input: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{},
			},
			expected: admin.PreconfiguredNSGDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			externalObj := converter.ToExternal(tt.input)
			oc, ok := externalObj.(*admin.OpenShiftCluster)
			if !ok {
				t.Fatalf("expected *admin.OpenShiftCluster, got %T", externalObj)
			}
			if oc.Properties.NetworkProfile.PreconfiguredNSG != tt.expected {
				t.Errorf("expected %v, got %v",
					tt.expected, oc.Properties.NetworkProfile.PreconfiguredNSG)
			}
		})
	}
}

// TestRegression_PreconfiguredNSGField ensures we don't regress on preserving
// the PreconfiguredNSG field in future changes to conversion logic.
func TestRegression_PreconfiguredNSGField(t *testing.T) {
	converter := api.APIs["admin"].OpenShiftClusterConverter

	input := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			NetworkProfile: api.NetworkProfile{
				PreconfiguredNSG: api.PreconfiguredNSGEnabled,
			},
		},
	}

	externalObj := converter.ToExternal(input)
	oc, ok := externalObj.(*admin.OpenShiftCluster)
	if !ok {
		t.Fatalf("expected *admin.OpenShiftCluster, got %T", externalObj)
	}

	if oc.Properties.NetworkProfile.PreconfiguredNSG != admin.PreconfiguredNSGEnabled {
		t.Errorf("expected PreconfiguredNSG=Enabled, got %v",
			oc.Properties.NetworkProfile.PreconfiguredNSG)
	}
}
