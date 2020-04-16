package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
)

// ResourceGroup is an azure resource group
type ResourceGroup struct {
	// in advanced network case, the network resource group deletion is dependent on AKS client's status. If this is nil,
	// then it means this resource group doesn't have any dependency, otherwise, we should delete it after AKS cluster is clear.
	cluster      *ClusterInfo
	subscription *Subscription
	sdkGroup     *resources.Group
}
