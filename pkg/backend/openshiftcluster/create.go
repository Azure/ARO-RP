package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/installconfig"
	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/asset/releaseimage"
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
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (m *Manager) Create(ctx context.Context) error {
	var err error

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		if doc.OpenShiftCluster.Properties.SSHKey == nil {
			sshKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				return err
			}

			doc.OpenShiftCluster.Properties.SSHKey = x509.MarshalPKCS1PrivateKey(sshKey)
		}

		if doc.OpenShiftCluster.Properties.StorageSuffix == "" {
			doc.OpenShiftCluster.Properties.StorageSuffix, err = randomLowerCaseAlphanumericString(5)
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

	privateKey, err := x509.ParsePKCS1PrivateKey(m.doc.OpenShiftCluster.Properties.SSHKey)
	if err != nil {
		return err
	}

	sshkey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	domain := m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain
	if !strings.ContainsRune(domain, '.') {
		domain += "." + m.env.Domain()
	}

	masterZones, err := m.env.Zones(string(m.doc.OpenShiftCluster.Properties.MasterProfile.VMSize))
	if err != nil {
		return err
	}
	if len(masterZones) == 0 {
		masterZones = []string{""}
	}

	workerZones, err := m.env.Zones(string(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].VMSize))
	if err != nil {
		return err
	}
	if len(workerZones) == 0 {
		masterZones = []string{""}
	}

	platformCreds := &installconfig.PlatformCreds{
		Azure: &icazure.Credentials{
			TenantID:       m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
			ClientID:       m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientID,
			ClientSecret:   string(m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret),
			SubscriptionID: r.SubscriptionID,
		},
	}

	installConfig := &installconfig.InstallConfig{
		Config: &types.InstallConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: domain[:strings.IndexByte(domain, '.')],
			},
			SSHKey:     sshkey.Type() + " " + base64.StdEncoding.EncodeToString(sshkey.Marshal()),
			BaseDomain: domain[strings.IndexByte(domain, '.')+1:],
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
						Zones:        masterZones,
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
							Zones:        workerZones,
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
					Region:                   m.doc.OpenShiftCluster.Location,
					ResourceGroupName:        resourceGroup,
					NetworkResourceGroupName: vnetr.ResourceGroup,
					VirtualNetwork:           vnetr.ResourceName,
					ControlPlaneSubnet:       masterSubnetName,
					ComputeSubnet:            workerSubnetName,
					ARO:                      true,
				},
			},
			PullSecret: os.Getenv("PULL_SECRET"),
			ImageContentSources: []types.ImageContentSource{
				{
					Source: "quay.io/openshift-release-dev/ocp-release",
					Mirrors: []string{
						"arosvc.azurecr.io/openshift-release-dev/ocp-release",
					},
				},
				{
					Source: "quay.io/openshift-release-dev/ocp-release-nightly",
					Mirrors: []string{
						"arosvc.azurecr.io/openshift-release-dev/ocp-release-nightly",
					},
				},
				{
					Source: "quay.io/openshift-release-dev/ocp-v4.0-art-dev",
					Mirrors: []string{
						"arosvc.azurecr.io/openshift-release-dev/ocp-v4.0-art-dev",
					},
				},
			},
			Publish: types.ExternalPublishingStrategy,
		},
	}

	if m.doc.OpenShiftCluster.Properties.IngressProfiles[0].Visibility == api.VisibilityPrivate {
		installConfig.Config.Publish = types.InternalPublishingStrategy
	}

	installConfig.Config.Azure.Image, err = getRHCOSImage(ctx)
	if err != nil {
		return err
	}

	image := &releaseimage.Image{}
	if m.doc.OpenShiftCluster.Properties.ClusterProfile.Version == version.OpenShiftVersion {
		image.PullSpec = version.OpenShiftPullSpec
	} else {
		return fmt.Errorf("unimplemented version %q", m.doc.OpenShiftCluster.Properties.ClusterProfile.Version)
	}

	err = validation.ValidateInstallConfig(installConfig.Config, openstackvalidation.NewValidValuesFetcher()).ToAggregate()
	if err != nil {
		return err
	}

	i, err := install.NewInstaller(ctx, m.log, m.env, m.db, m.billing, m.doc)
	if err != nil {
		return err
	}

	return i.Install(ctx, installConfig, platformCreds, image)
}

var rxRHCOS = regexp.MustCompile(`rhcos-((\d+)\.\d+\.\d{8})\d{4}\.\d+-azure\.x86_64\.vhd`)

func getRHCOSImage(ctx context.Context) (*azuretypes.Image, error) {
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
