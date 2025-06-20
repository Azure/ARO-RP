package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_dns "github.com/Azure/ARO-RP/pkg/util/mocks/dns"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
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
		mocks          func(*mock_armnetwork.MockPublicIPAddressesClient, *mock_dns.MockManager, *mock_armnetwork.MockSubnetsClient)
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
			mocks: func(publicIPAddresses *mock_armnetwork.MockPublicIPAddressesClient, dns *mock_dns.MockManager, subnet *mock_armnetwork.MockSubnetsClient) {
				subnet.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				publicIPAddresses.EXPECT().
					Get(gomock.Any(), "clusterResourceGroup", "infra-default-v4", nil).
					Return(armnetwork.PublicIPAddressesClientGetResponse{
						PublicIPAddress: armnetwork.PublicIPAddress{
							Properties: &armnetwork.PublicIPAddressPropertiesFormat{
								IPAddress: pointerutils.ToPtr(publicIP),
							},
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
									SubnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
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
				doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = "10.0.255.254"
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(publicIPAddresses *mock_armnetwork.MockPublicIPAddressesClient, dns *mock_dns.MockManager, subnet *mock_armnetwork.MockSubnetsClient) {
				publicIPAddresses.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				subnet.EXPECT().
					Get(gomock.Any(), "vnetResourceGroup", "vnet", "worker", gomock.Any()).
					Return(armnetwork.SubnetsClientGetResponse{
						Subnet: armnetwork.Subnet{
							Properties: &armnetwork.SubnetPropertiesFormat{
								AddressPrefix: pointerutils.ToPtr("10.0.0.0/16"),
							},
						},
					}, nil)
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
									SubnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
								},
							},
							WorkerProfilesStatus: []api.WorkerProfile{
								{
									SubnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/enrichedWorkerProfile",
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
				doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = "10.0.255.254"
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(publicIPAddresses *mock_armnetwork.MockPublicIPAddressesClient, dns *mock_dns.MockManager, subnet *mock_armnetwork.MockSubnetsClient) {
				publicIPAddresses.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				subnet.EXPECT().Get(gomock.Any(), "vnetResourceGroup", "vnet", "worker", gomock.Any()).Times(0)
				subnet.EXPECT().
					Get(gomock.Any(), "vnetResourceGroup", "vnet", "enrichedWorkerProfile", gomock.Any()).
					Return(armnetwork.SubnetsClientGetResponse{
						Subnet: armnetwork.Subnet{
							Properties: &armnetwork.SubnetPropertiesFormat{
								AddressPrefix: pointerutils.ToPtr("10.0.0.0/16"),
							},
						},
					}, nil)
				dns.EXPECT().
					CreateOrUpdateRouter(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			publicIPAddresses := mock_armnetwork.NewMockPublicIPAddressesClient(controller)
			dns := mock_dns.NewMockManager(controller)
			subnet := mock_armnetwork.NewMockSubnetsClient(controller)
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
				doc:                  doc,
				db:                   dbOpenShiftClusters,
				armPublicIPAddresses: publicIPAddresses,
				dns:                  dns,
				armSubnets:           subnet,
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
		mocks          func(*mock_armnetwork.MockLoadBalancersClient)
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
			mocks: func(loadBalancers *mock_armnetwork.MockLoadBalancersClient) {
				loadBalancers.EXPECT().
					Get(gomock.Any(), "clusterResourceGroup", "infra-internal-lb", nil).
					Return(armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Name: pointerutils.ToPtr("doesntmatter"),
										Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
											PrivateIPAddress: pointerutils.ToPtr(privateIP),
										},
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
			mocks: func(loadBalancers *mock_armnetwork.MockLoadBalancersClient) {
				loadBalancers.EXPECT().
					Get(gomock.Any(), "clusterResourceGroup", "infra-internal", nil).
					Return(armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{
							Properties: &armnetwork.LoadBalancerPropertiesFormat{
								FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
									{
										Name: pointerutils.ToPtr("doesntmatter"),
										Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
											PrivateIPAddress: pointerutils.ToPtr(privateIP),
										},
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

			loadBalancersClient := mock_armnetwork.NewMockLoadBalancersClient(controller)
			if tt.mocks != nil {
				tt.mocks(loadBalancersClient)
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
				doc:              doc,
				db:               dbOpenShiftClusters,
				armLoadBalancers: loadBalancersClient,
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
											PrivateIPAddress: pointerutils.ToPtr(privateIP),
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
								IPAddress: pointerutils.ToPtr(publicIP),
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
											PrivateIPAddress: pointerutils.ToPtr(privateIP),
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
				privateEndpoints.EXPECT().Get(ctx, "clusterResourceGroup", "infra-pe", &armnetwork.PrivateEndpointsClientGetOptions{Expand: pointerutils.ToPtr("networkInterfaces")}).Return(armnetwork.PrivateEndpointsClientGetResponse{
					PrivateEndpoint: armnetwork.PrivateEndpoint{
						Properties: &armnetwork.PrivateEndpointProperties{
							NetworkInterfaces: []*armnetwork.Interface{
								{
									Properties: &armnetwork.InterfacePropertiesFormat{
										IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
											{
												Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
													PrivateIPAddress: pointerutils.ToPtr(privateIP),
												},
											},
										},
									},
								},
							},
						},
						ID: pointerutils.ToPtr("peID"),
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
				privateEndpoints.EXPECT().Get(ctx, "clusterResourceGroup", "infra-pe", &armnetwork.PrivateEndpointsClientGetOptions{Expand: pointerutils.ToPtr("networkInterfaces")}).Return(armnetwork.PrivateEndpointsClientGetResponse{
					PrivateEndpoint: armnetwork.PrivateEndpoint{
						Properties: &armnetwork.PrivateEndpointProperties{
							NetworkInterfaces: []*armnetwork.Interface{
								{
									Properties: &armnetwork.InterfacePropertiesFormat{
										IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
											{
												Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
													PrivateIPAddress: pointerutils.ToPtr(privateIP),
												},
											},
										},
									},
								},
							},
						},
						ID: pointerutils.ToPtr("peID"),
					},
				}, nil)
				rpPrivateLinkServices.EXPECT().Get(ctx, "gatewayResourceGroup", "gateway-pls-001", nil).Return(armnetwork.PrivateLinkServicesClientGetResponse{
					PrivateLinkService: armnetwork.PrivateLinkService{
						Properties: &armnetwork.PrivateLinkServiceProperties{
							PrivateEndpointConnections: []*armnetwork.PrivateEndpointConnection{
								{
									Properties: &armnetwork.PrivateEndpointConnectionProperties{
										PrivateEndpoint: &armnetwork.PrivateEndpoint{
											ID: pointerutils.ToPtr("otherPeID"),
										},
									},
								},
								{
									Properties: &armnetwork.PrivateEndpointConnectionProperties{
										PrivateEndpoint: &armnetwork.PrivateEndpoint{
											ID: pointerutils.ToPtr("peID"),
										},
										PrivateLinkServiceConnectionState: &armnetwork.PrivateLinkServiceConnectionState{
											Status: pointerutils.ToPtr(""),
										},
										LinkIdentifier: pointerutils.ToPtr("1234"),
									},
									Name: pointerutils.ToPtr("conn"),
								},
							},
						},
					},
				}, nil)
				rpPrivateLinkServices.EXPECT().UpdatePrivateEndpointConnection(ctx, "gatewayResourceGroup", "gateway-pls-001", "conn", armnetwork.PrivateEndpointConnection{
					Properties: &armnetwork.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armnetwork.PrivateEndpoint{
							ID: pointerutils.ToPtr("peID"),
						},
						PrivateLinkServiceConnectionState: &armnetwork.PrivateLinkServiceConnectionState{
							Status:      pointerutils.ToPtr("Approved"),
							Description: pointerutils.ToPtr("Approved"),
						},
						LinkIdentifier: pointerutils.ToPtr("1234"),
					},
					Name: pointerutils.ToPtr("conn"),
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

func TestGetHighestFreeIP(t *testing.T) {
	const subnetID = "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet"

	type test struct {
		name    string
		subnet  armnetwork.Subnet
		wantIP  string
		wantErr string
	}

	for _, tt := range []*test{
		{
			name: "valid",
			subnet: armnetwork.Subnet{
				ID: pointerutils.ToPtr(subnetID),
				Properties: &armnetwork.SubnetPropertiesFormat{
					AddressPrefix: pointerutils.ToPtr("10.0.0.0/29"),
				},
			},
			wantIP: "10.0.0.6",
		},
		{
			name: "valid, use addressPrefixes",
			subnet: armnetwork.Subnet{
				ID: pointerutils.ToPtr(subnetID),
				Properties: &armnetwork.SubnetPropertiesFormat{
					AddressPrefixes: []*string{pointerutils.ToPtr("10.0.0.0/29")},
				},
			},
			wantIP: "10.0.0.6",
		},
		{
			name: "valid, top address used",
			subnet: armnetwork.Subnet{
				ID: pointerutils.ToPtr(subnetID),
				Properties: &armnetwork.SubnetPropertiesFormat{
					AddressPrefix: pointerutils.ToPtr("10.0.0.0/29"),
					IPConfigurations: []*armnetwork.IPConfiguration{
						{
							Properties: &armnetwork.IPConfigurationPropertiesFormat{
								PrivateIPAddress: pointerutils.ToPtr("10.0.0.6"),
							},
						},
						{
							Properties: &armnetwork.IPConfigurationPropertiesFormat{},
						},
					},
				},
			},
			wantIP: "10.0.0.5",
		},
		{
			name: "exhausted",
			subnet: armnetwork.Subnet{
				ID: pointerutils.ToPtr(subnetID),
				Properties: &armnetwork.SubnetPropertiesFormat{
					AddressPrefix: pointerutils.ToPtr("10.0.0.0/29"),
					IPConfigurations: []*armnetwork.IPConfiguration{
						{
							Properties: &armnetwork.IPConfigurationPropertiesFormat{
								PrivateIPAddress: pointerutils.ToPtr("10.0.0.4"),
							},
						},
						{
							Properties: &armnetwork.IPConfigurationPropertiesFormat{
								PrivateIPAddress: pointerutils.ToPtr("10.0.0.5"),
							},
						},
						{
							Properties: &armnetwork.IPConfigurationPropertiesFormat{
								PrivateIPAddress: pointerutils.ToPtr("10.0.0.6"),
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ip, err := getHighestFreeIP(&tt.subnet)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if ip != tt.wantIP {
				t.Error(ip)
			}
		})
	}
}
