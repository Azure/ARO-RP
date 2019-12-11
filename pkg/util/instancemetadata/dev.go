package instancemetadata

import (
	"os"
)

func NewDev() InstanceMetadata {
	return &instanceMetadata{
		subscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		location:       os.Getenv("LOCATION"),
		resourceGroup:  os.Getenv("RESOURCEGROUP"),
	}
}
