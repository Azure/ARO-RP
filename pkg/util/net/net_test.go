package net

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	mcofake "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	mock_privatedns "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/privatedns"
	mock_net "github.com/Azure/ARO-RP/pkg/util/mocks/net"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var (
	resourceGroupName = "testGroup"
	subscriptionID    = "0000000-0000-0000-0000-000000000000"
	resourceGroupID   = "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroupName
	vnetName          = "testVnet"
	resourceID        = resourceGroupID + "/providers/Microsoft.Network/virtualNetworks/" + vnetName
)

const id = "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/privateDnsZones/zone1"

func TestUpdateClusterDNSFn(t *testing.T) {
	type testCase struct {
		name                string
		resourceGroupID     string
		wantErr             string
		ensureMocksBehavior func(dnsI mock_net.MockDNSIClient)
	}

	testcases := []testCase{
		{
			name:    "should propagate the error from dnsi.Get",
			wantErr: "some error",
			ensureMocksBehavior: func(dnsI mock_net.MockDNSIClient) {
				dnsI.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("some error"))
			},
		},
		{
			name:            "should not call dnsI.Update when private zone id does not have the resource group id as prefix",
			resourceGroupID: "rg",
			ensureMocksBehavior: func(dnsI mock_net.MockDNSIClient) {
				v := &configv1.DNS{
					Spec: configv1.DNSSpec{
						PrivateZone: &configv1.DNSZone{
							ID: "not_same_rg_the_id",
						},
					},
				}
				dnsI.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(v, nil)
				dnsI.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
		},
		{
			name:            "should call dnsI.Update with Spec.PrivateZone equal to nil",
			resourceGroupID: "rg_",
			ensureMocksBehavior: func(dnsI mock_net.MockDNSIClient) {
				v := configv1.DNS{
					Spec: configv1.DNSSpec{
						PrivateZone: &configv1.DNSZone{
							ID: "rg_the_id",
						},
					},
				}

				dnsI.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(&v, nil)
				dnsI.EXPECT().Update(gomock.Any(), MatchesNilPrivateZone{}, gomock.Any())
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			dnsI := mock_net.NewMockDNSIClient(controller)

			if tc.ensureMocksBehavior != nil {
				tc.ensureMocksBehavior(*dnsI)
			}

			fn := updateClusterDNSFn(context.Background(), dnsI, tc.resourceGroupID)
			err := fn()
			utilerror.AssertErrorMessage(t, err, tc.wantErr)
		})
	}

	t.Run("dnsI is nil", func(t *testing.T) {
		fn := updateClusterDNSFn(context.Background(), nil, "")
		err := fn()
		utilerror.AssertErrorMessage(t, err, "dnsClient interface is nil")
	})
}

type MatchesNilPrivateZone struct {
}

func (MatchesNilPrivateZone) Matches(x interface{}) bool {
	arg, ok := x.(*configv1.DNS)
	if !ok {
		return false
	}

	return arg.Spec.PrivateZone == nil
}

func (MatchesNilPrivateZone) String() string {
	return "arg.Spec.PrivateZone does not match (is not nil)"
}

func TestDeletePrivateDNSVNetLinks(t *testing.T) {
	type testCase struct {
		name                string
		resourceID          string
		wantErr             string
		ensureMocksBehavior func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient)
	}
	testcases := []testCase{
		{
			name:       "propagates invalid resource id error",
			resourceID: "invalid_resourceId",
			wantErr:    "parsing failed for invalid_resourceId. Invalid resource Id format",
			ensureMocksBehavior: func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient) {
				vNetLinksClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("parsing failed for invalid_resourceId. Invalid resource Id format"))
			},
		},
		{
			name:    "propagates error from vNetLinksClient.List",
			wantErr: "some_error",
			ensureMocksBehavior: func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient) {
				vNetLinksClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("some_error"))
			},
		},
		{
			name:    "ppropagates error from vNetLinksClient.DeleteAndWait",
			wantErr: "some_error",
			ensureMocksBehavior: func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient) {
				name := "name"
				listResult := []mgmtprivatedns.VirtualNetworkLink{{Name: &name}}
				vNetLinksClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(listResult, nil)
				vNetLinksClient.EXPECT().DeleteAndWait(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("some_error"))
			},
		},
		{
			name: "returns nil when no errors found",
			ensureMocksBehavior: func(vNetLinksClient *mock_privatedns.MockVirtualNetworkLinksClient) {
				name := "name"
				listResult := []mgmtprivatedns.VirtualNetworkLink{{Name: &name}}
				vNetLinksClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(listResult, nil)
				vNetLinksClient.EXPECT().DeleteAndWait(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			vNetLinksClient := mock_privatedns.NewMockVirtualNetworkLinksClient(controller)

			if tc.ensureMocksBehavior != nil {
				tc.ensureMocksBehavior(vNetLinksClient)
			}

			err := DeletePrivateDNSVNetLinks(context.Background(), vNetLinksClient, resourceID)
			utilerror.AssertErrorMessage(t, err, tc.wantErr)
		})
	}
}

func TestRemovePrivateDNSZone(t *testing.T) {
	privateZone := []mgmtprivatedns.PrivateZone{{ID: to.StringPtr(id)}}

	doc := &api.OpenShiftClusterDocument{
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{ResourceGroupID: resourceGroupID},
			},
		},
	}

	dns := &configv1.DNS{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec: configv1.DNSSpec{
			PrivateZone: &configv1.DNSZone{ID: id},
		},
	}

	mcp := &mcv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "master",
		},
		Status: mcv1.MachineConfigPoolStatus{
			Configuration: mcv1.MachineConfigPoolStatusConfiguration{
				Source: []corev1.ObjectReference{{Name: "99-master-aro-dns"}},
			},
			MachineCount: 1,
		},
	}

	ctx := context.Background()

	for _, tt := range []struct {
		name                      string
		doc                       *api.OpenShiftClusterDocument
		mocks                     func(*mock_privatedns.MockPrivateZonesClient, *mock_privatedns.MockVirtualNetworkLinksClient)
		kubernetescli             kubernetes.Interface
		mcocli                    mcoclient.Interface
		configcli                 configclient.Interface
		wantDNSPrivateZoneRemoved bool
		wantError                 string
	}{
		{
			name: "no private zones",
			doc:  doc,
			mocks: func(privateZones *mock_privatedns.MockPrivateZonesClient, virtualNetworkLinks *mock_privatedns.MockVirtualNetworkLinksClient) {
				privateZones.EXPECT().ListByResourceGroup(ctx, "testGroup", nil).Return(nil, nil)
			},
			configcli:                 configfake.NewSimpleClientset(dns),
			wantDNSPrivateZoneRemoved: true,
		},
		{
			name: "has private zone, dnsmasq config not yet reconciled",
			doc:  doc,
			mocks: func(privateZones *mock_privatedns.MockPrivateZonesClient, virtualNetworkLinks *mock_privatedns.MockVirtualNetworkLinksClient) {
				privateZones.EXPECT().ListByResourceGroup(ctx, "testGroup", nil).Return([]mgmtprivatedns.PrivateZone{{ID: to.StringPtr(id)}}, nil)
			},
			mcocli:    mcofake.NewSimpleClientset(&mcv1.MachineConfigPool{}),
			configcli: configfake.NewSimpleClientset(),
			kubernetescli: fake.NewSimpleClientset(
				&corev1.Node{},
			),
		},
		{
			name: "has private zone, pool not yet ready",
			doc:  doc,
			mocks: func(privateZones *mock_privatedns.MockPrivateZonesClient, virtualNetworkLinks *mock_privatedns.MockVirtualNetworkLinksClient) {
				privateZones.EXPECT().ListByResourceGroup(ctx, "testGroup", nil).Return(privateZone, nil)
			},
			mcocli: mcofake.NewSimpleClientset(mcp),
			kubernetescli: fake.NewSimpleClientset(
				&corev1.Node{},
			),
			wantError: "configcli is nil",
		},
		{
			name: "has private zone, nodes match, 4.4, dnsmasq rolled out",
			doc:  doc,
			mocks: func(privateZones *mock_privatedns.MockPrivateZonesClient, virtualNetworkLinks *mock_privatedns.MockVirtualNetworkLinksClient) {
				privateZones.EXPECT().ListByResourceGroup(ctx, "testGroup", nil).Return(privateZone, nil)

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
			kubernetescli: fake.NewSimpleClientset(
				&corev1.Node{},
			),
			mcocli: mcofake.NewSimpleClientset(
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "master",
					},
					Status: mcv1.MachineConfigPoolStatus{
						Configuration: mcv1.MachineConfigPoolStatusConfiguration{
							Source: []corev1.ObjectReference{
								{
									Name: "99-master-aro-dns",
								},
							},
						},
						MachineCount:        1,
						UpdatedMachineCount: 1,
						ReadyMachineCount:   1,
					},
				},
			),
			configcli: configfake.NewSimpleClientset(
				&configv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name: "version",
					},
					Status: configv1.ClusterVersionStatus{
						History: []configv1.UpdateHistory{
							{
								State:   configv1.CompletedUpdate,
								Version: "4.4.0",
							},
						},
					},
				},
				dns,
			),
			wantDNSPrivateZoneRemoved: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			privateZones := mock_privatedns.NewMockPrivateZonesClient(controller)
			virtualNetworkLinks := mock_privatedns.NewMockVirtualNetworkLinksClient(controller)
			tt.mocks(privateZones, virtualNetworkLinks)

			config := PrivateZoneRemovalConfig{
				Log:                utillog.GetLogger(),
				PrivateZonesClient: privateZones,
				Configcli:          tt.configcli,
				Mcocli:             tt.mcocli,
				Kubernetescli:      tt.kubernetescli,
				VNetLinksClient:    virtualNetworkLinks,
				ResourceGroupID:    resourceGroupID,
			}
			err := RemovePrivateDNSZone(ctx, config)

			utilerror.AssertErrorMessage(t, err, tt.wantError)

			if tt.wantDNSPrivateZoneRemoved {
				dns, err := tt.configcli.ConfigV1().DNSes().Get(ctx, "cluster", metav1.GetOptions{})
				if err != nil {
					t.Fatal(err)
				}
				if dns.Spec.PrivateZone != nil {
					t.Error(dns.Spec.PrivateZone)
				}
			}
		})
	}
}
