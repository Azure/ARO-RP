package clients

import (
	"encoding/json"

	"github.com/zhuoli/ARO-RP/test/e2e/designs"
	"github.com/zhuoli/ARO-RP/test/e2e/objects/azure"
)

// AzureClient represents a user interface to Azure e.g. the `az` cli
type AzureClient interface {
	ResourceGroup() ResourceGroupClient
}

// AzureRestClient ...
type AzureRestClient interface {
	PutSubscription(tenantID, subID string, features []string) error
	ValidatePreflight(subID, rgName, apiVersion string, resources []json.RawMessage) error
}

// ResourceGroupClient is the logical stand in of `az group`
type ResourceGroupClient interface {
	Create(designs.ResourceGroup) (*azure.ResourceGroup, error)
	Get(name string) (*azure.ResourceGroup, error)
	Exists(name string) bool
	Delete(*azure.ResourceGroup) error
	SetTag(rg *azure.ResourceGroup, key, value string) error
	SetTags(rg *azure.ResourceGroup, tags map[string]*string) error
	Validate(name string) bool
}
