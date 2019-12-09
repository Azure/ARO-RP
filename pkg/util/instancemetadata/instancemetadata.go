package instancemetadata

type InstanceMetadata interface {
	TenantID() string
	SubscriptionID() string
	Location() string
	ResourceGroup() string
}

type instanceMetadata struct {
	tenantID       string
	subscriptionID string
	location       string
	resourceGroup  string
}

func (im *instanceMetadata) TenantID() string {
	return im.tenantID
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
