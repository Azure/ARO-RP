package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/test/validate"
)

func TestOpenShiftClusterStaticValidateDelta(t *testing.T) {
	var (
		subscriptionID = "af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1"
		id             = fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/microsoft.redhatopenshift/openshiftclusters/resourceName", subscriptionID)
	)

	tests := []struct {
		name    string
		oc      func() *OpenShiftCluster
		modify  func(oc *OpenShiftCluster)
		wantErr string
	}{
		{
			name: "valid",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{ID: id}
			},
		},
		{
			name: "valid id case change",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{ID: id}
			},
			modify: func(oc *OpenShiftCluster) { oc.ID = strings.ToUpper(oc.ID) },
		},
		{
			name: "valid name case change",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{Name: "resourceName"}
			},
			modify: func(oc *OpenShiftCluster) { oc.Name = strings.ToUpper(oc.Name) },
		},
		{
			name: "valid type case change",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{Type: "Microsoft.RedHatOpenShift/openShiftClusters"}
			},
			modify: func(oc *OpenShiftCluster) { oc.Type = strings.ToUpper(oc.Type) },
		},
		{
			name: "location change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{Location: "eastus"}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Location = strings.ToUpper(oc.Location) },
			wantErr: "400: PropertyChangeNotAllowed: location: Changing property 'location' is not allowed.",
		},
		{
			name: "tags change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Tags: map[string]string{
						"key": "value",
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Tags = map[string]string{"new": "value"} },
			wantErr: "400: PropertyChangeNotAllowed: tags: Changing property 'tags' is not allowed.",
		},
		{
			name: "provisioningState change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						ProvisioningState: ProvisioningStateSucceeded,
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ProvisioningState = ProvisioningStateFailed },
			wantErr: "400: PropertyChangeNotAllowed: properties.provisioningState: Changing property 'properties.provisioningState' is not allowed.",
		},
		{
			name: "lastProvisioningState change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.LastProvisioningState = ProvisioningStateSucceeded },
			wantErr: "400: PropertyChangeNotAllowed: properties.lastProvisioningState: Changing property 'properties.lastProvisioningState' is not allowed.",
		},
		{
			name: "failedProvisioningState change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						ProvisioningState:       ProvisioningStateFailed,
						FailedProvisioningState: ProvisioningStateCreating,
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.FailedProvisioningState = ProvisioningStateUpdating },
			wantErr: "400: PropertyChangeNotAllowed: properties.failedProvisioningState: Changing property 'properties.failedProvisioningState' is not allowed.",
		},
		{
			name: "lastAdminUpdateError change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.LastAdminUpdateError = "error" },
			wantErr: "400: PropertyChangeNotAllowed: properties.lastAdminUpdateError: Changing property 'properties.lastAdminUpdateError' is not allowed.",
		},
		{
			name: "console url change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						ConsoleProfile: ConsoleProfile{
							URL: "https://console-openshift-console.apps.cluster.location.aroapp.io/",
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ConsoleProfile.URL = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.consoleProfile.url: Changing property 'properties.consoleProfile.url' is not allowed.",
		},
		{
			name: "domain change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						ClusterProfile: ClusterProfile{
							Domain: "cluster.location.aroapp.io",
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ClusterProfile.Domain = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.clusterProfile.domain: Changing property 'properties.clusterProfile.domain' is not allowed.",
		},
		{
			name: "version change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						ClusterProfile: ClusterProfile{
							Version: "4.3.0",
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ClusterProfile.Version = "" },
			wantErr: "400: PropertyChangeNotAllowed: properties.clusterProfile.version: Changing property 'properties.clusterProfile.version' is not allowed.",
		},
		{
			name: "resource group change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						ClusterProfile: ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", subscriptionID),
						},
					},
				}
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.ResourceGroupID = oc.Properties.ClusterProfile.ResourceGroupID[:strings.LastIndexByte(oc.Properties.ClusterProfile.ResourceGroupID, '/')] + "/changed"
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.clusterProfile.resourceGroupId: Changing property 'properties.clusterProfile.resourceGroupId' is not allowed.",
		},
		{
			name: "apiServer private change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						APIServerProfile: APIServerProfile{
							Visibility: VisibilityPublic,
						},
					},
				}
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerProfile.Visibility = VisibilityPrivate
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.apiserverProfile.visibility: Changing property 'properties.apiserverProfile.visibility' is not allowed.",
		},
		{
			name: "apiServer url change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						APIServerProfile: APIServerProfile{
							URL: "https://api.cluster.location.aroapp.io:6443/",
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.APIServerProfile.URL = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.apiserverProfile.url: Changing property 'properties.apiserverProfile.url' is not allowed.",
		},
		{
			name: "apiServer ip change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						APIServerProfile: APIServerProfile{
							IP: "1.2.3.4",
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.APIServerProfile.IP = "2.3.4.5" },
			wantErr: "400: PropertyChangeNotAllowed: properties.apiserverProfile.ip: Changing property 'properties.apiserverProfile.ip' is not allowed.",
		},
		{
			name: "ingress private change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						IngressProfiles: []IngressProfile{
							{
								Name:       "default",
								Visibility: VisibilityPublic,
							},
						},
					},
				}
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.IngressProfiles[0].Visibility = VisibilityPrivate
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.ingressProfiles['default'].visibility: Changing property 'properties.ingressProfiles['default'].visibility' is not allowed.",
		},
		{
			name: "ingress ip change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						IngressProfiles: []IngressProfile{
							{
								Name: "default",
								IP:   "1.2.3.4",
							},
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.IngressProfiles[0].IP = "2.3.4.5" },
			wantErr: "400: PropertyChangeNotAllowed: properties.ingressProfiles['default'].ip: Changing property 'properties.ingressProfiles['default'].ip' is not allowed.",
		},
		{
			name: "clientId change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						ServicePrincipalProfile: ServicePrincipalProfile{
							ClientID: "clientId",
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ServicePrincipalProfile.ClientID = uuid.NewV4().String() },
			wantErr: "400: PropertyChangeNotAllowed: properties.servicePrincipalProfile.clientId: Changing property 'properties.servicePrincipalProfile.clientId' is not allowed.",
		},
		{
			name: "tenantId change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						ServicePrincipalProfile: ServicePrincipalProfile{
							TenantID: "tenantId",
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ServicePrincipalProfile.TenantID = uuid.NewV4().String() },
			wantErr: "400: PropertyChangeNotAllowed: properties.servicePrincipalProfile.tenantId: Changing property 'properties.servicePrincipalProfile.tenantId' is not allowed.",
		},
		{
			name: "podCidr change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						NetworkProfile: NetworkProfile{
							PodCIDR: "10.128.0.0/14",
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.NetworkProfile.PodCIDR = "0.0.0.0/0" },
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.podCidr: Changing property 'properties.networkProfile.podCidr' is not allowed.",
		},
		{
			name: "serviceCidr change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						NetworkProfile: NetworkProfile{
							PodCIDR: "172.30.0.0/16",
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.NetworkProfile.ServiceCIDR = "0.0.0.0/0" },
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.serviceCidr: Changing property 'properties.networkProfile.serviceCidr' is not allowed.",
		},
		{
			name: "privateEndpointIp change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						NetworkProfile: NetworkProfile{
							PrivateEndpointIP: "1.2.3.4",
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.NetworkProfile.PrivateEndpointIP = "4.3.2.1" },
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.privateEndpointIp: Changing property 'properties.networkProfile.privateEndpointIp' is not allowed.",
		},
		{
			name: "master subnetId change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						MasterProfile: MasterProfile{
							SubnetID: fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", subscriptionID),
						},
					},
				}
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.MasterProfile.SubnetID = oc.Properties.MasterProfile.SubnetID[:strings.LastIndexByte(oc.Properties.MasterProfile.SubnetID, '/')] + "/changed"
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.masterProfile.subnetId: Changing property 'properties.masterProfile.subnetId' is not allowed.",
		},
		{
			name: "master vmSize change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						MasterProfile: MasterProfile{
							VMSize: VMSizeStandardD8sV3,
						},
					},
				}
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.MasterProfile.VMSize = VMSizeStandardD4sV3
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.masterProfile.vmSize: Changing property 'properties.masterProfile.vmSize' is not allowed.",
		},
		{
			name: "worker vmSize change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						WorkerProfiles: []WorkerProfile{
							{
								Name:   "worker",
								VMSize: VMSizeStandardD2sV3,
							},
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].VMSize = VMSizeStandardD4sV3 },
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles['worker'].vmSize: Changing property 'properties.workerProfiles['worker'].vmSize' is not allowed.",
		},
		{
			name: "worker diskSizeGB change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						WorkerProfiles: []WorkerProfile{
							{
								Name:       "worker",
								DiskSizeGB: 128,
							},
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].DiskSizeGB++ },
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles['worker'].diskSizeGB: Changing property 'properties.workerProfiles['worker'].diskSizeGB' is not allowed.",
		},
		{
			name: "worker subnetId change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						WorkerProfiles: []WorkerProfile{
							{
								Name:     "worker",
								SubnetID: fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker", subscriptionID),
							},
						},
					},
				}
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].SubnetID = oc.Properties.WorkerProfiles[0].SubnetID[:strings.LastIndexByte(oc.Properties.WorkerProfiles[0].SubnetID, '/')] + "/changed"
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles['worker'].subnetId: Changing property 'properties.workerProfiles['worker'].subnetId' is not allowed.",
		},
		{
			name: "workerProfiles count change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						WorkerProfiles: []WorkerProfile{
							{
								Name:  "worker",
								Count: 3,
							},
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].Count++ },
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles['worker'].count: Changing property 'properties.workerProfiles['worker'].count' is not allowed.",
		},
		{
			name: "install phase change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						Install: &Install{
							Phase: InstallPhaseBootstrap,
						},
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.Install.Phase = InstallPhaseRemoveBootstrap },
			wantErr: "400: PropertyChangeNotAllowed: properties.install.phase: Changing property 'properties.install.phase' is not allowed.",
		},
		{
			name: "install now change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						Install: &Install{
							Now: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
						},
					},
				}
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.Install.Now = time.Date(1971, 1, 1, 0, 0, 0, 0, time.UTC)
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.install.now.ext: Changing property 'properties.install.now.ext' is not allowed.",
		},
		{
			name: "storageSuffix change is not allowed",
			oc: func() *OpenShiftCluster {
				return &OpenShiftCluster{
					Properties: OpenShiftClusterProperties{
						StorageSuffix: "rexs1",
					},
				}
			},
			modify:  func(oc *OpenShiftCluster) { oc.Properties.StorageSuffix = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.storageSuffix: Changing property 'properties.storageSuffix' is not allowed.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oc := tt.oc()
			if tt.modify != nil {
				tt.modify(oc)
			}

			current := &api.OpenShiftCluster{}
			(&openShiftClusterConverter{}).ToInternal(tt.oc(), current)

			v := &openShiftClusterStaticValidator{}
			err := v.Static(oc, current)
			if err == nil {
				if tt.wantErr != "" {
					t.Error(err)
				}

			} else {
				if err.Error() != tt.wantErr {
					t.Error(err)
				}

				cloudErr := err.(*api.CloudError)

				if cloudErr.StatusCode != http.StatusBadRequest {
					t.Error(cloudErr.StatusCode)
				}
				if cloudErr.Target == "" {
					t.Error("target is required")
				}

				validate.CloudError(t, err)
			}
		})
	}
}
