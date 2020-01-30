package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/install/template"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (i *Installer) deploy(ctx context.Context, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds, image *releaseimage.Image) error {
	err := i.dns.Create(ctx, i.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	g, err := NewGraph(ctx, installConfig, platformCreds, image)
	if err != nil {
		return err
	}

	resourceGroup := i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID[strings.LastIndexByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')+1:]
	err = i.createResourceGroup(ctx, installConfig.Config.Azure.Region, resourceGroup)
	if err != nil {
		return err
	}

	storageTemplate := template.NewStorageTemplate(i.log, i.env.SubscriptionID(), installConfig.Config.Azure.Region, resourceGroup, i.deployments, i.doc.OpenShiftCluster.Properties.StorageSuffix)
	err = storageTemplate.Deploy(ctx)
	if err != nil {
		return err
	}

	blobService, err := i.getBlobService(ctx)
	if err != nil {
		return err
	}
	err = g.Store(blobService)
	if err != nil {
		return err
	}

	err = i.attachNSGToSubnets(ctx)
	if err != nil {
		return err
	}

	adminClient := g.GetMap()[reflect.TypeOf(&kubeconfig.AdminClient{})].(*kubeconfig.AdminClient)
	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		// used for the SAS token with which the bootstrap node retrieves its
		// ignition payload
		doc.OpenShiftCluster.Properties.Install.Now = time.Now().UTC()
		doc.OpenShiftCluster.Properties.AdminKubeconfig = adminClient.File.Data
		return nil
	})

	lbIP, err := i.getLBIP(ctx)
	if err != nil {
		return err
	}

	objectID, err := getServicePrincipalsIDBySPP(ctx, &i.doc.OpenShiftCluster.Properties.ServicePrincipalProfile)
	if err != nil {
		return err
	}

	clusterTemplate, err := template.NewClusterTemplate(i.log, i.env.SubscriptionID(), i.deployments, i.doc.OpenShiftCluster, installConfig.Config, lbIP, objectID, g.GetMap()[reflect.TypeOf(&machine.Master{})].(*machine.Master))
	if err != nil {
		return err
	}

	err = clusterTemplate.Deploy(ctx)
	if err != nil {
		return err
	}

	err = i.updateIPAdresses(ctx, resourceGroup, lbIP)
	if err != nil {
		return err
	}

	return i.waitForBootstrapConfigmap(ctx)
}

func (i *Installer) attachNSGToSubnets(ctx context.Context) error {
	for _, subnetID := range []string{
		i.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID,
		i.doc.OpenShiftCluster.Properties.WorkerProfiles[0].SubnetID,
	} {
		i.log.Printf("attaching network security group to subnet %s", subnetID)

		// TODO: there is probably an undesirable race condition here - check if etags can help.
		s, err := i.subnet.Get(ctx, subnetID)
		if err != nil {
			return err
		}

		if s.SubnetPropertiesFormat == nil {
			s.SubnetPropertiesFormat = &mgmtnetwork.SubnetPropertiesFormat{}
		}

		nsgID, err := subnet.NetworkSecurityGroupID(i.doc.OpenShiftCluster, subnetID)
		if err != nil {
			return err
		}

		if s.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
			if strings.EqualFold(*s.SubnetPropertiesFormat.NetworkSecurityGroup.ID, nsgID) {
				continue
			}

			return fmt.Errorf("tried to overwrite non-nil network security group")
		}

		s.SubnetPropertiesFormat.NetworkSecurityGroup = &mgmtnetwork.SecurityGroup{
			ID: to.StringPtr(nsgID),
		}

		err = i.subnet.CreateOrUpdate(ctx, subnetID, s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Installer) createResourceGroup(ctx context.Context, region, resourceGroup string) error {
	i.log.Print("creating resource group")
	group := mgmtresources.Group{
		Location:  &region,
		ManagedBy: to.StringPtr(i.doc.OpenShiftCluster.ID),
	}
	if _, ok := i.env.(env.Dev); ok {
		group.ManagedBy = nil
	}
	_, err := i.groups.CreateOrUpdate(ctx, resourceGroup, group)
	if err != nil {
		return err
	}

	if development, ok := i.env.(env.Dev); ok {
		err = development.CreateARMResourceGroupRoleAssignment(ctx, i.fpAuthorizer, resourceGroup)
		if err != nil {
			return err
		}
	}
	return nil
}

func getServicePrincipalsIDBySPP(ctx context.Context, spp *api.ServicePrincipalProfile) (string, error) {
	conf := auth.NewClientCredentialsConfig(spp.ClientID, spp.ClientSecret, spp.TenantID)
	conf.Resource = azure.PublicCloud.GraphEndpoint

	spGraphAuthorizer, err := conf.Authorizer()
	if err != nil {
		return "", err
	}

	applications := graphrbac.NewApplicationsClient(spp.TenantID, spGraphAuthorizer)

	res, err := applications.GetServicePrincipalsIDByAppID(ctx, spp.ClientID)
	if err != nil {
		return "", err
	}

	return *res.Value, nil
}

func (i *Installer) getLBIP(ctx context.Context) (string, error) {
	masterSubnet, err := i.subnet.Get(ctx, i.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
	if err != nil {
		return "", err
	}

	_, masterSubnetCIDR, err := net.ParseCIDR(*masterSubnet.AddressPrefix)
	if err != nil {
		return "", err
	}

	var lbIP net.IP
	{
		_, last := cidr.AddressRange(masterSubnetCIDR)
		lbIP = cidr.Dec(cidr.Dec(last))
	}
	return lbIP.String(), nil
}

func (i *Installer) updateIPAdresses(ctx context.Context, lbIP, resourceGroup string) error {
	i.log.Print("creating private endpoint")
	err := i.privateendpoint.Create(ctx, i.doc)
	if err != nil {
		return err
	}

	ipAddress := lbIP

	if i.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		ip, err := i.publicipaddresses.Get(ctx, resourceGroup, "aro-pip", "")
		if err != nil {
			return err
		}

		ipAddress = *ip.IPAddress
	}

	err = i.dns.Update(ctx, i.doc.OpenShiftCluster, ipAddress)
	if err != nil {
		return err
	}

	privateEndpointIP, err := i.privateendpoint.GetIP(ctx, i.doc)
	if err != nil {
		return err
	}

	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.NetworkProfile.PrivateEndpointIP = privateEndpointIP
		doc.OpenShiftCluster.Properties.APIServerProfile.IP = ipAddress
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (i *Installer) waitForBootstrapConfigmap(ctx context.Context) error {
	restConfig, err := restconfig.RestConfig(ctx, i.env, i.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	cli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.log.Print("waiting for bootstrap configmap")
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		cm, err := cli.CoreV1().ConfigMaps("kube-system").Get("bootstrap", metav1.GetOptions{})
		return err == nil && cm.Data["status"] == "complete", nil

	}, timeoutCtx.Done())
}
