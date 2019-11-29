package dns

//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/jim-minter/rp/pkg/util/azureclient/$GOPACKAGE RecordSetsClient
//go:generate gofmt -s -l -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go
//go:generate go run ../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/jim-minter/rp -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest"
)

// RecordSetsClient is a minimal interface for azure RecordSetsClient
type RecordSetsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType dns.RecordType, parameters dns.RecordSet, ifMatch string, ifNoneMatch string) (result dns.RecordSet, err error)
	Delete(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType dns.RecordType, ifMatch string) (result autorest.Response, err error)
	Get(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType dns.RecordType) (result dns.RecordSet, err error)
}

type recordSetsClient struct {
	dns.RecordSetsClient
}

var _ RecordSetsClient = &recordSetsClient{}

// NewRecordSetsClient creates a new RecordSetsClient
func NewRecordSetsClient(subscriptionID string, authorizer autorest.Authorizer) RecordSetsClient {
	client := dns.NewRecordSetsClient(subscriptionID)
	client.Authorizer = authorizer

	return &recordSetsClient{
		RecordSetsClient: client,
	}
}
