package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/apparentlymart/go-cidr/cidr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/installer"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) updateClusterData(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	pg, err := m.graph.LoadPersisted(ctx, resourceGroup, account)
	if err != nil {
		return err
	}

	var installConfig *installer.InstallConfig
	var kubeadminPassword *installer.KubeadminPasswordData
	err = pg.GetByName(false, "*password.KubeadminPassword", &kubeadminPassword)
	if err != nil {
		return err
	}
	err = pg.GetByName(false, "*installconfig.InstallConfig", &installConfig)
	if err != nil {
		return err
	}

	// See aro-installer/pkg/installer/generateinstallconfig.go
	domain := m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain
	if !strings.ContainsRune(domain, '.') {
		domain += "." + m.env.Domain()
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.APIServerProfile.URL = "https://api." + domain + ":6443/"
		doc.OpenShiftCluster.Properties.ConsoleProfile.URL = "https://console-openshift-console.apps." + domain + "/"
		doc.OpenShiftCluster.Properties.KubeadminPassword = api.SecureString(kubeadminPassword.Password)
		doc.OpenShiftCluster.Properties.NetworkProfile.SoftwareDefinedNetwork = api.SoftwareDefinedNetwork(installConfig.Config.NetworkType)
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

// createOrUpdateRouterIPEarly prepares IP address for the API server early
func (m *manager) createOrUpdateRouterIPEarly(ctx context.Context) error {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	var ipAddress string
	if m.doc.OpenShiftCluster.Properties.IngressProfiles[0].Visibility == api.VisibilityPublic {
		ip, err := m.armPublicIPAddresses.Get(ctx, resourceGroup, infraID+"-default-v4", nil)
		if err != nil {
			return err
		}
		ipAddress = *ip.Properties.IPAddress
	} else {
		// there's no way to reserve private IPs in Azure, so we pick the
		// highest free address in the subnet (i.e., there's a race here). Azure
		// specifically documents that dynamic allocation proceeds from the
		// bottom of the subnet, so there's a good chance that we'll get away
		// with this.
		// https://docs.microsoft.com/en-us/azure/virtual-network/private-ip-addresses#allocation-method
		var err error

		workerProfiles, _ := api.GetEnrichedWorkerProfiles(m.doc.OpenShiftCluster.Properties)
		workerSubnetId := workerProfiles[0].SubnetID

		r, err := arm.ParseResourceID(workerSubnetId)
		if err != nil {
			return err
		}
		subnet, err := m.armSubnets.Get(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, &armnetwork.SubnetsClientGetOptions{Expand: to.StringPtr("ipConfigurations")})
		if err != nil {
			return err
		}
		ipAddress, err = getHighestFreeIP(&subnet.Subnet)
		if err != nil {
			return err
		}
		if ipAddress == "" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", fmt.Sprintf("The subnet '%s' has no remaining IP addresses.", m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID))
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

// getHighestFreeIP retrieves the highest free private IP address in the given subnetID.
func getHighestFreeIP(subnet *armnetwork.Subnet) (string, error) {
	// grab the first addressPrefix in the subnet
	var (
		subnetCIDR *net.IPNet
		err        error
	)
	if subnet.Properties.AddressPrefix != nil {
		_, subnetCIDR, err = net.ParseCIDR(*subnet.Properties.AddressPrefix)
	} else if len(subnet.Properties.AddressPrefixes) > 0 {
		_, subnetCIDR, err = net.ParseCIDR(*subnet.Properties.AddressPrefixes[0])
	} else {
		// subnet must have at least one address prefix, so it shouldn't be called.
		return "", fmt.Errorf("addressPrefix is not found in the subnet")
	}

	if err != nil {
		return "", err
	}

	bottom, top := cidr.AddressRange(subnetCIDR)

	allocated := map[string]struct{}{}

	// first four addresses and the broadcast address are reserved:
	// https://docs.microsoft.com/en-us/azure/virtual-network/private-ip-addresses#allocation-method
	for i, ip := 0, bottom; i < 4 && !ip.Equal(top); i, ip = i+1, cidr.Inc(ip) {
		allocated[ip.String()] = struct{}{}
	}
	allocated[top.String()] = struct{}{}

	if subnet.Properties.IPConfigurations != nil {
		for _, ipconfig := range subnet.Properties.IPConfigurations {
			if ipconfig.Properties.PrivateIPAddress != nil {
				allocated[*ipconfig.Properties.PrivateIPAddress] = struct{}{}
			}
		}
	}

	for ip := top; !ip.Equal(cidr.Dec(bottom)); ip = cidr.Dec(ip) {
		if _, ok := allocated[ip.String()]; !ok {
			return ip.String(), nil
		}
	}

	return "", nil
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

// updateAPIIPEarly updates the `doc` with the public and private IP of the API server,
// and updates the DNS record of the API server according to the API server visibility.
// This function can only be called on create - not on update - because it
// refers to -pip-v4, which doesn't exist on pre-DNS change clusters.
func (m *manager) updateAPIIPEarly(ctx context.Context) error {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	lb, err := m.armLoadBalancers.Get(ctx, resourceGroup, infraID+"-internal", nil)
	if err != nil {
		return err
	}
	intIPAddress := *lb.Properties.FrontendIPConfigurations[0].Properties.PrivateIPAddress

	ipAddress := intIPAddress
	if m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		ip, err := m.armPublicIPAddresses.Get(ctx, resourceGroup, infraID+"-pip-v4", nil)
		if err != nil {
			return err
		}
		ipAddress = *ip.Properties.IPAddress
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

	pe, err := m.armPrivateEndpoints.Get(ctx, resourceGroup, infraID+"-pe", &armnetwork.PrivateEndpointsClientGetOptions{Expand: pointerutils.ToPtr("networkInterfaces")})
	if err != nil {
		return err
	}

	pls, err := m.armRPPrivateLinkServices.Get(ctx, m.env.GatewayResourceGroup(), "gateway-pls-001", nil)
	if err != nil {
		return err
	}

	// this is O(N), which is not great, but this is only called once per
	// cluster, and N < 1000.  The portal handles this by making a kusto-style
	// call to the resource graph service, but it's not worth the effort to do
	// that here.
	var linkIdentifier string
	for _, conn := range pls.Properties.PrivateEndpointConnections {
		if !strings.EqualFold(*conn.Properties.PrivateEndpoint.ID, *pe.ID) {
			continue
		}

		linkIdentifier = *conn.Properties.LinkIdentifier

		if !strings.EqualFold(*conn.Properties.PrivateLinkServiceConnectionState.Status, "Approved") {
			conn.Properties.PrivateLinkServiceConnectionState.Status = to.StringPtr("Approved")
			conn.Properties.PrivateLinkServiceConnectionState.Description = to.StringPtr("Approved")

			_, err = m.armRPPrivateLinkServices.UpdatePrivateEndpointConnection(ctx, m.env.GatewayResourceGroup(), "gateway-pls-001", *conn.Name, *conn, nil)
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
		if !strings.EqualFold(gwyDoc.Gateway.ID, m.doc.OpenShiftCluster.ID) ||
			!strings.EqualFold(gwyDoc.ID, m.doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateLinkID) ||
			!strings.EqualFold(gwyDoc.Gateway.ImageRegistryStorageAccountName, m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName) ||
			!strings.EqualFold(gwyDoc.Gateway.StorageSuffix, m.doc.OpenShiftCluster.Properties.StorageSuffix) {
			return fmt.Errorf("gateway record '%s' already exists for a different cluster '%s'", linkIdentifier, gwyDoc.Gateway.ID)
		}
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateEndpointIP = *pe.Properties.NetworkInterfaces[0].Properties.IPConfigurations[0].Properties.PrivateIPAddress
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

	err := m.armFPPrivateEndpoints.CreateOrUpdateAndWait(ctx, m.env.ResourceGroup(), env.RPPrivateEndpointPrefix+m.doc.ID, armnetwork.PrivateEndpoint{
		Properties: &armnetwork.PrivateEndpointProperties{
			Subnet: &armnetwork.Subnet{
				// TODO: in the future we will need multiple vnets for our PEs.
				// It will be necessary to decide the vnet for a cluster's PE
				// somewhere around here.
				ID: to.StringPtr("/subscriptions/" + m.env.SubscriptionID() + "/resourceGroups/" + m.env.ResourceGroup() + "/providers/Microsoft.Network/virtualNetworks/rp-pe-vnet-001/subnets/rp-pe-subnet"),
			},
			ManualPrivateLinkServiceConnections: []*armnetwork.PrivateLinkServiceConnection{
				{
					Name: to.StringPtr("rp-plsconnection"),
					Properties: &armnetwork.PrivateLinkServiceConnectionProperties{
						PrivateLinkServiceID: to.StringPtr(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID + "/providers/Microsoft.Network/privateLinkServices/" + infraID + "-pls"),
					},
				},
			},
		},
		Location: &m.doc.OpenShiftCluster.Location,
	}, nil)
	if err != nil {
		return err
	}

	pe, err := m.armFPPrivateEndpoints.Get(ctx, m.env.ResourceGroup(), env.RPPrivateEndpointPrefix+m.doc.ID, &armnetwork.PrivateEndpointsClientGetOptions{Expand: to.StringPtr("networkInterfaces")})
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.NetworkProfile.APIServerPrivateEndpointIP = *pe.Properties.NetworkInterfaces[0].Properties.IPConfigurations[0].Properties.PrivateIPAddress
		return nil
	})
	return err
}
