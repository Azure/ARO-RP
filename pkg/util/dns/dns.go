package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/dns"
)

const resourceID = "resourceId"

type Manager interface {
	Create(context.Context, *api.OpenShiftCluster) error
	Update(context.Context, *api.OpenShiftCluster, string) error
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

func (m *manager) Create(ctx context.Context, oc *api.OpenShiftCluster) error {
	clusterDomain := m.managedDomain(oc.Properties.ClusterDomain)
	if clusterDomain == "" {
		return nil
	}

	rs, err := m.recordsets.Get(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+clusterDomain, mgmtdns.A)
	if err == nil {
		if rs.Metadata[resourceID] == nil || *rs.Metadata[resourceID] != oc.ID {
			return fmt.Errorf("recordset %q already registered", "api."+clusterDomain)
		}

		return nil
	}

	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		err = nil
	}
	if err != nil {
		return err
	}

	return m.createOrUpdate(ctx, oc, "", "", "*")
}

func (m *manager) Update(ctx context.Context, oc *api.OpenShiftCluster, ip string) error {
	clusterDomain := m.managedDomain(oc.Properties.ClusterDomain)
	if clusterDomain == "" {
		return nil
	}

	rs, err := m.recordsets.Get(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+clusterDomain, mgmtdns.A)
	if err != nil {
		return err
	}

	if rs.Metadata[resourceID] == nil || *rs.Metadata[resourceID] != oc.ID {
		return fmt.Errorf("recordset %q already registered", "api."+clusterDomain)
	}

	return m.createOrUpdate(ctx, oc, ip, *rs.Etag, "")
}

func (m *manager) CreateOrUpdateRouter(ctx context.Context, oc *api.OpenShiftCluster, routerIP string) error {
	clusterDomain := m.managedDomain(oc.Properties.ClusterDomain)
	if clusterDomain == "" {
		return nil
	}

	_, err := m.recordsets.CreateOrUpdate(ctx, m.env.ResourceGroup(), m.env.Domain(), "*.apps."+clusterDomain, mgmtdns.A, mgmtdns.RecordSet{
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			TTL: to.Int64Ptr(300),
			ARecords: &[]mgmtdns.ARecord{
				{
					Ipv4Address: to.StringPtr(routerIP),
				},
			},
		},
	}, "", "")

	return err
}

func (m *manager) Delete(ctx context.Context, oc *api.OpenShiftCluster) error {
	clusterDomain := m.managedDomain(oc.Properties.ClusterDomain)
	if clusterDomain == "" {
		return nil
	}

	rs, err := m.recordsets.Get(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+clusterDomain, mgmtdns.A)
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	if rs.Metadata[resourceID] == nil || *rs.Metadata[resourceID] != oc.ID {
		return nil
	}

	_, err = m.recordsets.Delete(ctx, m.env.ResourceGroup(), m.env.Domain(), "*.apps."+clusterDomain, mgmtdns.A, "")
	if err != nil {
		return err
	}

	_, err = m.recordsets.Delete(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+clusterDomain, mgmtdns.A, *rs.Etag)

	return err
}

func (m *manager) createOrUpdate(ctx context.Context, oc *api.OpenShiftCluster, ip, ifMatch, ifNoneMatch string) error {
	clusterDomain := m.managedDomain(oc.Properties.ClusterDomain)
	if clusterDomain == "" {
		return nil
	}

	rs := mgmtdns.RecordSet{
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			Metadata: map[string]*string{
				resourceID: to.StringPtr(oc.ID),
			},
			TTL: to.Int64Ptr(300),
		},
	}

	if ip != "" {
		rs.ARecords = &[]mgmtdns.ARecord{
			{
				Ipv4Address: to.StringPtr(ip),
			},
		}
	}

	_, err := m.recordsets.CreateOrUpdate(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+clusterDomain, mgmtdns.A, rs, ifMatch, ifNoneMatch)

	return err
}

func (m *manager) managedDomain(clusterDomain string) string {
	clusterDomain = strings.TrimSuffix(clusterDomain, "."+m.env.Domain())
	if !strings.ContainsRune(clusterDomain, '.') {
		return clusterDomain
	}
	return ""
}
