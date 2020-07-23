package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mockcompute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mockfeatures "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mocknetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	mockdatabase "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
)

func TestAdminListResourcesList(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	clientkey, clientcerts, err := utiltls.GenerateKeyAndCertificate("client", nil, nil, false, true)
	if err != nil {
		t.Fatal(err)
	}

	serverkey, servercerts, err := utiltls.GenerateKeyAndCertificate("server", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(servercerts[0])

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
				Certificates: []tls.Certificate{
					{
						Certificate: [][]byte{clientcerts[0].Raw},
						PrivateKey:  clientkey,
					},
				},
			},
		},
	}

	type test struct {
		name           string
		resourceID     string
		mocks          func(*test, *mockdatabase.MockOpenShiftClusters, *mockdatabase.MockSubscriptions, *mockfeatures.MockResourcesClient, *mockcompute.MockVirtualMachinesClient, *mocknetwork.MockVirtualNetworksClient)
		wantStatusCode int
		wantResponse   func() []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "basic coverage",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, openshiftClusters *mockdatabase.MockOpenShiftClusters, subscriptions *mockdatabase.MockSubscriptions, resources *mockfeatures.MockResourcesClient, compute *mockcompute.MockVirtualMachinesClient, network *mocknetwork.MockVirtualNetworksClient) {
				clusterDoc := &api.OpenShiftClusterDocument{
					Key: tt.resourceID,
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
							MasterProfile: api.MasterProfile{
								SubnetID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockSubID),
							},
						},
					},
				}
				subscriptionDoc := &api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				}

				openshiftClusters.EXPECT().Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(clusterDoc, nil)

				subscriptions.EXPECT().Get(gomock.Any(), mockSubID).
					Return(subscriptionDoc, nil)

				resources.EXPECT().List(gomock.Any(), "resourceGroup eq 'test-cluster'", "", nil).Return([]mgmtfeatures.GenericResourceExpanded{
					{
						Name: to.StringPtr("vm-1"),
						ID:   to.StringPtr("/subscriptions/id"),
						Type: to.StringPtr("Microsoft.Compute/virtualMachines"),
					},
					{
						Name: to.StringPtr("storage"),
						ID:   to.StringPtr("/subscriptions/id"),
						Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
					},
				}, nil)

				network.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mgmtnetwork.VirtualNetwork{
					ID:   to.StringPtr("/subscriptions/id"),
					Type: to.StringPtr("Microsoft.Network/virtualNetworks"),
					VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
						DhcpOptions: &mgmtnetwork.DhcpOptions{
							DNSServers: &[]string{},
						},
					},
				}, nil)

				resources.EXPECT().GetByID(gomock.Any(), "/subscriptions/id", azureclient.APIVersions["Microsoft.Storage"]).Return(mgmtfeatures.GenericResource{
					Name:     to.StringPtr("storage"),
					ID:       to.StringPtr("/subscriptions/id"),
					Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
					Location: to.StringPtr("eastus"),
				}, nil)

				compute.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mgmtcompute.VirtualMachine{
					ID:   to.StringPtr("/subscriptions/id"),
					Type: to.StringPtr("Microsoft.Compute/virtualMachines"),
					VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
						ProvisioningState: to.StringPtr("Succeeded"),
					},
				}, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func() []byte {
				return []byte(`[{"properties":{"dhcpOptions":{"dnsServers":[]}},"id":"/subscriptions/id","type":"Microsoft.Network/virtualNetworks"},{"properties":{"provisioningState":"Succeeded"},"id":"/subscriptions/id","type":"Microsoft.Compute/virtualMachines"},{"id":"/subscriptions/id","name":"storage","type":"Microsoft.Storage/storageAccounts","location":"eastus"}]` + "\n")
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer cli.CloseIdleConnections()
			l := listener.NewListener()
			defer l.Close()
			env := &env.Test{
				L:            l,
				TestLocation: "eastus",
				TLSKey:       serverkey,
				TLSCerts:     servercerts,
			}
			env.SetAdminClientAuthorizer(clientauthorizer.NewOne(clientcerts[0].Raw))
			cli.Transport.(*http.Transport).Dial = l.Dial

			controller := gomock.NewController(t)
			defer controller.Finish()

			resourcesClient := mockfeatures.NewMockResourcesClient(controller)
			openshiftClusters := mockdatabase.NewMockOpenShiftClusters(controller)
			subscriptions := mockdatabase.NewMockSubscriptions(controller)
			vmClient := mockcompute.NewMockVirtualMachinesClient(controller)
			vNetClient := mocknetwork.NewMockVirtualNetworksClient(controller)
			tt.mocks(tt, openshiftClusters, subscriptions, resourcesClient, vmClient, vNetClient)

			f, err := New(ctx, logrus.NewEntry(logrus.StandardLogger()), env, &database.Database{
				OpenShiftClusters: openshiftClusters,
				Subscriptions:     subscriptions,
			}, api.APIs, &noop.Noop{}, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.WithResourcesClientFactory(resourcesClient).
				WithVMClientFactory(vmClient).
				WithVNetClientFactory(vNetClient).
				Run(ctx, nil, nil)
			url := fmt.Sprintf("https://server/admin/%s/resources", tt.resourceID)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := cli.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
			}

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if tt.wantError == "" {
				if tt.wantResponse != nil {
					if !bytes.Equal(b, tt.wantResponse()) {
						t.Error(string(b))
					}
				}
			} else {
				cloudErr := &api.CloudError{StatusCode: resp.StatusCode}
				err = json.Unmarshal(b, &cloudErr)
				if err != nil {
					t.Fatal(err)
				}

				if cloudErr.Error() != tt.wantError {
					t.Error(cloudErr)
				}
			}
		})
	}
}
