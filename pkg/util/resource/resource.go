package resource

import (
	"github.com/Azure/go-autorest/autorest/azure"
)

// ResourceName returns the resource name of a resource ID
func ResourceName(resourceID string) (string, error) {
	r, err := azure.ParseResourceID(resourceID)
	return r.ResourceName, err
}

// SubscriptionID returns the subscription ID of a resource ID
func SubscriptionID(resourceID string) (string, error) {
	r, err := azure.ParseResourceID(resourceID)
	return r.SubscriptionID, err
}
