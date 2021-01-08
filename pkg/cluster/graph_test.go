package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"
	"github.com/openshift/installer/pkg/asset/installconfig"
	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/asset/rhcos"
	"github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/ipnet"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/azure"
	azuredefaults "github.com/openshift/installer/pkg/types/azure/defaults"
	"github.com/openshift/installer/pkg/types/validation"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestGraphRoundTrip builds a representative graph, then marshals and
// unmarshals it.  It tests that registeredTypes isn't missing any obvious keys,
// and along the way is a smoke test for graph.resolve, ensuring (among other
// things) that it does not attempt to read values from stdin and that it can
// find its assets.
func TestGraphRoundTrip(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}

	sshkey, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	installConfig := &installconfig.InstallConfig{
		Azure: icazure.NewMetadata(azure.PublicCloud, nil),
		Config: &types.InstallConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "dummy",
			},
			SSHKey:     sshkey.Type() + " " + base64.StdEncoding.EncodeToString(sshkey.Marshal()),
			BaseDomain: "dummy",
			Networking: &types.Networking{
				MachineNetwork: []types.MachineNetworkEntry{
					{
						CIDR: *ipnet.MustParseCIDR("10.0.0.0/16"),
					},
				},
				NetworkType: "OpenShiftSDN",
				ClusterNetwork: []types.ClusterNetworkEntry{
					{
						CIDR:       *ipnet.MustParseCIDR("10.128.0.0/14"),
						HostPrefix: 23,
					},
				},
				ServiceNetwork: []ipnet.IPNet{
					*ipnet.MustParseCIDR("172.30.0.0/16"),
				},
			},
			ControlPlane: &types.MachinePool{
				Name:           "master",
				Replicas:       to.Int64Ptr(3),
				Hyperthreading: "Enabled",
				Architecture:   types.ArchitectureAMD64,
			},
			Compute: []types.MachinePool{
				{
					Name:           "worker",
					Replicas:       to.Int64Ptr(3),
					Hyperthreading: "Enabled",
					Architecture:   types.ArchitectureAMD64,
				},
			},
			Platform: types.Platform{
				Azure: &azure.Platform{
					Region:                      "dummy",
					ResourceGroupName:           "dummy",
					BaseDomainResourceGroupName: "dummy",
				},
			},
			PullSecret: `{"auths":{"dummy":{"auth":"dummy"}}}`,
			Publish:    types.ExternalPublishingStrategy,
		},
	}

	azuredefaults.SetPlatformDefaults(installConfig.Config.Platform.Azure)

	errs := validation.ValidateInstallConfig(installConfig.Config).ToAggregate()
	if errs != nil {
		t.Fatal(errs.Error())
	}

	g := newGraph(installConfig)

	for _, a := range targets.Cluster {
		err = g.resolve(a)
		if err != nil {
			t.Fatal(err)
		}
	}

	b, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}

	var g2 graph

	err = json.Unmarshal(b, &g2)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(g, g2) {
		t.Fatal(deep.Equal(g, g2))
	}
}

func TestGraphMarshalledFormat(t *testing.T) {
	b := []byte(`{"*rhcos.Image":"testimage","*unknown.Key":"unknown.Value"}`)

	var g graph
	err := json.Unmarshal(b, &g)
	if err != nil {
		t.Fatal(err)
	}

	i := g.get(new(rhcos.Image)).(*rhcos.Image)
	if i == nil || *i != "testimage" {
		t.Fatal(i)
	}

	if g["*unknown.Key"] != "unknown.Value" {
		t.Fatal(g["*unknown.Key"])
	}

	b2, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b, b2) {
		t.Fatal(string(b))
	}
}
