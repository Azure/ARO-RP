package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/password"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) updateClusterData(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	pg, err := m.graph.LoadPersisted(ctx, resourceGroup, account)
	if err != nil {
		return err
	}

	var installConfig *installconfig.InstallConfig
	var kubeadminPassword *password.KubeadminPassword
	err = pg.Get(false, &installConfig, &kubeadminPassword)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.APIServerProfile.URL = "https://api." + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + ":6443/"
		doc.OpenShiftCluster.Properties.ConsoleProfile.URL = "https://console-openshift-console.apps." + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/"
		doc.OpenShiftCluster.Properties.KubeadminPassword = api.SecureString(kubeadminPassword.Password)
		return nil
	})
	return err
}

func (m *manager) createOrUpdateRouterIPFromCluster(ctx context.Context) error {
	if !m.isIngressProfileAvailable() {
		m.log.Error("skip createOrUpdateRouterIPFromCluster")
		return nil
	}

	svc, err := m.kubernetescli.CoreV1().Services("openshift-ingress").Get(ctx, "router-default", metav1.GetOptions{})
	// default ingress must be present in the cluster
	if err != nil {
		return err
	}

	// This must be present always. If not - we have an issue
	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		return fmt.Errorf("routerIP not found")
	}

	ipAddress := svc.Status.LoadBalancer.Ingress[0].IP

	err = m.dns.CreateOrUpdateRouter(ctx, m.doc.OpenShiftCluster, ipAddress)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = ipAddress
		return nil
	})
	return err
}

func (m *manager) createOrUpdateRouterIPEarly(ctx context.Context) error {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	var ipAddress string
	if m.doc.OpenShiftCluster.Properties.IngressProfiles[0].Visibility == api.VisibilityPublic {
		ip, err := m.publicIPAddresses.Get(ctx, resourceGroup, infraID+"-default-v4", "")
		if err != nil {
			return err
		}
		ipAddress = *ip.IPAddress
	} else {
		// there's no way to reserve private IPs in Azure, so we pick the
		// highest free address in the subnet (i.e., there's a race here). Azure
		// specifically documents that dynamic allocation proceeds from the
		// bottom of the subnet, so there's a good chance that we'll get away
		// with this.
		// https://docs.microsoft.com/en-us/azure/virtual-network/private-ip-addresses#allocation-method
		var err error
		ipAddress, err = m.subnet.GetHighestFreeIP(ctx, m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].SubnetID)
		if err != nil {
			return err
		}
		if ipAddress == "" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The subnet '%s' has no remaining IP addresses.", m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
		}
	}

	err := m.dns.CreateOrUpdateRouter(ctx, m.doc.OpenShiftCluster, ipAddress)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = ipAddress
		return nil
	})
	return err
}

func (m *manager) populateDatabaseIntIP(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.APIServerProfile.IntIP != "" {
		return nil
	}
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	var lbName string
	switch m.doc.OpenShiftCluster.Properties.ArchitectureVersion {
	case api.ArchitectureVersionV1:
		lbName = infraID + "-internal-lb"
	case api.ArchitectureVersionV2:
		lbName = infraID + "-internal"
	default:
		return fmt.Errorf("unknown architecture version %d", m.doc.OpenShiftCluster.Properties.ArchitectureVersion)
	}

	lb, err := m.loadBalancers.Get(ctx, resourceGroup, lbName, "")
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.APIServerProfile.IntIP = *((*lb.FrontendIPConfigurations)[0].PrivateIPAddress)
		return nil
	})
	return err
}

// this function can only be called on create - not on update - because it
// refers to -pip-v4, which doesn't exist on pre-DNS change clusters.
func (m *manager) updateAPIIPEarly(ctx context.Context) error {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	lb, err := m.loadBalancers.Get(ctx, resourceGroup, infraID+"-internal", "")
	if err != nil {
		return err
	}
	intIPAddress := *((*lb.FrontendIPConfigurations)[0].PrivateIPAddress)

	ipAddress := intIPAddress
	if m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		ip, err := m.publicIPAddresses.Get(ctx, resourceGroup, infraID+"-pip-v4", "")
		if err != nil {
			return err
		}
		ipAddress = *ip.IPAddress
	}

	err = m.dns.Update(ctx, m.doc.OpenShiftCluster, ipAddress)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.APIServerProfile.IP = ipAddress
		doc.OpenShiftCluster.Properties.APIServerProfile.IntIP = intIPAddress
		return nil
	})
	return err
}

// ensureGatewayCreate approves the gateway PE/PLS connection, creates the
// gateway database record and updates the model with the private endpoint IP.
func (m *manager) ensureGatewayCreate(ctx context.Context) error {
	if !m.doc.OpenShiftCluster.Properties.FeatureProfile.GatewayEnabled ||
		m.doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateEndpointIP != "" {
		return nil
	}

	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	pe, err := m.privateEndpoints.Get(ctx, resourceGroup, infraID+"-pe", "networkInterfaces")
	if err != nil {
		return err
	}

	pls, err := m.rpPrivateLinkServices.Get(ctx, m.env.GatewayResourceGroup(), "gateway-pls-001", "")
	if err != nil {
		return err
	}

	// this is O(N), which is not great, but this is only called once per
	// cluster, and N < 1000.  The portal handles this by making a kusto-style
	// call to the resource graph service, but it's not worth the effort to do
	// that here.
	var linkIdentifier string
	for _, conn := range *pls.PrivateEndpointConnections {
		if !strings.EqualFold(*conn.PrivateEndpoint.ID, *pe.ID) {
			continue
		}

		linkIdentifier = *conn.LinkIdentifier

		if !strings.EqualFold(*conn.PrivateLinkServiceConnectionState.Status, "Approved") {
			conn.PrivateLinkServiceConnectionState.Status = to.StringPtr("Approved")
			conn.PrivateLinkServiceConnectionState.Description = to.StringPtr("Approved")

			_, err = m.rpPrivateLinkServices.UpdatePrivateEndpointConnection(ctx, m.env.GatewayResourceGroup(), "gateway-pls-001", *conn.Name, conn)
			if err != nil {
				return err
			}
		}

		break
	}

	if linkIdentifier == "" {
		return errors.New("private endpoint connection not found")
	}

	_, err = m.dbGateway.Create(ctx, &api.GatewayDocument{
		ID: linkIdentifier,
		Gateway: &api.Gateway{
			ID:                              m.doc.OpenShiftCluster.ID,
			StorageSuffix:                   m.doc.OpenShiftCluster.Properties.StorageSuffix,
			ImageRegistryStorageAccountName: m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName,
		},
	})

	recordExists := err != nil && cosmosdb.IsErrorStatusCode(err, http.StatusConflict)
	if err != nil && !recordExists /* already exists */ {
		return err
	}

	// ensure the record is this clusters if it exists
	if recordExists {
		gwyDoc, err := m.dbGateway.Get(ctx, linkIdentifier)
		if err != nil {
			return err
		}
		if !strings.EqualFold(gwyDoc.ID, m.doc.OpenShiftCluster.ID) ||
			!strings.EqualFold(gwyDoc.Gateway.ImageRegistryStorageAccountName, m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName) ||
			!strings.EqualFold(gwyDoc.Gateway.StorageSuffix, m.doc.OpenShiftCluster.Properties.StorageSuffix) {
			return errors.New("gateway record already exists for a different cluster")
		}
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateEndpointIP = *(*(*pe.PrivateEndpointProperties.NetworkInterfaces)[0].IPConfigurations)[0].PrivateIPAddress
		doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateLinkID = linkIdentifier
		return nil
	})
	return err
}

func (m *manager) createAPIServerPrivateEndpoint(ctx context.Context) error {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	err := m.fpPrivateEndpoints.CreateOrUpdateAndWait(ctx, m.env.ResourceGroup(), env.RPPrivateEndpointPrefix+m.doc.ID, mgmtnetwork.PrivateEndpoint{
		PrivateEndpointProperties: &mgmtnetwork.PrivateEndpointProperties{
			Subnet: &mgmtnetwork.Subnet{
				// TODO: in the future we will need multiple vnets for our PEs.
				// It will be necessary to decide the vnet for a cluster's PE
				// somewhere around here.
				ID: to.StringPtr("/subscriptions/" + m.env.SubscriptionID() + "/resourceGroups/" + m.env.ResourceGroup() + "/providers/Microsoft.Network/virtualNetworks/rp-pe-vnet-001/subnets/rp-pe-subnet"),
			},
			ManualPrivateLinkServiceConnections: &[]mgmtnetwork.PrivateLinkServiceConnection{
				{
					Name: to.StringPtr("rp-plsconnection"),
					PrivateLinkServiceConnectionProperties: &mgmtnetwork.PrivateLinkServiceConnectionProperties{
						PrivateLinkServiceID: to.StringPtr(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID + "/providers/Microsoft.Network/privateLinkServices/" + infraID + "-pls"),
					},
				},
			},
		},
		Location: &m.doc.OpenShiftCluster.Location,
	})
	if err != nil {
		return err
	}

	pe, err := m.fpPrivateEndpoints.Get(ctx, m.env.ResourceGroup(), env.RPPrivateEndpointPrefix+m.doc.ID, "networkInterfaces")
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.NetworkProfile.APIServerPrivateEndpointIP = *(*(*pe.PrivateEndpointProperties.NetworkInterfaces)[0].IPConfigurations)[0].PrivateIPAddress
		return nil
	})
	return err
}
