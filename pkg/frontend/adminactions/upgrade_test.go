package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func TestUpgradeCluster(t *testing.T) {
	ctx := context.Background()

	stream43 := &version.Stream{
		Version: version.NewVersion(4, 3, 27),
	}
	stream44 := &version.Stream{
		Version: version.NewVersion(4, 4, 10),
	}
	stream45 := &version.Stream{
		Version: version.NewVersion(4, 5, 3),
	}

	newFakecli := func(status configv1.ClusterVersionStatus) *fake.Clientset {
		return fake.NewSimpleClientset(&configv1.ClusterVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name: "version",
			},
			Spec: configv1.ClusterVersionSpec{
				Channel: "",
			},
			Status: status,
		})
	}

	for _, tt := range []struct {
		name    string
		fakecli *fake.Clientset

		desiredVersion string
		upgradeY       bool
		wantUpdated    bool
		wantErr        string
	}{
		{
			name: "unhealthy cluster",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: stream43.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionFalse,
					},
				},
			}),
			wantErr: "500: InternalServerError: : Not upgrading: cvo is unhealthy.",
		},
		{
			name: "upgrade to Y latest",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: "4.3.1",
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
			desiredVersion: stream43.Version.String(),
			wantUpdated:    true,
		},
		{
			name: "no upgrade, Y higher than expected",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: "4.3.99",
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
		},
		{
			name: "no upgrade, Y match but unhealthy cluster",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: stream43.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionFalse,
					},
				},
			}),
			wantErr: "500: InternalServerError: : Not upgrading: cvo is unhealthy.",
		},
		{
			name: "upgrade, Y match, Y upgrades NOT allowed",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: stream43.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
		},
		{
			name: "upgrade, Y match, Y upgrades allowed (4.3 to 4.4)",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: stream43.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
			desiredVersion: stream44.Version.String(),
			upgradeY:       true,
			wantUpdated:    true,
		},
		{
			name: "upgrade, Y match, Y upgrades allowed (4.4 to 4.5)",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: stream44.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
			desiredVersion: stream45.Version.String(),
			upgradeY:       true,
			wantUpdated:    true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var updated bool

			tt.fakecli.PrependReactor("update", "clusterversions", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				updated = true
				return false, nil, nil
			})

			a := &adminactions{
				log:          logrus.NewEntry(logrus.StandardLogger()),
				configClient: tt.fakecli,
			}

			err := upgrade(ctx, a.log, a.configClient, []*version.Stream{stream43, stream44, stream45}, tt.upgradeY)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if updated != tt.wantUpdated {
				t.Fatal(updated)
			}

			cv, err := a.configClient.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			if tt.wantUpdated {
				if cv.Spec.DesiredUpdate == nil {
					t.Fatal(cv.Spec.DesiredUpdate)
				}
				if cv.Spec.DesiredUpdate.Version != tt.desiredVersion {
					t.Error(cv.Spec.DesiredUpdate.Version)
				}
			}
		})
	}
}

func TestCheckCustomDNS(t *testing.T) {
	ctx := context.Background()
	subscriptionID := "af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1"

	tests := []struct {
		name    string
		mocks   func(*mock_network.MockVirtualNetworksClient)
		wantErr string
	}{
		{
			name: "default dns",
			mocks: func(vnetc *mock_network.MockVirtualNetworksClient) {
				vnetc.EXPECT().Get(gomock.Any(), "test-cluster", "test-vnet", "").Return(
					mgmtnetwork.VirtualNetwork{
						VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
							DhcpOptions: &mgmtnetwork.DhcpOptions{
								DNSServers: &[]string{},
							},
						},
					}, nil)
			},
		},
		{
			name: "custom dns",
			mocks: func(vnetc *mock_network.MockVirtualNetworksClient) {
				vnetc.EXPECT().Get(gomock.Any(), "test-cluster", "test-vnet", "").Return(
					mgmtnetwork.VirtualNetwork{
						VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
							DhcpOptions: &mgmtnetwork.DhcpOptions{
								DNSServers: &[]string{"1.1.1.1"},
							},
						},
					}, nil)
			},
			wantErr: "not upgrading: custom DNS is set",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			vnetClient := mock_network.NewMockVirtualNetworksClient(controller)
			if tt.mocks != nil {
				tt.mocks(vnetClient)
			}

			a := &adminactions{
				log:        logrus.NewEntry(logrus.StandardLogger()),
				vNetClient: vnetClient,
			}

			oc := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", subscriptionID),
					},
				},
			}

			err := checkCustomDNS(ctx, oc, a.vNetClient)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
