package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/installconfig"
	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/openshift/installer/pkg/ipnet"
	"github.com/openshift/installer/pkg/types"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
	"github.com/openshift/installer/pkg/types/validation"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
	"github.com/Azure/ARO-RP/pkg/util/rhcos"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (m *manager) generateInstallConfig(ctx context.Context) (*installconfig.InstallConfig, *releaseimage.Image, error) {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	pullSecret, err := pullsecret.Build(m.doc.OpenShiftCluster, string(m.doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret))
	if err != nil {
		return nil, nil, err
	}

	for _, key := range []string{"cloud.openshift.com"} {
		pullSecret, err = pullsecret.RemoveKey(pullSecret, key)
		if err != nil {
			return nil, nil, err
		}
	}

	r, err := azure.ParseResourceID(m.doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, nil, err
	}

	_, masterSubnetName, err := subnet.Split(m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
	if err != nil {
		return nil, nil, err
	}

	vnetID, workerSubnetName, err := subnet.Split(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].SubnetID)
	if err != nil {
		return nil, nil, err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return nil, nil, err
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(m.doc.OpenShiftCluster.Properties.SSHKey)
	if err != nil {
		return nil, nil, err
	}

	sshkey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	domain := m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain
	if !strings.ContainsRune(domain, '.') {
		domain += "." + m.env.Domain()
	}

	masterSKU, err := m.env.VMSku(string(m.doc.OpenShiftCluster.Properties.MasterProfile.VMSize))
	if err != nil {
		return nil, nil, err
	}
	masterZones := computeskus.Zones(masterSKU)
	if len(masterZones) == 0 {
		masterZones = []string{""}
	}

	workerSKU, err := m.env.VMSku(string(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].VMSize))
	if err != nil {
		return nil, nil, err
	}
	workerZones := computeskus.Zones(workerSKU)
	if len(workerZones) == 0 {
		workerZones = []string{""}
	}

	SoftwareDefinedNetwork := string(api.SoftwareDefinedNetworkOpenShiftSDN)
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.SoftwareDefinedNetwork != "" {
		SoftwareDefinedNetwork = string(m.doc.OpenShiftCluster.Properties.NetworkProfile.SoftwareDefinedNetwork)
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
				NetworkType: SoftwareDefinedNetwork,
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
						Zones:            masterZones,
						InstanceType:     string(m.doc.OpenShiftCluster.Properties.MasterProfile.VMSize),
						EncryptionAtHost: m.doc.OpenShiftCluster.Properties.MasterProfile.EncryptionAtHost == api.EncryptionAtHostEnabled,
						OSDisk: azuretypes.OSDisk{
							DiskEncryptionSetID: m.doc.OpenShiftCluster.Properties.MasterProfile.DiskEncryptionSetID,
							DiskSizeGB:          1024,
						},
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
							Zones:            workerZones,
							InstanceType:     string(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].VMSize),
							EncryptionAtHost: m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].EncryptionAtHost == api.EncryptionAtHostEnabled,
							OSDisk: azuretypes.OSDisk{
								DiskEncryptionSetID: m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].DiskEncryptionSetID,
								DiskSizeGB:          int32(m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].DiskSizeGB),
							},
						},
					},
					Hyperthreading: "Enabled",
					Architecture:   types.ArchitectureAMD64,
				},
			},
			Platform: types.Platform{
				Azure: &azuretypes.Platform{
					Region:                   strings.ToLower(m.doc.OpenShiftCluster.Location), // Used in k8s object names, so must pass DNS-1123 validation
					NetworkResourceGroupName: vnetr.ResourceGroup,
					VirtualNetwork:           vnetr.ResourceName,
					ControlPlaneSubnet:       masterSubnetName,
					ComputeSubnet:            workerSubnetName,
					CloudName:                azuretypes.CloudEnvironment(m.env.Environment().Name),
					OutboundType:             azuretypes.LoadbalancerOutboundType,
					ResourceGroupName:        resourceGroup,
				},
			},
			PullSecret: pullSecret,
			FIPS:       m.doc.OpenShiftCluster.Properties.ClusterProfile.FipsValidatedModules == api.FipsValidatedModulesEnabled,
			ImageContentSources: []types.ImageContentSource{
				{
					Source: "quay.io/openshift-release-dev/ocp-release",
					Mirrors: []string{
						fmt.Sprintf("%s/openshift-release-dev/ocp-release", m.env.ACRDomain()),
					},
				},
				{
					Source: "quay.io/openshift-release-dev/ocp-release-nightly",
					Mirrors: []string{
						fmt.Sprintf("%s/openshift-release-dev/ocp-release-nightly", m.env.ACRDomain()),
					},
				},
				{
					Source: "quay.io/openshift-release-dev/ocp-v4.0-art-dev",
					Mirrors: []string{
						fmt.Sprintf("%s/openshift-release-dev/ocp-v4.0-art-dev", m.env.ACRDomain()),
					},
				},
			},
			Publish: types.ExternalPublishingStrategy,
		},
		Azure: icazure.NewMetadataWithCredentials(
			azuretypes.CloudEnvironment(m.env.Environment().Name),
			m.env.Environment().ResourceManagerEndpoint,
			&icazure.Credentials{
				TenantID:       m.subscriptionDoc.Subscription.Properties.TenantID,
				ClientID:       m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientID,
				ClientSecret:   string(m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret),
				SubscriptionID: r.SubscriptionID,
			},
		),
	}

	if m.doc.OpenShiftCluster.Properties.IngressProfiles[0].Visibility == api.VisibilityPrivate {
		installConfig.Config.Publish = types.InternalPublishingStrategy
	}

	installConfig.Config.Azure.Image, err = rhcos.Image(ctx)
	if err != nil {
		return nil, nil, err
	}

	image := &releaseimage.Image{}
	if m.doc.OpenShiftCluster.Properties.ClusterProfile.Version == version.InstallStream.Version.String() {
		image.PullSpec = version.InstallStream.PullSpec
	} else {
		return nil, nil, fmt.Errorf("unimplemented version %q", m.doc.OpenShiftCluster.Properties.ClusterProfile.Version)
	}

	err = validation.ValidateInstallConfig(installConfig.Config).ToAggregate()
	if err != nil {
		return nil, nil, err
	}

	return installConfig, image, err
}
