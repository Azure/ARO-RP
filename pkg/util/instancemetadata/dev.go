package instancemetadata

import (
	"os"
)

func NewDev() InstanceMetadata {
	return &instanceMetadata{
		tenantID:       os.Getenv("AZURE_TENANT_ID"),
		subscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		location:       os.Getenv("AZURE_LOCATION"),
		resourceGroup:  os.Getenv("RESOURCEGROUP"),
	}
}
