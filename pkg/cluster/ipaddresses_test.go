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
				doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = "1.2.3.4"
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
							IP: "1.2.3.4",
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
				doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = "1.2.3.4"
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(publicIPAddresses *mock_network.MockPublicIPAddressesClient, dns *mock_dns.MockManager, subnet *mock_subnet.MockManager) {
				publicIPAddresses.EXPECT().
					Get(gomock.Any(), "clusterResourceGroup", "infra-default-v4", "").
					Return(mgmtnetwork.PublicIPAddress{
						PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
							IPAddress: to.StringPtr("1.2.3.4"),
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
				doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = "1.2.3.4"
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(publicIPAddresses *mock_network.MockPublicIPAddressesClient, dns *mock_dns.MockManager, subnet *mock_subnet.MockManager) {
				subnet.EXPECT().
					GetHighestFreeIP(gomock.Any(), "subnetid").
					Return("1.2.3.4", nil)
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
				doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = "1.2.3.4"
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(publicIPAddresses *mock_network.MockPublicIPAddressesClient, dns *mock_dns.MockManager, subnet *mock_subnet.MockManager) {
				subnet.EXPECT().
					GetHighestFreeIP(gomock.Any(), "enricheWPsubnetid").
					Return("1.2.3.4", nil)
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
				doc.OpenShiftCluster.Properties.APIServerProfile.IntIP = "10.0.0.1"
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
										PrivateIPAddress: to.StringPtr("10.0.0.1"),
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
				doc.OpenShiftCluster.Properties.APIServerProfile.IntIP = "10.0.0.1"
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
										PrivateIPAddress: to.StringPtr("10.0.0.1"),
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
								IntIP: "10.0.0.1",
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
				doc.OpenShiftCluster.Properties.APIServerProfile.IP = "1.2.3.4"
				doc.OpenShiftCluster.Properties.APIServerProfile.IntIP = "10.0.0.1"
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
											PrivateIPAddress: to.StringPtr("10.0.0.1"),
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
								IPAddress: to.StringPtr("1.2.3.4"),
							},
						},
					}, nil)
				dns.EXPECT().
					Update(gomock.Any(), gomock.Any(), "1.2.3.4").
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
				doc.OpenShiftCluster.Properties.APIServerProfile.IP = "10.0.0.1"
				doc.OpenShiftCluster.Properties.APIServerProfile.IntIP = "10.0.0.1"
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
											PrivateIPAddress: to.StringPtr("10.0.0.1"),
										},
									},
								},
							},
						},
					}, nil)
				dns.EXPECT().
					Update(gomock.Any(), gomock.Any(), "10.0.0.1").
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
		mocks                    func(*mock_env.MockInterface, *mock_network.MockPrivateEndpointsClient, *mock_network.MockPrivateLinkServicesClient)
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
			gatewayPrivateEndpointIP: "1.2.3.4",
		},
		{
			name: "error: private endpoint connection not found",
			mocks: func(env *mock_env.MockInterface, privateEndpoints *mock_network.MockPrivateEndpointsClient, rpPrivateLinkServices *mock_network.MockPrivateLinkServicesClient) {
				env.EXPECT().GatewayResourceGroup().AnyTimes().Return("gatewayResourceGroup")
				privateEndpoints.EXPECT().Get(ctx, "clusterResourceGroup", "infra-pe", "networkInterfaces").Return(mgmtnetwork.PrivateEndpoint{
					PrivateEndpointProperties: &mgmtnetwork.PrivateEndpointProperties{
						NetworkInterfaces: &[]mgmtnetwork.Interface{
							{
								InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
									IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
										{
											InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
												PrivateIPAddress: to.StringPtr("1.2.3.4"),
											},
										},
									},
								},
							},
						},
					},
					ID: to.StringPtr("peID"),
				}, nil)
				rpPrivateLinkServices.EXPECT().Get(ctx, "gatewayResourceGroup", "gateway-pls-001", "").Return(mgmtnetwork.PrivateLinkService{
					PrivateLinkServiceProperties: &mgmtnetwork.PrivateLinkServiceProperties{
						PrivateEndpointConnections: &[]mgmtnetwork.PrivateEndpointConnection{},
					},
				}, nil)
			},
			gatewayEnabled: true,
			wantErr:        "private endpoint connection not found",
		},
		{
			name: "ok",
			mocks: func(env *mock_env.MockInterface, privateEndpoints *mock_network.MockPrivateEndpointsClient, rpPrivateLinkServices *mock_network.MockPrivateLinkServicesClient) {
				env.EXPECT().GatewayResourceGroup().AnyTimes().Return("gatewayResourceGroup")
				privateEndpoints.EXPECT().Get(ctx, "clusterResourceGroup", "infra-pe", "networkInterfaces").Return(mgmtnetwork.PrivateEndpoint{
					PrivateEndpointProperties: &mgmtnetwork.PrivateEndpointProperties{
						NetworkInterfaces: &[]mgmtnetwork.Interface{
							{
								InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
									IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
										{
											InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
												PrivateIPAddress: to.StringPtr("1.2.3.4"),
											},
										},
									},
								},
							},
						},
					},
					ID: to.StringPtr("peID"),
				}, nil)
				rpPrivateLinkServices.EXPECT().Get(ctx, "gatewayResourceGroup", "gateway-pls-001", "").Return(mgmtnetwork.PrivateLinkService{
					PrivateLinkServiceProperties: &mgmtnetwork.PrivateLinkServiceProperties{
						PrivateEndpointConnections: &[]mgmtnetwork.PrivateEndpointConnection{
							{
								PrivateEndpointConnectionProperties: &mgmtnetwork.PrivateEndpointConnectionProperties{
									PrivateEndpoint: &mgmtnetwork.PrivateEndpoint{
										ID: to.StringPtr("otherPeID"),
									},
								},
							},
							{
								PrivateEndpointConnectionProperties: &mgmtnetwork.PrivateEndpointConnectionProperties{
									PrivateEndpoint: &mgmtnetwork.PrivateEndpoint{
										ID: to.StringPtr("peID"),
									},
									PrivateLinkServiceConnectionState: &mgmtnetwork.PrivateLinkServiceConnectionState{
										Status: to.StringPtr(""),
									},
									LinkIdentifier: to.StringPtr("1234"),
								},
								Name: to.StringPtr("conn"),
							},
						},
					},
				}, nil)
				rpPrivateLinkServices.EXPECT().UpdatePrivateEndpointConnection(ctx, "gatewayResourceGroup", "gateway-pls-001", "conn", mgmtnetwork.PrivateEndpointConnection{
					PrivateEndpointConnectionProperties: &mgmtnetwork.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &mgmtnetwork.PrivateEndpoint{
							ID: to.StringPtr("peID"),
						},
						PrivateLinkServiceConnectionState: &mgmtnetwork.PrivateLinkServiceConnectionState{
							Status:      to.StringPtr("Approved"),
							Description: to.StringPtr("Approved"),
						},
						LinkIdentifier: to.StringPtr("1234"),
					},
					Name: to.StringPtr("conn"),
				}).Return(mgmtnetwork.PrivateEndpointConnection{}, nil)
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
								GatewayPrivateEndpointIP: "1.2.3.4",
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
			privateEndpoints := mock_network.NewMockPrivateEndpointsClient(controller)
			rpPrivateLinkServices := mock_network.NewMockPrivateLinkServicesClient(controller)

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
				privateEndpoints:      privateEndpoints,
				rpPrivateLinkServices: rpPrivateLinkServices,
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
