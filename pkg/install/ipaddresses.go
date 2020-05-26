package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Azure/go-autorest/autorest/to"

	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/password"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (i *Installer) updateRouterIP(ctx context.Context) error {
	g, err := i.loadGraph(ctx)
	if err != nil {
		return err
	}

	installConfig := g[reflect.TypeOf(&installconfig.InstallConfig{})].(*installconfig.InstallConfig)
	kubeadminPassword := g[reflect.TypeOf(&password.KubeadminPassword{})].(*password.KubeadminPassword)

	svc, err := i.kubernetescli.CoreV1().Services("openshift-ingress").Get("router-default", metav1.GetOptions{})
	if err != nil {
		return err
	}

	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		return fmt.Errorf("routerIP not found")
	}

	routerIP := svc.Status.LoadBalancer.Ingress[0].IP

	err = i.dns.CreateOrUpdateRouter(ctx, i.doc.OpenShiftCluster, routerIP)
	if err != nil {
		return err
	}

	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.APIServerProfile.URL = "https://api." + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + ":6443/"
		doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = routerIP
		doc.OpenShiftCluster.Properties.ConsoleProfile.URL = "https://console-openshift-console.apps." + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/"
		doc.OpenShiftCluster.Properties.KubeadminPassword = api.SecureString(kubeadminPassword.Password)
		return nil
	})
	return err
}

func (i *Installer) updateAPIIP(ctx context.Context) error {
	infraID := i.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro" // TODO: remove after deploy
	}

	resourceGroup := stringutils.LastTokenByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	var ipAddress string
	if i.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		ip, err := i.publicipaddresses.Get(ctx, resourceGroup, infraID+"-pip-v4", "")
		if err != nil {
			return err
		}
		ipAddress = *ip.IPAddress
	} else {
		lb, err := i.loadbalancers.Get(ctx, resourceGroup, infraID+"-internal-lb", "")
		if err != nil {
			return err
		}
		ipAddress = *((*lb.FrontendIPConfigurations)[0].PrivateIPAddress)
	}

	err := i.dns.Update(ctx, i.doc.OpenShiftCluster, ipAddress)
	if err != nil {
		return err
	}

	privateEndpointIP, err := i.privateendpoint.GetRPPEIP(ctx, i.doc)
	if err != nil {
		return err
	}

	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.NetworkProfile.PrivateEndpointIP = privateEndpointIP
		doc.OpenShiftCluster.Properties.APIServerProfile.IP = ipAddress
		return nil
	})
	return err
}

func (i *Installer) createRPPrivateEndpoint(ctx context.Context) error {
	return i.privateendpoint.CreateRPPrivateEndpoint(ctx, i.doc)
}

func (i *Installer) createACRPrivateEndpoint(ctx context.Context) error {
	return i.privateendpoint.CreateACRPrivateEndpoint(ctx, i.doc)
}

func (i *Installer) updateACRIP(ctx context.Context) error {
	infraID := i.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro" // TODO: remove after deploy
	}

	resourceGroup := stringutils.LastTokenByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	ipAddress, err := i.privateendpoint.GetACRPEIP(ctx, i.doc)
	if err != nil {
		return err
	}

	_, err = i.dnscli.CreateOrUpdate(ctx, resourceGroup, "privatelink.azurecr.io", "", mgmtdns.A, mgmtdns.RecordSet{
		Name: to.StringPtr("acrsvc"),
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			ARecords: &[]mgmtdns.ARecord{
				{
					Ipv4Address: to.StringPtr(ipAddress),
				},
			},
		},
	}, "", "")
	if err != nil {
		return err
	}

	_, err = i.dnscli.CreateOrUpdate(ctx, resourceGroup, "privatelink.azurecr.io", "", mgmtdns.A, mgmtdns.RecordSet{
		Name: to.StringPtr(fmt.Sprint("arosvc.%s.data", i.doc.OpenShiftCluster.Location)),
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			ARecords: &[]mgmtdns.ARecord{
				{
					Ipv4Address: to.StringPtr(ipAddress),
				},
			},
		},
	}, "", "")
	if err != nil {
		return err
	}

	privateEndpointIP, err := i.privateendpoint.GetRPPEIP(ctx, i.doc)
	if err != nil {
		return err
	}

	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.NetworkProfile.PrivateEndpointIP = privateEndpointIP
		doc.OpenShiftCluster.Properties.APIServerProfile.IP = ipAddress
		return nil
	})
	return err
}
