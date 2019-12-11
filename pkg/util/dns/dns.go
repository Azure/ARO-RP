package dns

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/util/instancemetadata"
)

type Manager interface {
	Domain() string
	CreateOrUpdate(context.Context, *api.OpenShiftCluster, string) error
	Delete(context.Context, *api.OpenShiftCluster) error
}

type manager struct {
	instancemetadata instancemetadata.InstanceMetadata

	recordsets dns.RecordSetsClient

	domain string
}

func NewManager(ctx context.Context, instancemetadata instancemetadata.InstanceMetadata, rpAuthorizer autorest.Authorizer) (Manager, error) {
	m := &manager{
		instancemetadata: instancemetadata,

		recordsets: dns.NewRecordSetsClient(instancemetadata.SubscriptionID()),
	}

	m.recordsets.Authorizer = rpAuthorizer

	zones := dns.NewZonesClient(instancemetadata.SubscriptionID())
	zones.Authorizer = rpAuthorizer

	page, err := zones.ListByResourceGroup(ctx, m.instancemetadata.ResourceGroup(), nil)
	if err != nil {
		return nil, err
	}

	zs := page.Values()
	if len(zs) != 1 {
		return nil, fmt.Errorf("found at least %d zones, expected 1", len(zs))
	}

	m.domain = *zs[0].Name

	return m, nil
}

func (m *manager) Domain() string {
	return m.domain
}

func (m *manager) CreateOrUpdate(ctx context.Context, oc *api.OpenShiftCluster, routerIP string) error {
	_, err := m.recordsets.CreateOrUpdate(ctx, m.instancemetadata.ResourceGroup(), m.domain, "api."+oc.Properties.DomainName, dns.CNAME, dns.RecordSet{
		RecordSetProperties: &dns.RecordSetProperties{
			TTL: to.Int64Ptr(300),
			CnameRecord: &dns.CnameRecord{
				Cname: to.StringPtr(oc.Properties.DomainName + "." + oc.Location + ".cloudapp.azure.com"),
			},
		},
	}, "", "")
	if err != nil {
		return err
	}

	_, err = m.recordsets.CreateOrUpdate(ctx, m.instancemetadata.ResourceGroup(), m.domain, "*.apps."+oc.Properties.DomainName, dns.A, dns.RecordSet{
		RecordSetProperties: &dns.RecordSetProperties{
			TTL: to.Int64Ptr(300),
			ARecords: &[]dns.ARecord{
				{
					Ipv4Address: to.StringPtr(routerIP),
				},
			},
		},
	}, "", "")

	return err
}

func (m *manager) Delete(ctx context.Context, oc *api.OpenShiftCluster) error {
	_, err := m.recordsets.Delete(ctx, m.instancemetadata.ResourceGroup(), m.domain, "api."+oc.Properties.DomainName, dns.CNAME, "")
	if err != nil {
		return err
	}

	_, err = m.recordsets.Delete(ctx, m.instancemetadata.ResourceGroup(), m.domain, "*.apps."+oc.Properties.DomainName, dns.A, "")

	return err
}
