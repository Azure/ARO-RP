package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	networkv1 "github.com/openshift/api/network/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterNetworkEntry struct {
	CIDR             string `json:"cidr"`
	HostSubnetLength string `json:"hostsubnetlength"`
}

type ClusterNetwork struct {
	Name                  string                `json:"name"`
	PluginName            string                `json:"pluginname"`
	NetworkCIDR           string                `json:"networkcidr"`
	ServiceNetworkCIDR    string                `json:"servicenetworkcidr"`
	HostSubnetLength      string                `json:"hostsubnetlength"`
	MTU                   string                `json:"mtu"`
	VXLANPort             string                `json:"vxlanport"`
	ClusterNetworkEntries []ClusterNetworkEntry `json:"clusternetworkentry"`
}

type ClusterNetworkList struct {
	ClusterNetworks []ClusterNetwork `json:"clusternetworks"`
}

type VNetPeering struct {
	Name         string `json:"name"`
	RemoteVNet   string `json:"remotevnet"`
	State        string `json:"state"`
	Provisioning string `json:"provisioning"`
}

type VNetPeeringList struct {
	VNetPeerings []VNetPeering `json:"vnetpeerings"`
}

type Subnet struct {
	Name          string `json:"name"`
	ID            string `json:"id"`
	AddressPrefix string `json:"addressprefix"`
	Provisioning  string `json:"provisioning"`
	RouteTable    string `json:"routetable"`
}

type SubnetList struct {
	Subnets []Subnet `json:"subnets"`
}

type IngressProfile struct {
	Name       string `json:"name"`
	IP         string `json:"ip"`
	Visibility string `json:"visibility"`
}

type IngressProfileList struct {
	IngressProfiles []IngressProfile `json:"ingressprofiles"`
}

type NetworkInformation struct {
	ClusterNetworkList ClusterNetworkList `json:"clusternetworklist"`
	VNetPeeringList    VNetPeeringList    `json:"vnetpeeringlist"`
	SubnetList         SubnetList         `json:"subnetlist"`
	IngressProfileList IngressProfileList `json:"ingressprofilelist"`
}

type ClusterDetails struct {
	Auth     autorest.Authorizer `json:"auth"`
	SubsId   string              `json:"subsID"`
	ResGrp   string              `json:"resgrp"`
	VNet     string              `json:"vnet"`
	ClusName string              `json:"clusname"`
}

func NetworkData(clusterNetworks *networkv1.ClusterNetworkList, doc *api.OpenShiftClusterDocument) *NetworkInformation {

	clusDet := getClusterDetails(doc)

	// Get all subnetids for getting the subnet details
	subnetIds := []string{doc.OpenShiftCluster.Properties.MasterProfile.SubnetID}
	for _, wp := range doc.OpenShiftCluster.Properties.WorkerProfiles {
		subnetIds = append(subnetIds, wp.SubnetID)
	}

	// Response of request for network information
	final := &NetworkInformation{
		ClusterNetworkList: getClusterNetworkList(clusterNetworks),
		VNetPeeringList:    getVNetPeeringList(clusDet),
		SubnetList:         getSubnetList(subnetIds, clusDet),
		IngressProfileList: getIngressProfileList(doc),
	}

	return final
}

// helper functions
func getClusterDetails(doc *api.OpenShiftClusterDocument) (envDetails ClusterDetails) {
	// Get an authorizer
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		fmt.Println("Error creating Azure authorizer:", err)
		return
	}

	clusterName := doc.OpenShiftCluster.Name
	resourceID := strings.Split(doc.OpenShiftCluster.Properties.MasterProfile.SubnetID, "/")

	envDet := ClusterDetails{
		Auth:     authorizer,
		SubsId:   resourceID[2],
		ResGrp:   resourceID[4],
		VNet:     resourceID[8],
		ClusName: clusterName,
	}
	return envDet
}

func getClusterNetworkList(clusterNetworks *networkv1.ClusterNetworkList) ClusterNetworkList {
	final := ClusterNetworkList{
		ClusterNetworks: make([]ClusterNetwork, len(clusterNetworks.Items)),
	}

	for i, clusNet := range clusterNetworks.Items {
		clusterNetwork := ClusterNetwork{
			Name:                  clusNet.Name,
			PluginName:            clusNet.PluginName,
			NetworkCIDR:           clusNet.Network,
			ServiceNetworkCIDR:    clusNet.ServiceNetwork,
			HostSubnetLength:      strconv.FormatUint(uint64(clusNet.HostSubnetLength), 10),
			MTU:                   getMTU(clusNet),
			VXLANPort:             getVXLANPort(clusNet),
			ClusterNetworkEntries: getClusterNetworkEntries(clusNet),
		}
		final.ClusterNetworks[i] = clusterNetwork
	}
	return final
}

func getVNetPeeringList(clusDet ClusterDetails) VNetPeeringList {

	// Create a new VirtualNetworkPeerings client
	vnetPeeringClient := mgmtnetwork.NewVirtualNetworkPeeringsClient(clusDet.SubsId)
	vnetPeeringClient.Authorizer = clusDet.Auth

	// Get a list of all the virtual network peerings in the specified virtual network
	vnetPeerings, err := vnetPeeringClient.List(context.Background(), clusDet.ResGrp, clusDet.VNet)
	if err != nil {
		fmt.Println("Error getiing Vnet Peerings:", err)
	}

	final := VNetPeeringList{
		VNetPeerings: make([]VNetPeering, len(vnetPeerings.Values())),
	}

	// Loop through the list of virtual network peerings and create final list
	for i, peering := range vnetPeerings.Values() {

		vnetPeering := VNetPeering{
			Name:         *peering.Name,
			RemoteVNet:   strings.Split(*peering.RemoteVirtualNetwork.ID, "/")[8],
			State:        string(peering.PeeringState),
			Provisioning: string(peering.VirtualNetworkPeeringPropertiesFormat.ProvisioningState),
		}
		final.VNetPeerings[i] = vnetPeering
	}

	return final
}

func getSubnetList(subnetIds []string, clusDet ClusterDetails) SubnetList {

	// create a new SubnetsClient
	subnetsClient := mgmtnetwork.NewSubnetsClient(clusDet.SubsId)
	subnetsClient.Authorizer = clusDet.Auth

	final := SubnetList{
		Subnets: make([]Subnet, len(subnetIds)),
	}

	for i, subnetID := range subnetIds {
		// get the subnet details
		subnet, err := subnetsClient.Get(context.Background(), clusDet.ResGrp, clusDet.VNet, strings.Split(subnetID, "/")[10], "")
		if err != nil {
			fmt.Println("Failed to get subnet details:", err)
		}

		subNet := Subnet{
			Name:          *subnet.Name,
			ID:            *subnet.ID,
			AddressPrefix: *subnet.AddressPrefix,
			Provisioning:  string(subnet.ProvisioningState),
			RouteTable:    strings.Split(*subnet.RouteTable.ID, "/")[8],
		}
		final.Subnets[i] = subNet
	}

	return final
}

func getIngressProfileList(doc *api.OpenShiftClusterDocument) IngressProfileList {

	final := IngressProfileList{
		IngressProfiles: make([]IngressProfile, len(doc.OpenShiftCluster.Properties.IngressProfiles)),
	}

	for i, ip := range doc.OpenShiftCluster.Properties.IngressProfiles {
		ingressProfile := IngressProfile{
			Name:       ip.Name,
			IP:         ip.IP,
			Visibility: string(ip.Visibility),
		}
		final.IngressProfiles[i] = ingressProfile
	}

	return final
}

func getClusterNetworkEntries(clusterNetwork networkv1.ClusterNetwork) []ClusterNetworkEntry {
	final := make([]ClusterNetworkEntry, len(clusterNetwork.ClusterNetworks))

	for i, clusNetEnt := range clusterNetwork.ClusterNetworks {
		clusterNetworkEntry := ClusterNetworkEntry{
			CIDR:             clusNetEnt.CIDR,
			HostSubnetLength: strconv.FormatUint(uint64(clusNetEnt.HostSubnetLength), 10),
		}
		final[i] = clusterNetworkEntry
	}

	return final
}

func getMTU(clusterNetwork networkv1.ClusterNetwork) string {
	MTU := "Unknown"
	if clusterNetwork.MTU != nil {
		MTU = strconv.FormatUint(uint64(*clusterNetwork.MTU), 10)
	}
	return MTU
}

func getVXLANPort(clusterNetwork networkv1.ClusterNetwork) string {
	VXLANPort := "Unknown"
	if clusterNetwork.VXLANPort != nil {
		VXLANPort = strconv.FormatUint(uint64(*clusterNetwork.VXLANPort), 10)
	}
	return VXLANPort
}

func (f *realFetcher) Network(ctx context.Context, doc *api.OpenShiftClusterDocument) (*NetworkInformation, error) {
	r1, err := f.networkClient.NetworkV1().ClusterNetworks().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return NetworkData(r1, doc), nil
}

func (c *client) Network(ctx context.Context, doc *api.OpenShiftClusterDocument) (*NetworkInformation, error) {
	return c.fetcher.Network(ctx, doc)
}
