package backend

import (
	"context"
	"encoding/base64"
	"os"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/installconfig"
	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/ipnet"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/azure"
	openstackvalidation "github.com/openshift/installer/pkg/types/openstack/validation"
	"github.com/openshift/installer/pkg/types/validation"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/install"
)

func (b *backend) install(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument) error {
	sshkey, err := ssh.NewPublicKey(&doc.OpenShiftCluster.Properties.SSHKey.PublicKey)
	if err != nil {
		return err
	}

	platformCreds := &installconfig.PlatformCreds{
		Azure: &icazure.Credentials{
			TenantID:       os.Getenv("AZURE_TENANT_ID"),
			ClientID:       doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientID,
			ClientSecret:   doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret,
			SubscriptionID: doc.SubscriptionID,
		},
		Passthrough: true, // TODO: not working yet
	}

	installConfig := &installconfig.InstallConfig{
		Config: &types.InstallConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: doc.OpenShiftCluster.Name,
			},
			SSHKey:     sshkey.Type() + " " + base64.StdEncoding.EncodeToString(sshkey.Marshal()),
			BaseDomain: b.domain,
			Networking: &types.Networking{
				MachineCIDR: ipnet.MustParseCIDR(doc.OpenShiftCluster.Properties.NetworkProfile.VNetCIDR),
				NetworkType: "OpenShiftSDN",
				ClusterNetwork: []types.ClusterNetworkEntry{
					{
						CIDR:       *ipnet.MustParseCIDR(doc.OpenShiftCluster.Properties.NetworkProfile.PodCIDR),
						HostPrefix: 23,
					},
				},
				ServiceNetwork: []ipnet.IPNet{
					*ipnet.MustParseCIDR(doc.OpenShiftCluster.Properties.NetworkProfile.ServiceCIDR),
				},
			},
			ControlPlane: &types.MachinePool{
				Name:     "master",
				Replicas: to.Int64Ptr(3),
				Platform: types.MachinePoolPlatform{
					Azure: &azure.MachinePool{
						InstanceType: string(doc.OpenShiftCluster.Properties.MasterProfile.VMSize),
					},
				},
				Hyperthreading: "Enabled",
			},
			Platform: types.Platform{
				Azure: &azure.Platform{
					Region:                      doc.OpenShiftCluster.Location,
					ResourceGroup:               doc.OpenShiftCluster.Properties.ResourceGroup,
					BaseDomainResourceGroupName: os.Getenv("RESOURCEGROUP"),
				},
			},
			PullSecret: string(os.Getenv("PULL_SECRET")),
		},
	}

	for _, wp := range doc.OpenShiftCluster.Properties.WorkerProfiles {
		installConfig.Config.Compute = append(installConfig.Config.Compute, types.MachinePool{
			Name:     wp.Name,
			Replicas: to.Int64Ptr(int64(wp.Count)),
			Platform: types.MachinePoolPlatform{
				Azure: &azure.MachinePool{
					InstanceType: string(wp.VMSize),
					OSDisk: azure.OSDisk{
						DiskSizeGB: int32(wp.DiskSizeGB),
					},
				},
			},
			Hyperthreading: "Enabled",
		})
	}

	err = validation.ValidateInstallConfig(installConfig.Config, openstackvalidation.NewValidValuesFetcher()).ToAggregate()
	if err != nil {
		return err
	}

	return install.NewInstaller(log, b.db, b.domain, b.authorizer, doc.SubscriptionID).Install(ctx, doc, installConfig, platformCreds)
}
