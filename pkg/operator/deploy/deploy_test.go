package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

var (
	subscriptionId    = "0000000-0000-0000-0000-000000000000"
	vnetResourceGroup = "vnet-rg"
	vnetName          = "vnet"
	subnetNameMaster  = "master"

	genevakey   *rsa.PrivateKey
	genevacerts []*x509.Certificate
)

func init() {
	var err error

	genevakey, genevacerts, err = utiltls.GenerateKeyAndCertificate("client", nil, nil, false, true)
	if err != nil {
		panic(err)
	}
}

func TestOperatorVersion(t *testing.T) {
	for _, tt := range []struct {
		name             string
		flags            api.OperatorFlags
		expectedPullspec string
	}{
		{
			name:             "default deploy uses version",
			expectedPullspec: "ver=" + version.GitCommit,
		},
		{
			name:             "feature flag version used if set",
			expectedPullspec: "ver=somethingnew",
			flags: api.OperatorFlags{
				"aro.operator.version": "somethingnew",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)
			_env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
			_env.EXPECT().Hostname().AnyTimes().Return("testhost")
			_env.EXPECT().Location().AnyTimes().Return("eastus")
			_env.EXPECT().ClusterGenevaLoggingSecret().AnyTimes().Return(genevakey, genevacerts[0])
			_env.EXPECT().ACRDomain().AnyTimes().Return("acr.example.com")
			_env.EXPECT().ClusterGenevaLoggingConfigVersion().AnyTimes().Return("1")
			_env.EXPECT().ClusterGenevaLoggingEnvironment().AnyTimes().Return("testenvironment")
			_env.EXPECT().ClusterGenevaLoggingAccount().AnyTimes().Return("testaccount")
			_env.EXPECT().ClusterGenevaLoggingNamespace().AnyTimes().Return("testnamespace")

			_env.EXPECT().AROOperatorImage(gomock.Any()).AnyTimes().DoAndReturn(
				func(hash string) string { return fmt.Sprintf("ver=%s", hash) },
			)

			cluster := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Domain: "example.com",
					},
					MasterProfile: api.MasterProfile{
						SubnetID: "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster,
					},
					IngressProfiles: []api.IngressProfile{
						{
							IP: "127.0.0.1",
						},
					},
					OperatorFlags: tt.flags.Copy(),
				},
			}

			deployer := &operator{
				env: _env,
				oc:  cluster,
			}

			resources, err := deployer.resources()
			if err != nil {
				t.Fatal(err)
			}

			for _, obj := range resources {
				if d, ok := obj.(*appsv1.Deployment); ok {
					for _, err := range deep.Equal(d.Spec.Template.Spec.Containers[0].Image, tt.expectedPullspec) {
						t.Error(err)
					}
				}
			}
		})
	}
}
