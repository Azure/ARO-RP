package compute

//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/jim-minter/rp/pkg/util/azureclient/$GOPACKAGE VirtualMachinesClient,DisksClient
//go:generate go run ../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/jim-minter/rp -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest"
)

// VirtualMachinesClient is a minimal interface for azure VirtualMachinesClient
type VirtualMachinesClient interface {
	VirtualMachinesClientAddons
}

type virtualMachinesClient struct {
	compute.VirtualMachinesClient
}

var _ VirtualMachinesClient = &virtualMachinesClient{}

// NewVirtualMachinesClient creates a new VirtualMachinesClient
func NewVirtualMachinesClient(subscriptionID string, authorizer autorest.Authorizer) VirtualMachinesClient {
	client := compute.NewVirtualMachinesClient(subscriptionID)
	client.Authorizer = authorizer

	return &virtualMachinesClient{
		VirtualMachinesClient: client,
	}
}
