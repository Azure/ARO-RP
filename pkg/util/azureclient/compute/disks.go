package compute

//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/jim-minter/rp/pkg/util/azureclient/$GOPACKAGE DisksClient,VirtualMachinesClient
//go:generate go run ../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/jim-minter/rp -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest"
)

// DisksClient is a minimal interface for azure DisksClient
type DisksClient interface {
	DisksClientAddons
}

type disksClient struct {
	compute.DisksClient
}

var _ DisksClient = &disksClient{}

// NewDisksClient creates a new DisksClient
func NewDisksClient(subscriptionID string, authorizer autorest.Authorizer) DisksClient {
	client := compute.NewDisksClient(subscriptionID)
	client.Authorizer = authorizer

	return &disksClient{
		DisksClient: client,
	}
}
