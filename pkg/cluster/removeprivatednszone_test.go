package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	fakemcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_privatedns "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/privatedns"
)

func TestRemovePrivateDNSZone(t *testing.T) {
	ctx := context.Background()
	const resourceGroupID = "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup"

	for _, tt := range []struct {
		name   string
		doc    *api.OpenShiftClusterDocument
		mocks  func(*mock_privatedns.MockPrivateZonesClient, *mock_privatedns.MockVirtualNetworkLinksClient)
		mcocli mcoclient.Interface
	}{
		{
			name: "no private zones",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: resourceGroupID,
						},
					},
				},
			},
			mocks: func(privateZones *mock_privatedns.MockPrivateZonesClient, virtualNetworkLinks *mock_privatedns.MockVirtualNetworkLinksClient) {
				privateZones.EXPECT().
					ListByResourceGroup(ctx, "testGroup", nil).
					Return(nil, nil)
			},
		},
		{
			name: "has private zone, dnsmasq config not yet reconciled",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: resourceGroupID,
						},
					},
				},
			},
			mocks: func(privateZones *mock_privatedns.MockPrivateZonesClient, virtualNetworkLinks *mock_privatedns.MockVirtualNetworkLinksClient) {
				privateZones.EXPECT().
					ListByResourceGroup(ctx, "testGroup", nil).
					Return([]mgmtprivatedns.PrivateZone{
						{
							ID: to.StringPtr("/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/privateZones/zone1"),
						},
					}, nil)
			},
			mcocli: fakemcoclient.NewSimpleClientset(
				&mcv1.MachineConfigPool{},
			),
		},
		{
			name: "has private zone, pool not yet ready",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: resourceGroupID,
						},
					},
				},
			},
			mocks: func(privateZones *mock_privatedns.MockPrivateZonesClient, virtualNetworkLinks *mock_privatedns.MockVirtualNetworkLinksClient) {
				privateZones.EXPECT().
					ListByResourceGroup(ctx, "testGroup", nil).
					Return([]mgmtprivatedns.PrivateZone{
						{
							ID: to.StringPtr("/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/privateZones/zone1"),
						},
					}, nil)
			},
			mcocli: fakemcoclient.NewSimpleClientset(
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "master",
					},
					Status: mcv1.MachineConfigPoolStatus{
						Configuration: mcv1.MachineConfigPoolStatusConfiguration{
							Source: []v1.ObjectReference{
								{
									Name: "99-master-aro-dns",
								},
							},
						},
						MachineCount: 1,
					},
				},
			),
		},
		{
			name: "has private zone, dnsmasq rolled out",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: resourceGroupID,
						},
					},
				},
			},
			mocks: func(privateZones *mock_privatedns.MockPrivateZonesClient, virtualNetworkLinks *mock_privatedns.MockVirtualNetworkLinksClient) {
				privateZones.EXPECT().
					ListByResourceGroup(ctx, "testGroup", nil).
					Return([]mgmtprivatedns.PrivateZone{
						{
							ID: to.StringPtr("/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/privateZones/zone1"),
						},
					}, nil)

				virtualNetworkLinks.EXPECT().
					List(ctx, "testGroup", "zone1", nil).
					Return([]mgmtprivatedns.VirtualNetworkLink{
						{
							Name: to.StringPtr("link1"),
						},
					}, nil)

				virtualNetworkLinks.EXPECT().
					DeleteAndWait(ctx, "testGroup", "zone1", "link1", "").
					Return(nil)

				privateZones.EXPECT().
					DeleteAndWait(ctx, "testGroup", "zone1", "").
					Return(nil)
			},
			mcocli: fakemcoclient.NewSimpleClientset(
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "master",
					},
					Status: mcv1.MachineConfigPoolStatus{
						Configuration: mcv1.MachineConfigPoolStatusConfiguration{
							Source: []v1.ObjectReference{
								{
									Name: "99-master-aro-dns",
								},
							},
						},
					},
				},
			),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			privateZones := mock_privatedns.NewMockPrivateZonesClient(controller)
			virtualNetworkLinks := mock_privatedns.NewMockVirtualNetworkLinksClient(controller)
			tt.mocks(privateZones, virtualNetworkLinks)

			m := &manager{
				log:                 logrus.NewEntry(logrus.StandardLogger()),
				doc:                 tt.doc,
				privateZones:        privateZones,
				virtualNetworkLinks: virtualNetworkLinks,
				mcocli:              tt.mcocli,
			}

			err := m.removePrivateDNSZone(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
