package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	mock_dns "github.com/Azure/ARO-RP/pkg/util/mocks/dns"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

const (
	privateIP = "10.0.0.1"
	publicIP  = "1.2.3.4"
)

func TestCreateOrUpdateRouterIPFromCluster(t *testing.T) {
	ctx := context.Background()

	const (
		key = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
	)

	for _, tt := range []struct {
		name           string
		kubernetescli  *fake.Clientset
		fixtureChecker func(*testdatabase.Fixture, *testdatabase.Checker, *cosmosdb.FakeOpenShiftClusterDocumentClient)
		mocks          func(*mock_dns.MockManager)
		wantErr        string
	}{
		{
			name: "create/update success",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							IngressProfiles: []api.IngressProfile{
								{
									Visibility: api.VisibilityPublic,
									Name:       "default",
								},
							},
							ProvisioningState: api.ProvisioningStateCreating,
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = publicIP
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(dns *mock_dns.MockManager) {
				dns.EXPECT().
					CreateOrUpdateRouter(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			kubernetescli: fake.NewSimpleClientset(&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "router-default",
					Namespace: "openshift-ingress",
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{
							IP: publicIP,
						}},
					},
				},
			}),
		},
		{
			name: "create/update failed - router IP issue",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							IngressProfiles: []api.IngressProfile{
								{
									Name: "default",
								},
							},
							ProvisioningState: api.ProvisioningStateCreating,
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				checker.AddOpenShiftClusterDocuments(doc)
			},
			kubernetescli: fake.NewSimpleClientset(&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "router-default",
					Namespace: "openshift-ingress",
				},
			}),
			wantErr: "routerIP not found",
		},
		{
			name: "enrich failed - return early",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							IngressProfiles:   nil,
							ProvisioningState: api.ProvisioningStateCreating,
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				checker.AddOpenShiftClusterDocuments(doc)
			},
			kubernetescli: fake.NewSimpleClientset(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			dns := mock_dns.NewMockManager(controller)
			if tt.mocks != nil {
				tt.mocks(dns)
			}

			dbOpenShiftClusters, dbClient := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters)
			checker := testdatabase.NewChecker()

			if tt.fixtureChecker != nil {
				tt.fixtureChecker(fixture, checker, dbClient)
			}

			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			doc, err := dbOpenShiftClusters.Dequeue(ctx)
			if err != nil {
				t.Fatal(err)
			}

			m := &manager{
				log:           logrus.NewEntry(logrus.StandardLogger()),
				doc:           doc,
				db:            dbOpenShiftClusters,
				dns:           dns,
				kubernetescli: tt.kubernetescli,
			}

			err = m.createOrUpdateRouterIPFromCluster(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			for _, err = range checker.CheckOpenShiftClusters(dbClient) {
				t.Error(err)
			}
		})
	}
}

func TestCreateOrUpdateRouterIPEarly(t *testing.T) {
	ctx := context.Background()

	const (
		key             = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
		resourceGroupID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterResourceGroup"
	)

	for _, tt := range []struct {
		name           string
		fixtureChecker func(*testdatabase.Fixture, *testdatabase.Checker, *cosmosdb.FakeOpenShiftClusterDocumentClient)
		mocks          func(*mock_network.MockPublicIPAddressesClient, *mock_dns.MockManager, *mock_subnet.MockManager)
		wantErr        string
	}{
		{
			name: "public",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: resourceGroupID,
							},
							IngressProfiles: []api.IngressProfile{
								{
									Visibility: api.VisibilityPublic,
								},
							},
							ProvisioningState: api.ProvisioningStateCreating,
							InfraID:           "infra",
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = publicIP
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(publicIPAddresses *mock_network.MockPublicIPAddressesClient, dns *mock_dns.MockManager, subnet *mock_subnet.MockManager) {
				publicIPAddresses.EXPECT().
					Get(gomock.Any(), "clusterResourceGroup", "infra-default-v4", "").
					Return(mgmtnetwork.PublicIPAddress{
						PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
							IPAddress: to.StringPtr(publicIP),
						},
					}, nil)
				dns.EXPECT().
					CreateOrUpdateRouter(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name: "private",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: resourceGroupID,
							},
							WorkerProfiles: []api.WorkerProfile{
								{
									SubnetID: "subnetid",
								},
							},
							IngressProfiles: []api.IngressProfile{
								{
									Visibility: api.VisibilityPrivate,
								},
							},
							ProvisioningState: api.ProvisioningStateCreating,
							InfraID:           "infra",
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = publicIP
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(publicIPAddresses *mock_network.MockPublicIPAddressesClient, dns *mock_dns.MockManager, subnet *mock_subnet.MockManager) {
				subnet.EXPECT().
					GetHighestFreeIP(gomock.Any(), "subnetid").
					Return(publicIP, nil)
				dns.EXPECT().
					CreateOrUpdateRouter(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name: "private - use enriched worker profile",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: resourceGroupID,
							},
							WorkerProfiles: []api.WorkerProfile{
								{
									SubnetID: "subnetid",
								},
							},
							WorkerProfilesStatus: []api.WorkerProfile{
								{
									SubnetID: "enricheWPsubnetid",
								},
							},
							IngressProfiles: []api.IngressProfile{
								{
									Visibility: api.VisibilityPrivate,
								},
							},
							ProvisioningState: api.ProvisioningStateCreating,
							InfraID:           "infra",
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = publicIP
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(publicIPAddresses *mock_network.MockPublicIPAddressesClient, dns *mock_dns.MockManager, subnet *mock_subnet.MockManager) {
				subnet.EXPECT().
					GetHighestFreeIP(gomock.Any(), "enricheWPsubnetid").
					Return(publicIP, nil)
				dns.EXPECT().
					CreateOrUpdateRouter(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			publicIPAddresses := mock_network.NewMockPublicIPAddressesClient(controller)
			dns := mock_dns.NewMockManager(controller)
			subnet := mock_subnet.NewMockManager(controller)
			if tt.mocks != nil {
				tt.mocks(publicIPAddresses, dns, subnet)
			}

			dbOpenShiftClusters, dbClient := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters)
			checker := testdatabase.NewChecker()

			if tt.fixtureChecker != nil {
				tt.fixtureChecker(fixture, checker, dbClient)
			}

			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			doc, err := dbOpenShiftClusters.Dequeue(ctx)
			if err != nil {
				t.Fatal(err)
			}

			m := &manager{
				doc:               doc,
				db:                dbOpenShiftClusters,
				publicIPAddresses: publicIPAddresses,
				dns:               dns,
				subnet:            subnet,
			}

			err = m.createOrUpdateRouterIPEarly(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			for _, err = range checker.CheckOpenShiftClusters(dbClient) {
				t.Error(err)
			}
		})
	}
}

func TestPopulateDatabaseIntIP(t *testing.T) {
	ctx := context.Background()

	const (
		key             = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
		resourceGroupID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterResourceGroup"
	)

	for _, tt := range []struct {
		name           string
		fixtureChecker func(*testdatabase.Fixture, *testdatabase.Checker, *cosmosdb.FakeOpenShiftClusterDocumentClient)
		mocks          func(*mock_network.MockLoadBalancersClient)
		wantErr        string
	}{
		{
			name: "v1",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ArchitectureVersion: api.ArchitectureVersionV1,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: resourceGroupID,
							},
							ProvisioningState: api.ProvisioningStateCreating,
							InfraID:           "infra",
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				doc.OpenShiftCluster.Properties.APIServerProfile.IntIP = privateIP
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(loadBalancers *mock_network.MockLoadBalancersClient) {
				loadBalancers.EXPECT().
					Get(gomock.Any(), "clusterResourceGroup", "infra-internal-lb", "").
					Return(mgmtnetwork.LoadBalancer{
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
								{
									FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
										PrivateIPAddress: to.StringPtr(privateIP),
									},
								},
							},
						},
					}, nil)
			},
		},
		{
			name: "v2",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ArchitectureVersion: api.ArchitectureVersionV2,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: resourceGroupID,
							},
							ProvisioningState: api.ProvisioningStateCreating,
							InfraID:           "infra",
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				doc.OpenShiftCluster.Properties.APIServerProfile.IntIP = privateIP
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(loadBalancers *mock_network.MockLoadBalancersClient) {
				loadBalancers.EXPECT().
					Get(gomock.Any(), "clusterResourceGroup", "infra-internal", "").
					Return(mgmtnetwork.LoadBalancer{
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
								{
									FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
										PrivateIPAddress: to.StringPtr(privateIP),
									},
								},
							},
						},
					}, nil)
			},
		},
		{
			name: "noop",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: resourceGroupID,
							},
							APIServerProfile: api.APIServerProfile{
								IntIP: privateIP,
							},
							ProvisioningState: api.ProvisioningStateCreating,
							InfraID:           "infra",
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				checker.AddOpenShiftClusterDocuments(doc)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			loadBalancers := mock_network.NewMockLoadBalancersClient(controller)
			if tt.mocks != nil {
				tt.mocks(loadBalancers)
			}

			dbOpenShiftClusters, dbClient := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters)
			checker := testdatabase.NewChecker()

			if tt.fixtureChecker != nil {
				tt.fixtureChecker(fixture, checker, dbClient)
			}

			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			doc, err := dbOpenShiftClusters.Dequeue(ctx)
			if err != nil {
				t.Fatal(err)
			}

			m := &manager{
				doc:           doc,
				db:            dbOpenShiftClusters,
				loadBalancers: loadBalancers,
			}

			err = m.populateDatabaseIntIP(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			for _, err = range checker.CheckOpenShiftClusters(dbClient) {
				t.Error(err)
			}
		})
	}
}

func TestUpdateAPIIPEarly(t *testing.T) {
	ctx := context.Background()

	const (
		key             = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
		resourceGroupID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterResourceGroup"
	)

	for _, tt := range []struct {
		name           string
		fixtureChecker func(*testdatabase.Fixture, *testdatabase.Checker, *cosmosdb.FakeOpenShiftClusterDocumentClient)
		mocks          func(*mock_armnetwork.MockLoadBalancersClient, *mock_armnetwork.MockPublicIPAddressesClient, *mock_dns.MockManager)
		wantErr        string
	}{
		{
			name: "public",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: resourceGroupID,
							},
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							ProvisioningState: api.ProvisioningStateCreating,
							InfraID:           "infra",
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				doc.OpenShiftCluster.Properties.APIServerProfile.IP = publicIP
				doc.OpenShiftCluster.Properties.APIServerProfile.IntIP = privateIP
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(loadBalancers *mock_armnetwork.MockLoadBalancersClient, publicIPAddresses *mock_armnetwork.MockPublicIPAddressesClient, dns *mock_dns.MockManager) {
				loadBalancers.EXPECT().
					Get(gomock.Any(), "clusterResourceGroup", "infra-internal", nil).
					Return(armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
											PrivateIPAddress: to.StringPtr(privateIP),
										},
									},
								},
							},
						},
					}, nil)
				publicIPAddresses.EXPECT().
					Get(gomock.Any(), "clusterResourceGroup", "infra-pip-v4", nil).
					Return(armnetwork.PublicIPAddressesClientGetResponse{
						PublicIPAddress: armnetwork.PublicIPAddress{
							Properties: &armnetwork.PublicIPAddressPropertiesFormat{
								IPAddress: to.StringPtr(publicIP),
							},
						},
					}, nil)
				dns.EXPECT().
					Update(gomock.Any(), gomock.Any(), publicIP).
					Return(nil)
			},
		},
		{
			name: "private",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: resourceGroupID,
							},
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPrivate,
							},
							ProvisioningState: api.ProvisioningStateCreating,
							InfraID:           "infra",
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				doc.OpenShiftCluster.Properties.APIServerProfile.IP = privateIP
				doc.OpenShiftCluster.Properties.APIServerProfile.IntIP = privateIP
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(loadBalancers *mock_armnetwork.MockLoadBalancersClient, publicIPAddresses *mock_armnetwork.MockPublicIPAddressesClient, dns *mock_dns.MockManager) {
				loadBalancers.EXPECT().
					Get(gomock.Any(), "clusterResourceGroup", "infra-internal", nil).
					Return(armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
											PrivateIPAddress: to.StringPtr(privateIP),
										},
									},
								},
							},
						},
					}, nil)
				dns.EXPECT().
					Update(gomock.Any(), gomock.Any(), privateIP).
					Return(nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			loadBalancers := mock_armnetwork.NewMockLoadBalancersClient(controller)
			publicIPAddresses := mock_armnetwork.NewMockPublicIPAddressesClient(controller)
			dns := mock_dns.NewMockManager(controller)
			if tt.mocks != nil {
				tt.mocks(loadBalancers, publicIPAddresses, dns)
			}

			dbOpenShiftClusters, dbClient := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters)
			checker := testdatabase.NewChecker()

			if tt.fixtureChecker != nil {
				tt.fixtureChecker(fixture, checker, dbClient)
			}

			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			doc, err := dbOpenShiftClusters.Dequeue(ctx)
			if err != nil {
				t.Fatal(err)
			}

			m := &manager{
				doc:                  doc,
				db:                   dbOpenShiftClusters,
				armPublicIPAddresses: publicIPAddresses,
				armLoadBalancers:     loadBalancers,
				dns:                  dns,
			}

			err = m.updateAPIIPEarly(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			for _, err = range checker.CheckOpenShiftClusters(dbClient) {
				t.Error(err)
			}
		})
	}
}

func TestEnsureGatewayCreate(t *testing.T) {
	ctx := context.Background()
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	for _, tt := range []struct {
		name                     string
		mocks                    func(*mock_env.MockInterface, *mock_armnetwork.MockPrivateEndpointsClient, *mock_armnetwork.MockPrivateLinkServicesClient)
		fixture                  func(*testdatabase.Fixture)
		checker                  func(*testdatabase.Checker)
		gatewayEnabled           bool
		gatewayPrivateEndpointIP string
		wantErr                  string
	}{
		{
			name: "noop: gateway not enabled",
		},
		{
			name:                     "noop: IP set",
			gatewayPrivateEndpointIP: privateIP,
		},
		{
			name: "error: private endpoint connection not found",
			mocks: func(env *mock_env.MockInterface, privateEndpoints *mock_armnetwork.MockPrivateEndpointsClient, rpPrivateLinkServices *mock_armnetwork.MockPrivateLinkServicesClient) {
				env.EXPECT().GatewayResourceGroup().AnyTimes().Return("gatewayResourceGroup")
				privateEndpoints.EXPECT().Get(ctx, "clusterResourceGroup", "infra-pe", &armnetwork.PrivateEndpointsClientGetOptions{Expand: to.StringPtr("networkInterfaces")}).Return(armnetwork.PrivateEndpointsClientGetResponse{
					PrivateEndpoint: armnetwork.PrivateEndpoint{
						Properties: &armnetwork.PrivateEndpointProperties{
							NetworkInterfaces: []*armnetwork.Interface{
								{
									Properties: &armnetwork.InterfacePropertiesFormat{
										IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
											{
												Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
													PrivateIPAddress: to.StringPtr(privateIP),
												},
											},
										},
									},
								},
							},
						},
						ID: to.StringPtr("peID"),
					},
				}, nil)
				rpPrivateLinkServices.EXPECT().Get(ctx, "gatewayResourceGroup", "gateway-pls-001", nil).Return(armnetwork.PrivateLinkServicesClientGetResponse{
					PrivateLinkService: armnetwork.PrivateLinkService{
						Properties: &armnetwork.PrivateLinkServiceProperties{
							PrivateEndpointConnections: []*armnetwork.PrivateEndpointConnection{},
						},
					},
				}, nil)
			},
			gatewayEnabled: true,
			wantErr:        "private endpoint connection not found",
		},
		{
			name: "ok",
			mocks: func(env *mock_env.MockInterface, privateEndpoints *mock_armnetwork.MockPrivateEndpointsClient, rpPrivateLinkServices *mock_armnetwork.MockPrivateLinkServicesClient) {
				env.EXPECT().GatewayResourceGroup().AnyTimes().Return("gatewayResourceGroup")
				privateEndpoints.EXPECT().Get(ctx, "clusterResourceGroup", "infra-pe", &armnetwork.PrivateEndpointsClientGetOptions{Expand: to.StringPtr("networkInterfaces")}).Return(armnetwork.PrivateEndpointsClientGetResponse{
					PrivateEndpoint: armnetwork.PrivateEndpoint{
						Properties: &armnetwork.PrivateEndpointProperties{
							NetworkInterfaces: []*armnetwork.Interface{
								{
									Properties: &armnetwork.InterfacePropertiesFormat{
										IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
											{
												Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
													PrivateIPAddress: to.StringPtr(privateIP),
												},
											},
										},
									},
								},
							},
						},
						ID: to.StringPtr("peID"),
					},
				}, nil)
				rpPrivateLinkServices.EXPECT().Get(ctx, "gatewayResourceGroup", "gateway-pls-001", nil).Return(armnetwork.PrivateLinkServicesClientGetResponse{
					PrivateLinkService: armnetwork.PrivateLinkService{
						Properties: &armnetwork.PrivateLinkServiceProperties{
							PrivateEndpointConnections: []*armnetwork.PrivateEndpointConnection{
								{
									Properties: &armnetwork.PrivateEndpointConnectionProperties{
										PrivateEndpoint: &armnetwork.PrivateEndpoint{
											ID: to.StringPtr("otherPeID"),
										},
									},
								},
								{
									Properties: &armnetwork.PrivateEndpointConnectionProperties{
										PrivateEndpoint: &armnetwork.PrivateEndpoint{
											ID: to.StringPtr("peID"),
										},
										PrivateLinkServiceConnectionState: &armnetwork.PrivateLinkServiceConnectionState{
											Status: to.StringPtr(""),
										},
										LinkIdentifier: to.StringPtr("1234"),
									},
									Name: to.StringPtr("conn"),
								},
							},
						},
					},
				}, nil)
				rpPrivateLinkServices.EXPECT().UpdatePrivateEndpointConnection(ctx, "gatewayResourceGroup", "gateway-pls-001", "conn", armnetwork.PrivateEndpointConnection{
					Properties: &armnetwork.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armnetwork.PrivateEndpoint{
							ID: to.StringPtr("peID"),
						},
						PrivateLinkServiceConnectionState: &armnetwork.PrivateLinkServiceConnectionState{
							Status:      to.StringPtr("Approved"),
							Description: to.StringPtr("Approved"),
						},
						LinkIdentifier: to.StringPtr("1234"),
					},
					Name: to.StringPtr("conn"),
				}, nil).Return(armnetwork.PrivateLinkServicesClientUpdatePrivateEndpointConnectionResponse{}, nil)
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: resourceID,
					},
				})
			},
			checker: func(c *testdatabase.Checker) {
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: resourceID,
						Properties: api.OpenShiftClusterProperties{
							NetworkProfile: api.NetworkProfile{
								GatewayPrivateEndpointIP: privateIP,
								GatewayPrivateLinkID:     "1234",
							},
						},
					},
				})
				c.AddGatewayDocuments(&api.GatewayDocument{
					ID: "1234",
					Gateway: &api.Gateway{
						ID:                              resourceID,
						StorageSuffix:                   "storageSuffix",
						ImageRegistryStorageAccountName: "imageRegistryStorageAccountName",
					},
				})
			},
			gatewayEnabled: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			privateEndpoints := mock_armnetwork.NewMockPrivateEndpointsClient(controller)
			rpPrivateLinkServices := mock_armnetwork.NewMockPrivateLinkServicesClient(controller)

			dbOpenShiftClusters, clientOpenShiftClusters := testdatabase.NewFakeOpenShiftClusters()
			dbGateway, clientGateway := testdatabase.NewFakeGateway()

			f := testdatabase.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters).WithGateway(dbGateway)
			if tt.mocks != nil {
				tt.mocks(env, privateEndpoints, rpPrivateLinkServices)
			}
			if tt.fixture != nil {
				tt.fixture(f)
			}
			err := f.Create()
			if err != nil {
				t.Fatal(err)
			}

			m := &manager{
				env:       env,
				db:        dbOpenShiftClusters,
				dbGateway: dbGateway,
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: resourceID,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: "/clusterResourceGroup",
							},
							NetworkProfile: api.NetworkProfile{
								GatewayPrivateEndpointIP: tt.gatewayPrivateEndpointIP,
							},
							FeatureProfile: api.FeatureProfile{
								GatewayEnabled: tt.gatewayEnabled,
							},
							StorageSuffix:                   "storageSuffix",
							ImageRegistryStorageAccountName: "imageRegistryStorageAccountName",
							InfraID:                         "infra",
						},
					},
				},
				armPrivateEndpoints:      privateEndpoints,
				armRPPrivateLinkServices: rpPrivateLinkServices,
			}

			err = m.ensureGatewayCreate(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			c := testdatabase.NewChecker()
			if tt.checker != nil {
				tt.checker(c)
			}

			errs := c.CheckOpenShiftClusters(clientOpenShiftClusters)
			for _, err := range errs {
				t.Error(err)
			}

			errs = c.CheckGateways(clientGateway)
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}
