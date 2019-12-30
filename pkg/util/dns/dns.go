package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	dnsmgmt "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/dns"
)

type Manager interface {
	Domain() string
	CreateOrUpdate(context.Context, *api.OpenShiftCluster) error
	CreateOrUpdateRouter(context.Context, *api.OpenShiftCluster, string) error
	Delete(context.Context, *api.OpenShiftCluster) error
}

type manager struct {
	env        env.Interface
	recordsets dns.RecordSetsClient
}

func NewManager(env env.Interface, localFPAuthorizer autorest.Authorizer) Manager {
	return &manager{
		env: env,

		recordsets: dns.NewRecordSetsClient(env.SubscriptionID(), localFPAuthorizer),
	}
}

func (m *manager) Domain() string {
	return m.env.Domain()
}

func (m *manager) CreateOrUpdate(ctx context.Context, oc *api.OpenShiftCluster) error {
	_, err := m.recordsets.CreateOrUpdate(ctx, m.env.ResourceGroup(), m.Domain(), "api."+oc.Properties.DomainName, dnsmgmt.CNAME, dnsmgmt.RecordSet{
		RecordSetProperties: &dnsmgmt.RecordSetProperties{
			TTL: to.Int64Ptr(300),
			CnameRecord: &dnsmgmt.CnameRecord{
				Cname: to.StringPtr(oc.Properties.DomainName + "." + oc.Location + ".cloudapp.azure.com"),
			},
		},
	}, "", "")

	return err
}

func (m *manager) CreateOrUpdateRouter(ctx context.Context, oc *api.OpenShiftCluster, routerIP string) error {
	_, err := m.recordsets.CreateOrUpdate(ctx, m.env.ResourceGroup(), m.Domain(), "*.apps."+oc.Properties.DomainName, dnsmgmt.A, dnsmgmt.RecordSet{
		RecordSetProperties: &dnsmgmt.RecordSetProperties{
			TTL: to.Int64Ptr(300),
			ARecords: &[]dnsmgmt.ARecord{
				{
					Ipv4Address: to.StringPtr(routerIP),
				},
			},
		},
	}, "", "")

	return err
}

func (m *manager) Delete(ctx context.Context, oc *api.OpenShiftCluster) error {
	_, err := m.recordsets.Delete(ctx, m.env.ResourceGroup(), m.Domain(), "api."+oc.Properties.DomainName, dnsmgmt.CNAME, "")
	if err != nil {
		return err
	}

	_, err = m.recordsets.Delete(ctx, m.env.ResourceGroup(), m.Domain(), "*.apps."+oc.Properties.DomainName, dnsmgmt.A, "")

	return err
}
