package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestPreflightValidation(t *testing.T) {
	ctx := context.Background()
	mockSubID := "00000000-0000-0000-0000-000000000000"
	apiVersion := "2020-04-30"
	clusterId := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"
	location := "eastus"
	defaultProfile := "default"
	resourceGroup := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resourceGroupTest"
	netProfile := "10.128.0.0/14"
	encryptionSet := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/ms-eastus/providers/Microsoft.Compute/diskEncryptionSets/ms-eastus-disk-encryption-set"
	masterSub := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master"
	workerSub := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker"

	preflightPayload := []byte(fmt.Sprintf(`
	{
		"apiVersion": "%s",
		"id": "%s",
		"name": "%s",
		"type": "%s",
		"location": "%s",
		"properties": {
			"clusterProfile": {
				"domain": "%s",
				"resourceGroupId": "%s",
				"fipsValidatedModules": "%s",
				"version": "%s"
			},
			"consoleProfile": {},
			"servicePrincipalProfile": {
				"clientId": "%s",
				"clientSecret": "%s"
			},
			"networkProfile": {
				"podCidr": "%s",
				"serviceCidr": "%s"
			},
			"masterProfile": {
				"vmSize": "%s",
				"subnetId": "%s",
				"encryptionAtHost": "%s",
				"diskEncryptionSetId": "%s"
			},
			"workerProfiles": [
			{
				"name": "%s",
				"vmSize": "%s",
				"diskSizeGB": %v,
				"encryptionAtHost": "%s",
				"subnetId": "%s",
				"count": %v,
				"diskEncryptionSetId": "%s"
			}
			],
			"apiserverProfile": {
				"visibility": "%s"
			},
			"ingressProfiles": [
			{
				"name": "%s",
				"visibility": "%s",
				"IP": "%s"
			}
			]
		}
	}
	`, apiVersion, clusterId, api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Name,
		api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Type,
		location, defaultProfile, resourceGroup, api.EncryptionAtHostEnabled, version.DefaultInstallStream.Version.String(),
		mockSubID, mockSubID, netProfile, netProfile, api.VMSizeStandardD32sV3, masterSub,
		api.EncryptionAtHostEnabled, encryptionSet,
		api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.WorkerProfiles[0].Name, api.VMSizeStandardD32sV3,
		api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.WorkerProfiles[0].DiskSizeGB,
		api.EncryptionAtHostEnabled, workerSub,
		api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.WorkerProfiles[0].Count,
		encryptionSet, api.VisibilityPublic, defaultProfile, api.VisibilityPublic,
		api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.IngressProfiles[0].IP))

	defaultVersionChangeFeed := map[string]*api.OpenShiftVersion{
		version.DefaultInstallStream.Version.String(): {
			Properties: api.OpenShiftVersionProperties{
				Version: version.DefaultInstallStream.Version.String(),
				Enabled: true,
			},
		},
	}

	type test struct {
		name             string
		preflightRequest func() *api.PreflightRequest
		fixture          func(*testdatabase.Fixture)
		changeFeed       map[string]*api.OpenShiftVersion
		wantStatusCode   int
		wantError        string
		wantResponse     *api.ValidationResult
	}
	for _, tt := range []*test{
		{
			name: "Successful Preflight Create",
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(api.ExampleSubscriptionDocument())
			},
			changeFeed: defaultVersionChangeFeed,
			preflightRequest: func() *api.PreflightRequest {
				return &api.PreflightRequest{
					Resources: []json.RawMessage{
						preflightPayload,
					},
				}
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.ValidationResult{
				Status: api.ValidationStatusSucceeded,
			},
		},
		{
			name: "Failed Preflight Static Invalid ResourceGroup",
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(api.ExampleSubscriptionDocument())
			},
			preflightRequest: func() *api.PreflightRequest {
				return &api.PreflightRequest{
					Resources: []json.RawMessage{
						[]byte(fmt.Sprintf(`
							{
								"apiVersion": "%s",
								"id": "%s",
								"name": "%s",
								"type": "%s",
								"location": "%s",
								"properties": {
									"clusterProfile": {
										"domain": "%s",
										"resourceGroupId": "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/resourcenameTest",
										"fipsValidatedModules": "%s"
									}
								}
							}
							`, apiVersion, clusterId,
							api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Name,
							api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Type,
							location, defaultProfile, api.FipsValidatedModulesEnabled)),
					},
				}
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.ValidationResult{
				Status: api.ValidationStatusFailed,
				Error: &api.CloudErrorBody{
					Message: "400: InvalidParameter: properties.clusterProfile.resourceGroupId: The provided resource group '/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/resourcenameTest' is invalid: must be in same subscription as cluster.",
				},
			},
		},
		{
			name: "Failed Preflight Cluster Check",
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(api.ExampleSubscriptionDocument())
			},
			preflightRequest: func() *api.PreflightRequest {
				return &api.PreflightRequest{
					Resources: []json.RawMessage{
						[]byte(fmt.Sprintf(`
							{
								"apiVersion": "%s",
								"id": "%s",
								"name": "%s",
								"type": "%s",
								"location": "%s",
								"properties": {}
							}
							`, apiVersion, "resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName",
							api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Name,
							api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Type,
							location)),
					},
				}
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.ValidationResult{
				Status: api.ValidationStatusFailed,
				Error: &api.CloudErrorBody{
					Message: "400: Cluster not found for resourceID: resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
				},
			},
		},
		{
			name: "Failed Preflight Install Version",
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(api.ExampleSubscriptionDocument())
			},
			changeFeed: defaultVersionChangeFeed,
			preflightRequest: func() *api.PreflightRequest {
				return &api.PreflightRequest{
					Resources: []json.RawMessage{
						[]byte(fmt.Sprintf(`
							{
								"apiVersion": "%s",
								"id": "%s",
								"name": "%s",
								"type": "%s",
								"location": "%s",
								"properties": {
									"clusterProfile": {
										"domain": "%s",
										"resourceGroupId": "%s",
										"fipsValidatedModules": "%s",
										"version": "4.11.43"
									},
									"consoleProfile": {},
									"servicePrincipalProfile": {
										"clientId": "%s",
										"clientSecret": "%s"
									},
									"networkProfile": {
										"podCidr": "%s",
										"serviceCidr": "%s"
									},
									"masterProfile": {
										"vmSize": "%s",
										"subnetId": "%s",
										"encryptionAtHost": "%s",
										"diskEncryptionSetId": "%s"
									},
									"workerProfiles": [
									{
										"name": "%s",
										"vmSize": "%s",
										"diskSizeGB": %v,
										"encryptionAtHost": "%s",
										"subnetId": "%s",
										"count": %v,
										"diskEncryptionSetId": "%s"
									}
									],
									"apiserverProfile": {
										"visibility": "%s"
									},
									"ingressProfiles": [
									{
										"name": "%s",
										"visibility": "%s",
										"IP": "%s"
									}
									]
								}
							}
							`, apiVersion, clusterId, api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Name,
							api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Type,
							location, defaultProfile, resourceGroup, api.EncryptionAtHostEnabled,
							mockSubID, mockSubID, netProfile, netProfile, api.VMSizeStandardD32sV3, masterSub,
							api.EncryptionAtHostEnabled, encryptionSet,
							api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.WorkerProfiles[0].Name, api.VMSizeStandardD32sV3,
							api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.WorkerProfiles[0].DiskSizeGB,
							api.EncryptionAtHostEnabled, workerSub,
							api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.WorkerProfiles[0].Count,
							encryptionSet, api.VisibilityPublic, defaultProfile, api.VisibilityPublic,
							api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.IngressProfiles[0].IP)),
					},
				}
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.ValidationResult{
				Status: api.ValidationStatusFailed,
				Error: &api.CloudErrorBody{
					Code:    "InvalidParameter",
					Message: "400: InvalidParameter: properties.clusterProfile.version: The requested OpenShift version '4.11.43' is invalid.",
				},
			},
		},
		{
			name: "Successful Preflight Update",
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(api.ExampleSubscriptionDocument())
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(api.ExampleOpenShiftClusterDocument().ID, api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Name)),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(api.ExampleOpenShiftClusterDocument().ID, api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Name),
						Name:     api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Name,
						Type:     api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Type,
						Tags:     map[string]string{"tag": "will-be-kept"},
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								Domain:               defaultProfile,
								FipsValidatedModules: api.FipsValidatedModulesEnabled,
								ResourceGroupID:      resourceGroup,
								Version:              version.DefaultInstallStream.Version.String(),
							},
							ServicePrincipalProfile: &api.ServicePrincipalProfile{
								ClientID:     mockSubID,
								ClientSecret: api.SecureString(mockSubID),
							},
							NetworkProfile: api.NetworkProfile{
								PodCIDR:     netProfile,
								ServiceCIDR: netProfile,
							},
							MasterProfile: api.MasterProfile{
								VMSize:              api.VMSizeStandardD32sV3,
								SubnetID:            masterSub,
								DiskEncryptionSetID: encryptionSet,
								EncryptionAtHost:    api.EncryptionAtHostEnabled,
							},
							WorkerProfiles: []api.WorkerProfile{
								{
									Name:                api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.WorkerProfiles[0].Name,
									VMSize:              api.VMSizeStandardD32sV3,
									DiskSizeGB:          api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.WorkerProfiles[0].DiskSizeGB,
									EncryptionAtHost:    api.EncryptionAtHostEnabled,
									SubnetID:            workerSub,
									Count:               api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.WorkerProfiles[0].Count,
									DiskEncryptionSetID: encryptionSet,
								},
							},
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							IngressProfiles: api.ExampleOpenShiftClusterDocument().OpenShiftCluster.Properties.IngressProfiles,
						},
					},
				})
			},
			preflightRequest: func() *api.PreflightRequest {
				return &api.PreflightRequest{
					Resources: []json.RawMessage{
						preflightPayload,
					},
				}
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.ValidationResult{
				Status: api.ValidationStatusSucceeded,
			},
		},
		{
			name: "Failed Preflight Update Invalid Domain",
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(api.ExampleSubscriptionDocument())
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(api.ExampleOpenShiftClusterDocument().ID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(api.ExampleOpenShiftClusterDocument().ID, "resourceName"),
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags:     map[string]string{"tag": "will-be-kept"},
						Location: "eastus",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								Domain: "different",
							},
						},
					},
				})
			},
			preflightRequest: func() *api.PreflightRequest {
				return &api.PreflightRequest{
					Resources: []json.RawMessage{
						preflightPayload,
					},
				}
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.ValidationResult{
				Status: api.ValidationStatusFailed,
				Error: &api.CloudErrorBody{
					Message: "400: PropertyChangeNotAllowed: properties.clusterProfile.domain: Changing property 'properties.clusterProfile.domain' is not allowed.",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).
				WithSubscriptions().
				WithOpenShiftClusters()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			oc := tt.preflightRequest()

			go f.Run(ctx, nil, nil)
			f.ocpVersionsMu.Lock()
			f.defaultOcpVersion = "4.13.40"
			f.enabledOcpVersions = map[string]*api.OpenShiftVersion{
				f.defaultOcpVersion: {
					Properties: api.OpenShiftVersionProperties{
						Version: f.defaultOcpVersion,
					},
				},
			}
			f.ocpVersionsMu.Unlock()

			headers := http.Header{
				"Content-Type": []string{"application/json"},
			}

			resp, b, err := ti.request(http.MethodPost,
				"https://server"+testdatabase.GetPreflightPath(api.ExampleOpenShiftClusterDocument().ID, "deploymentName")+"?api-version=2020-04-30",
				headers, oc)
			if err != nil {
				t.Error(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
