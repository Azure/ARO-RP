package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
)

// GroupsClient is a minimal interface for azure Resources Client
type GroupsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, parameters resources.Group) (result resources.Group, err error)
	Get(ctx context.Context, resourceGroupName string) (result resources.Group, err error)
	GroupsClientAddons
}

type groupsClient struct {
	resources.GroupsClient
}

var _ GroupsClient = &groupsClient{}

// NewGroupsClient creates a new ResourcesClient
func NewGroupsClient(subscriptionID string, authorizer autorest.Authorizer) GroupsClient {
	client := resources.NewGroupsClient(subscriptionID)
	client.Authorizer = authorizer

	return &groupsClient{
		GroupsClient: client,
	}
}
