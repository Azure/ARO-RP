package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/openshift/installer/pkg/asset/installconfig"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdb "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

type generateConfigTest struct {
	name    string
	fixture func(*testdb.Fixture)
	checker func(*installconfig.InstallConfig) []string
	wantErr string
}

func TestBackendTry(t *testing.T) {
	sshKeyGen, err := rsa.GenerateKey(rand.Reader, 128)
	if err != nil {
		t.Fatal(err)
	}

	sshKey := x509.MarshalPKCS1PrivateKey(sshKeyGen)

	mockSubID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)
	clusterResourceGroup := fmt.Sprintf("/subscriptions/%s/resourcegroups/cluster-rg", mockSubID)
	masterSubnet := fmt.Sprintf("/subscriptions/%s/resourcegroups/cluster-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockSubID)
	workerSubnet := fmt.Sprintf("/subscriptions/%s/resourcegroups/cluster-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", mockSubID)

	genGood := func() *api.OpenShiftClusterDocument {
		return &api.OpenShiftClusterDocument{
			Key: strings.ToLower(resourceID),
			OpenShiftCluster: &api.OpenShiftCluster{
				ID:       resourceID,
				Name:     "resourceName",
				Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
				Location: "location",
				Properties: api.OpenShiftClusterProperties{
					SSHKey:            sshKey,
					ProvisioningState: api.ProvisioningStateCreating,
					ClusterProfile: api.ClusterProfile{
						PullSecret:      `{"auths":{"registry.connect.redhat.com":{"auth":""},"registry.redhat.io":{"auth":""}}}`,
						ResourceGroupID: clusterResourceGroup,
						Domain:          "cluster.example.aroapp.io",
						Version:         version.InstallStream.Version.String(),
					},
					NetworkProfile: api.NetworkProfile{
						PodCIDR:     "10.128.0.0/14",
						ServiceCIDR: "172.30.0.0/16",
					},
					MasterProfile: api.MasterProfile{
						VMSize:   api.VMSizeStandardD8sV3,
						SubnetID: masterSubnet,
					},
					WorkerProfiles: []api.WorkerProfile{
						{
							Name:       "worker",
							VMSize:     api.VMSizeStandardD2sV3,
							DiskSizeGB: 128,
							SubnetID:   workerSubnet,
							Count:      3,
						},
					},
					IngressProfiles: []api.IngressProfile{
						{
							Name:       "default",
							Visibility: api.VisibilityPublic,
							IP:         "1.2.3.4",
						},
					},
				},
			},
		}
	}

	for _, tt := range []generateConfigTest{
		{
			name: "valid document generates valid installconfig",
			fixture: func(f *testdb.Fixture) {
				f.AddOpenShiftClusterDocuments(genGood())
			},
			checker: func(ic *installconfig.InstallConfig) []string {
				var errs []string
				errs = append(errs, deep.Equal(ic.Config.ObjectMeta.Name, "cluster")...)
				errs = append(errs, deep.Equal(ic.Config.BaseDomain, "example.aroapp.io")...)
				return errs
			},
		},
		{
			name: "incorrect version causes error",
			fixture: func(f *testdb.Fixture) {
				doc := genGood()
				doc.OpenShiftCluster.Properties.ClusterProfile.Version = "v3.11"
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantErr: `unimplemented version "v3.11"`,
		},
		{
			name: "incomplete domains have the environment base domain appended",
			fixture: func(f *testdb.Fixture) {
				doc := genGood()
				doc.OpenShiftCluster.Properties.ClusterProfile.Domain = "notadomain"
				f.AddOpenShiftClusterDocuments(doc)
			},
			checker: func(ic *installconfig.InstallConfig) []string {
				var errs []string
				errs = append(errs, deep.Equal(ic.Config.ObjectMeta.Name, "notadomain")...)
				errs = append(errs, deep.Equal(ic.Config.BaseDomain, "aroapp.io")...)
				return errs
			},
		},
		{
			name: "installer cluster validation failure causes error",
			fixture: func(f *testdb.Fixture) {
				doc := genGood()
				doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret = "{}"
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantErr: `pullSecret: Invalid value: "{}": auths required`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, log := testlog.New()

			controller := gomock.NewController(t)
			defer controller.Finish()
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().DeploymentMode().AnyTimes().Return(deployment.Development)
			_env.EXPECT().Zones(gomock.Any()).AnyTimes().Return([]string{"useast2a"}, nil)
			_env.EXPECT().ACRDomain().AnyTimes().Return("example.com")
			_env.EXPECT().Domain().AnyTimes().Return("aroapp.io")

			dbOpenShiftClusters, _ := testdb.NewFakeOpenShiftClusters()
			dbSubscriptions, _ := testdb.NewFakeSubscriptions()

			f := testdb.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters).WithSubscriptions(dbSubscriptions)
			tt.fixture(f)
			err := f.Create()
			if err != nil {
				t.Fatal(err)
			}

			doc, err := dbOpenShiftClusters.Get(ctx, strings.ToLower(resourceID))
			if err != nil {
				t.Error(err)
			}

			c := &manager{
				log: log,
				doc: doc,
				env: _env,
			}

			installConfig, platformCreds, image, err := c.generateInstallConfig(ctx)

			if tt.wantErr == "" {
				if err != nil {
					t.Error(err)
				}
				if installConfig == nil {
					t.Error(installConfig)
				}
				if platformCreds == nil {
					t.Error(platformCreds)
				}
				if image == nil {
					t.Error(installConfig)
				}
			} else {
				var errCheck string
				if err != nil {
					errCheck = err.Error()
				}
				errs := deep.Equal(tt.wantErr, errCheck)
				for _, err := range errs {
					t.Error(err)
				}
			}

			if tt.checker != nil {
				errs := tt.checker(installConfig)
				for _, err := range errs {
					t.Error(err)
				}
			}
		})
	}
}
