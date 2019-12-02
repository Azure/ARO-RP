package resources

//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/jim-minter/rp/pkg/util/azureclient/$GOPACKAGE DeploymentsClient,GroupsClient
//go:generate go run ../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/jim-minter/rp -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
)

// DeploymentsClient is a minimal interface for azure DeploymentsClient
type DeploymentsClient interface {
	DeploymentsClientAddons
}

type deploymentsClient struct {
	resources.DeploymentsClient
}

var _ DeploymentsClient = &deploymentsClient{}

// NewDeploymentsClient creates a new DeploymentsClient
func NewDeploymentsClient(subscriptionID string, authorizer autorest.Authorizer) DeploymentsClient {
	client := resources.NewDeploymentsClient(subscriptionID)
	client.Authorizer = authorizer
	client.PollingDuration = time.Hour
	client.PollingDelay = 10 * time.Second

	return &deploymentsClient{
		DeploymentsClient: client,
	}
}
