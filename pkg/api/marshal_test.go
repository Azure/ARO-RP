package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/encrypt"
)

func TestUnmarshalSecure(t *testing.T) {
	h := &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				ErrorIfNoField: true,
			},
		},
	}

	cipher, err := encrypt.New(make([]byte, 32))
	if err != nil {
		t.Error(err)
	}

	err = AddExtensions(&h.BasicHandle, cipher)
	if err != nil {
		t.Error(err)
	}

	for _, tt := range []struct {
		name   string
		modify func(doc *OpenShiftClusterDocument)
	}{
		{
			name: "noop",
		},
		{
			name: "secureByte",
			modify: func(doc *OpenShiftClusterDocument) {
				doc.OpenShiftCluster.Properties.AdminKubeconfig = []byte("adminKubeconfigSecure")
			},
		},
		{
			name: "secureString",
			modify: func(doc *OpenShiftClusterDocument) {
				doc.OpenShiftCluster.Properties.KubeadminPassword = "kubeadminPasswordSecure"
			},
		},
		{
			name: "rsa.PrivateKey",
			modify: func(doc *OpenShiftClusterDocument) {
				privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					t.Error(err)
				}
				doc.OpenShiftCluster.Properties.SSHKey = privateKey
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := validOpenShiftClusterDocument()
			if tt.modify != nil {
				tt.modify(input)
			}

			buf := &bytes.Buffer{}
			err = codec.NewEncoder(buf, h).Encode(input)
			if err != nil {
				t.Error(err)
			}
			data, err := ioutil.ReadAll(buf)
			if err != nil {
				t.Error(err)
			}

			output := &OpenShiftClusterDocument{}
			err = codec.NewDecoder(bytes.NewReader(data), h).Decode(output)
			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(output, input) {
				inputB, _ := json.Marshal(input)
				outputB, _ := json.Marshal(output)
				t.Errorf("wants: %s \n , got: %s \n ", string(inputB), string(outputB))
			}
		})
	}
}

func validOpenShiftClusterDocument() *OpenShiftClusterDocument {
	doc := &OpenShiftClusterDocument{}
	doc.OpenShiftCluster = exampleOpenShiftCluster()
	return doc
}

func exampleOpenShiftCluster() *OpenShiftCluster {
	return &OpenShiftCluster{
		ID:       "/subscriptions/subscriptionId/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName",
		Name:     "resourceName",
		Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
		Location: "location",
		Tags: map[string]string{
			"key": "value",
		},
		Properties: Properties{
			ProvisioningState: ProvisioningStateSucceeded,
			ClusterProfile: ClusterProfile{
				Domain:          "cluster.location.aroapp.io",
				Version:         "4.3.0",
				ResourceGroupID: "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup",
			},
			ConsoleProfile: ConsoleProfile{
				URL: "https://console-openshift-console.apps.cluster.location.aroapp.io/",
			},
			ServicePrincipalProfile: ServicePrincipalProfile{
				ClientSecret: "clientSecret",
				ClientID:     "clientId",
			},
			NetworkProfile: NetworkProfile{
				PodCIDR:     "10.128.0.0/14",
				ServiceCIDR: "172.30.0.0/16",
			},
			MasterProfile: MasterProfile{
				VMSize:   VMSizeStandardD8sV3,
				SubnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
			},
			WorkerProfiles: []WorkerProfile{
				{
					Name:       "worker",
					VMSize:     VMSizeStandardD2sV3,
					DiskSizeGB: 128,
					SubnetID:   "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
					Count:      3,
				},
			},
			APIServerProfile: APIServerProfile{
				Visibility: VisibilityPublic,
				URL:        "https://api.cluster.location.aroapp.io:6443/",
				IP:         "1.2.3.4",
			},
			IngressProfiles: []IngressProfile{
				{
					Name:       "default",
					Visibility: VisibilityPublic,
					IP:         "1.2.3.4",
				},
			},
		},
	}
}
