package instancemetadata

type InstanceMetadata interface {
	SubscriptionID() string
	Location() string
	ResourceGroup() string
}

type instanceMetadata struct {
	subscriptionID string
	location       string
	resourceGroup  string
}

func (im *instanceMetadata) SubscriptionID() string {
	return im.subscriptionID
}

func (im *instanceMetadata) Location() string {
	return im.location
}

func (im *instanceMetadata) ResourceGroup() string {
	return im.resourceGroup
}
