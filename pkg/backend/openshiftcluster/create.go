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
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/install"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (m *Manager) Create(ctx context.Context) error {
	var err error

	if m.doc.OpenShiftCluster.Properties.Install == nil {
		// we don't re-call Dynamic on subsequent entries here.  One reason is
		// that we would re-check quota *after* we had deployed our VMs, and
		// could fail with a false positive.
		err = m.ocDynamicValidator.Dynamic(ctx, m.doc.OpenShiftCluster)
		if err != nil {
			return err
		}
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	if _, ok := m.env.(env.Dev); !ok {
		rp := m.acrtoken.GetRegistryProfile(m.doc.OpenShiftCluster)
		if rp == nil {
			// 1. choose a name and establish the intent to create a token with
			// that name
			rp = m.acrtoken.NewRegistryProfile(m.doc.OpenShiftCluster)

			m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
				m.acrtoken.PutRegistryProfile(doc.OpenShiftCluster, rp)
				return nil
			})
			if err != nil {
				return err
			}
		}

		if rp.Password == "" {
			// 2. ensure a token with the chosen name exists, generate a
			// password for it and store it in the database
			password, err := m.acrtoken.EnsureTokenAndPassword(ctx, rp)
			if err != nil {
				return err
			}

			rp.Password = api.SecureString(password)

			m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
				m.acrtoken.PutRegistryProfile(doc.OpenShiftCluster, rp)
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		if doc.OpenShiftCluster.Properties.SSHKey == nil {
			sshKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				return err
			}

			doc.OpenShiftCluster.Properties.SSHKey = x509.MarshalPKCS1PrivateKey(sshKey)
		}

		if doc.OpenShiftCluster.Properties.StorageSuffix == "" {
			doc.OpenShiftCluster.Properties.StorageSuffix, err = randomLowerCaseAlphanumericStringWithNoVowels(5)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	pullSecret := os.Getenv("PULL_SECRET")

	pullSecret, err = pullsecret.Merge(pullSecret, string(m.doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret))
	if err != nil {
		return err
	}

	pullSecret, _, err = pullsecret.SetRegistryProfiles(pullSecret, m.doc.OpenShiftCluster.Properties.RegistryProfiles...)
	if err != nil {
		return err
	}

	for _, key := range []string{"cloud.openshift.com"} {
		pullSecret, err = pullsecret.RemoveKey(pullSecret, key)
		if err != nil {
			return err
		}
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
				MachineNetwork: []types.MachineNetworkEntry{
					{
						CIDR: *ipnet.MustParseCIDR("127.0.0.0/8"), // dummy
					},
				},
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
				Architecture:   types.ArchitectureAMD64,
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
					Architecture:   types.ArchitectureAMD64,
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
			PullSecret: pullSecret,
			ImageContentSources: []types.ImageContentSource{
				{
					Source: "quay.io/openshift-release-dev/ocp-release",
					Mirrors: []string{
						fmt.Sprintf("%s.azurecr.io/openshift-release-dev/ocp-release", m.env.ACRName()),
					},
				},
				{
					Source: "quay.io/openshift-release-dev/ocp-release-nightly",
					Mirrors: []string{
						fmt.Sprintf("%s.azurecr.io/openshift-release-dev/ocp-release-nightly", m.env.ACRName()),
					},
				},
				{
					Source: "quay.io/openshift-release-dev/ocp-v4.0-art-dev",
					Mirrors: []string{
						fmt.Sprintf("%s.azurecr.io/openshift-release-dev/ocp-v4.0-art-dev", m.env.ACRName()),
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
	osImage, err := rhcos.VHD(ctx, types.ArchitectureAMD64)
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

func randomLowerCaseAlphanumericStringWithNoVowels(n int) (string, error) {
	return randomString("bcdfghjklmnpqrstvwxyz0123456789", n)
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
