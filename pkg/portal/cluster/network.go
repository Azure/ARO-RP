package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strconv"

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
	ClusterNetwork []ClusterNetwork `json:"clusternetworks"`
}

type NetworkInformation struct {
	ClusterNetworkList ClusterNetworkList `json: "clusternetworklist"`
}

func NetworkData(clusterNetworks *networkv1.ClusterNetworkList) *NetworkInformation {
	final := &NetworkInformation{
		ClusterNetworkList: getClusterNetworkList(clusterNetworks),
	}
	fmt.Println()
	fmt.Println(final)
	fmt.Println()
	return final
}

// helper functions
func getClusterNetworkList(clusterNetworks *networkv1.ClusterNetworkList) ClusterNetworkList {
	final := ClusterNetworkList{
		ClusterNetwork: make([]ClusterNetwork, len(clusterNetworks.Items)),
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
		final.ClusterNetwork[i] = clusterNetwork
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
	MTU := "Unknown"
	if clusterNetwork.VXLANPort != nil {
		MTU = strconv.FormatUint(uint64(*clusterNetwork.VXLANPort), 10)
	}
	return MTU
}

func (f *realFetcher) Network(ctx context.Context) (*NetworkInformation, error) {
	r, err := f.networkClient.NetworkV1().ClusterNetworks().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return NetworkData(r), nil
}

func (c *client) Network(ctx context.Context) (*NetworkInformation, error) {
	return c.fetcher.Network(ctx)
}
