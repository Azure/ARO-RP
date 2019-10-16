package backend

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"os"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/installconfig"
	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/ipnet"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/azure"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/deploy"
)

func (b *backend) update(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument) error {
	if doc.OpenShiftCluster.Properties.Installation == nil {
		return nil
	}

	doc, err := b.db.Patch(doc.OpenShiftCluster.ID, func(doc *api.OpenShiftClusterDocument) error {
		if doc.OpenShiftCluster.Properties.SSHKey == nil {
			var err error
			doc.OpenShiftCluster.Properties.SSHKey, err = rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	sshkey, err := ssh.NewPublicKey(&doc.OpenShiftCluster.Properties.SSHKey.PublicKey)
	if err != nil {
		return err
	}

	platformCreds := &installconfig.PlatformCreds{
		Azure: &icazure.Credentials{
			TenantID:       os.Getenv("AZURE_TENANT_ID"),
			ClientID:       os.Getenv("AZURE_CLIENT_ID"),
			ClientSecret:   os.Getenv("AZURE_CLIENT_SECRET"),
			SubscriptionID: doc.SubscriptionID,
		},
	}

	installConfig := &installconfig.InstallConfig{
		Config: &types.InstallConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: doc.OpenShiftCluster.Name,
			},
			SSHKey:     sshkey.Type() + " " + base64.StdEncoding.EncodeToString(sshkey.Marshal()),
			BaseDomain: os.Getenv("DOMAIN"),
			Networking: &types.Networking{
				MachineCIDR: ipnet.MustParseCIDR(doc.OpenShiftCluster.Properties.NetworkProfile.VNetCIDR),
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
			},
			Platform: types.Platform{
				Azure: &azure.Platform{
					Region:                      doc.OpenShiftCluster.Location,
					BaseDomainResourceGroupName: os.Getenv("DOMAIN_RESOURCEGROUP"),
				},
			},
			PullSecret: string(doc.OpenShiftCluster.Properties.PullSecret),
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
		})
	}

	return deploy.NewDeployer(log, b.db, b.authorizer, doc.SubscriptionID).Deploy(ctx, doc, installConfig, platformCreds)
}
