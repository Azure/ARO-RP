package dns

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest"

	"github.com/jim-minter/rp/pkg/api"
)

type Manager interface {
	Domain() string
	Delete(context.Context, *api.OpenShiftCluster) error
}

type manager struct {
	recordsets dns.RecordSetsClient
	zones      dns.ZonesClient

	resourceGroup string
	domain        string
}

func NewManager(ctx context.Context, subscriptionID string, rpAuthorizer autorest.Authorizer, resourceGroup string) (Manager, error) {
	m := &manager{
		recordsets: dns.NewRecordSetsClient(subscriptionID),
		zones:      dns.NewZonesClient(subscriptionID),

		resourceGroup: resourceGroup,
	}

	m.recordsets.Authorizer = rpAuthorizer
	m.zones.Authorizer = rpAuthorizer

	page, err := m.zones.ListByResourceGroup(ctx, m.resourceGroup, nil)
	if err != nil {
		return nil, err
	}

	zones := page.Values()
	if len(zones) != 1 {
		return nil, fmt.Errorf("found at least %d zones, expected 1", len(zones))
	}

	m.domain = *zones[0].Name

	return m, nil
}

func (m *manager) Domain() string {
	return m.domain
}

func (m *manager) Delete(ctx context.Context, oc *api.OpenShiftCluster) error {
	_, err := m.recordsets.Delete(ctx, m.resourceGroup, m.domain, "api."+oc.Properties.DomainName, dns.CNAME, "")
	if err != nil {
		return err
	}

	return nil
}
