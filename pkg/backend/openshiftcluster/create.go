package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"regexp"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/installconfig"
	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/ipnet"
	"github.com/openshift/installer/pkg/rhcos"
	"github.com/openshift/installer/pkg/types"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
	openstackvalidation "github.com/openshift/installer/pkg/types/openstack/validation"
	"github.com/openshift/installer/pkg/types/validation"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/install"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (m *Manager) Create(ctx context.Context) error {
	var err error

	m.doc, err = m.db.Patch(m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		var err error

		if doc.OpenShiftCluster.Properties.SSHKey == nil {
			doc.OpenShiftCluster.Properties.SSHKey, err = rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				return err
			}
		}

		if doc.OpenShiftCluster.Properties.StorageSuffix == "" {
			doc.OpenShiftCluster.Properties.StorageSuffix, err = randomLowerCaseAlphanumericString(5)
			if err != nil {
				return err
			}
		}

		if doc.OpenShiftCluster.Properties.DomainName == "" {
			doc.OpenShiftCluster.Properties.DomainName, err = randomDomainName()
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	r, err := azure.ParseResourceID(m.doc.OpenShiftCluster.ID)
	if err != nil {
		return err
	}

	vnetID, masterSubnetName, err := subnet.Split(m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	vnetID, workerSubnetName, err := subnet.Split(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].SubnetID)
	if err != nil {
		return err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	sshkey, err := ssh.NewPublicKey(&m.doc.OpenShiftCluster.Properties.SSHKey.PublicKey)
	if err != nil {
		return err
	}

	platformCreds := &installconfig.PlatformCreds{
		Azure: &icazure.Credentials{
			TenantID:       m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
			ClientID:       m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientID,
			ClientSecret:   m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret,
			SubscriptionID: r.SubscriptionID,
		},
	}

	installConfig := &installconfig.InstallConfig{
		Config: &types.InstallConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: m.doc.OpenShiftCluster.Properties.DomainName,
			},
			SSHKey:     sshkey.Type() + " " + base64.StdEncoding.EncodeToString(sshkey.Marshal()),
			BaseDomain: m.env.DNS().Domain(),
			Networking: &types.Networking{
				MachineCIDR: ipnet.MustParseCIDR("127.0.0.0/8"), // dummy
				NetworkType: "OpenShiftSDN",
				ClusterNetwork: []types.ClusterNetworkEntry{
					{
						CIDR:       *ipnet.MustParseCIDR(m.doc.OpenShiftCluster.Properties.NetworkProfile.PodCIDR),
						HostPrefix: 23,
					},
				},
				ServiceNetwork: []ipnet.IPNet{
					*ipnet.MustParseCIDR(m.doc.OpenShiftCluster.Properties.NetworkProfile.ServiceCIDR),
				},
			},
			ControlPlane: &types.MachinePool{
				Name:     "master",
				Replicas: to.Int64Ptr(3),
				Platform: types.MachinePoolPlatform{
					Azure: &azuretypes.MachinePool{
						InstanceType: string(m.doc.OpenShiftCluster.Properties.MasterProfile.VMSize),
					},
				},
				Hyperthreading: "Enabled",
			},
			Compute: []types.MachinePool{
				{
					Name:     m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].Name,
					Replicas: to.Int64Ptr(int64(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].Count)),
					Platform: types.MachinePoolPlatform{
						Azure: &azuretypes.MachinePool{
							InstanceType: string(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].VMSize),
							OSDisk: azuretypes.OSDisk{
								DiskSizeGB: int32(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].DiskSizeGB),
							},
						},
					},
					Hyperthreading: "Enabled",
				},
			},
			Platform: types.Platform{
				Azure: &azuretypes.Platform{
					Region:                      m.doc.OpenShiftCluster.Location,
					ResourceGroupName:           m.doc.OpenShiftCluster.Properties.ResourceGroup,
					BaseDomainResourceGroupName: m.env.ResourceGroup(),
					NetworkResourceGroupName:    vnetr.ResourceGroup,
					VirtualNetwork:              vnetr.ResourceName,
					ControlPlaneSubnet:          masterSubnetName,
					ComputeSubnet:               workerSubnetName,
					ARO:                         true,
				},
			},
			PullSecret: string(os.Getenv("PULL_SECRET")),
			Publish:    types.ExternalPublishingStrategy,
		},
	}

	installConfig.Config.Azure.Image, err = getImage(ctx)
	if err != nil {
		return err
	}

	err = validation.ValidateInstallConfig(installConfig.Config, openstackvalidation.NewValidValuesFetcher()).ToAggregate()
	if err != nil {
		return err
	}

	return install.NewInstaller(m.log, m.env, m.db, m.fpAuthorizer, r.SubscriptionID).Install(ctx, m.doc, installConfig, platformCreds)
}

var rxRHCOS = regexp.MustCompile(`rhcos-((\d+)\.\d+\.\d{8})\d{4}\.\d+-azure\.x86_64\.vhd`)

func getImage(ctx context.Context) (*azuretypes.Image, error) {
	// https://rhcos.blob.core.windows.net/imagebucket/rhcos-43.81.201911221453.0-azure.x86_64.vhd
	osImage, err := rhcos.VHD(ctx)
	if err != nil {
		return nil, err
	}

	m := rxRHCOS.FindStringSubmatch(osImage)
	if m == nil {
		return nil, fmt.Errorf("couldn't match osImage %q", osImage)
	}

	return &azuretypes.Image{
		Publisher: "azureopenshift",
		Offer:     "aro4",
		SKU:       "aro_" + m[2], // "aro_43"
		Version:   m[1],          // "43.81.20191122"
	}, nil
}

func randomDomainName() (string, error) {
	prefix, err := randomString("abcdefghijklmnopqrstuvwxyz", 1)
	if err != nil {
		return "", err
	}
	suffix, err := randomString("abcdefghijklmnopqrstuvwxyz0123456789", 7)
	if err != nil {
		return "", err
	}
	return prefix + suffix, nil
}

func randomLowerCaseAlphanumericString(n int) (string, error) {
	return randomString("abcdefghijklmnopqrstuvwxyz0123456789", n)
}

func randomString(letterBytes string, n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		o, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", err
		}
		b[i] = letterBytes[o.Int64()]
	}

	return string(b), nil
}
