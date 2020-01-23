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
	prefix, err := m.managedDomainPrefix(oc.Properties.ClusterProfile.Domain)
	if err != nil || prefix == "" {
		return err
	}

	rs, err := m.recordsets.Get(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+prefix, mgmtdns.A)
	if err == nil {
		if rs.Metadata[resourceID] == nil || *rs.Metadata[resourceID] != oc.ID {
			return fmt.Errorf("recordset %q already registered", "api."+prefix)
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
	prefix, err := m.managedDomainPrefix(oc.Properties.ClusterProfile.Domain)
	if err != nil || prefix == "" {
		return err
	}

	rs, err := m.recordsets.Get(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+prefix, mgmtdns.A)
	if err != nil {
		return err
	}

	if rs.Metadata[resourceID] == nil || *rs.Metadata[resourceID] != oc.ID {
		return fmt.Errorf("recordset %q already registered", "api."+prefix)
	}

	return m.createOrUpdate(ctx, oc, ip, *rs.Etag, "")
}

func (m *manager) CreateOrUpdateRouter(ctx context.Context, oc *api.OpenShiftCluster, routerIP string) error {
	prefix, err := m.managedDomainPrefix(oc.Properties.ClusterProfile.Domain)
	if err != nil || prefix == "" {
		return err
	}

	_, err = m.recordsets.CreateOrUpdate(ctx, m.env.ResourceGroup(), m.env.Domain(), "*.apps."+prefix, mgmtdns.A, mgmtdns.RecordSet{
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
	prefix, err := m.managedDomainPrefix(oc.Properties.ClusterProfile.Domain)
	if err != nil || prefix == "" {
		return err
	}

	rs, err := m.recordsets.Get(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+prefix, mgmtdns.A)
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

	_, err = m.recordsets.Delete(ctx, m.env.ResourceGroup(), m.env.Domain(), "*.apps."+prefix, mgmtdns.A, "")
	if err != nil {
		return err
	}

	_, err = m.recordsets.Delete(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+prefix, mgmtdns.A, *rs.Etag)

	return err
}

func (m *manager) createOrUpdate(ctx context.Context, oc *api.OpenShiftCluster, ip, ifMatch, ifNoneMatch string) error {
	prefix, err := m.managedDomainPrefix(oc.Properties.ClusterProfile.Domain)
	if err != nil || prefix == "" {
		return err
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

	_, err = m.recordsets.CreateOrUpdate(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+prefix, mgmtdns.A, rs, ifMatch, ifNoneMatch)

	return err
}

func (m *manager) managedDomainPrefix(clusterDomain string) (string, error) {
	managedDomain, err := m.env.ManagedDomain(clusterDomain)
	if err != nil || managedDomain == "" {
		return "", err
	}

	return managedDomain[:strings.IndexByte(managedDomain, '.')], nil
}
