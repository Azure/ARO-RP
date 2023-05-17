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

const (
	resourceID = "resourceId"
)

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

		recordsets: dns.NewRecordSetsClient(env.Environment(), env.SubscriptionID(), localFPAuthorizer),
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
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeDuplicateDomain, "", "The provided domain '%s' is already in use by a cluster.", oc.Properties.ClusterProfile.Domain)
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

	var isCreate bool
	rs, err := m.recordsets.Get(ctx, m.env.ResourceGroup(), m.env.Domain(), "*.apps."+prefix, mgmtdns.A)
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		isCreate = true
	}

	// If record exists and routerIP already match - skip CreateOrUpdate
	if err == nil && !isCreate {
		for _, a := range *rs.ARecords {
			if *a.Ipv4Address == routerIP {
				return nil
			}
		}
	}

	_, err = m.recordsets.CreateOrUpdate(ctx, m.env.ResourceGroup(), m.env.Domain(), "*.apps."+prefix, mgmtdns.A, mgmtdns.RecordSet{
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			TTL: to.Int64Ptr(300),
			ARecords: &[]mgmtdns.ARecord{
				{
					Ipv4Address: &routerIP,
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
				resourceID: &oc.ID,
			},
			TTL: to.Int64Ptr(300),
		},
	}

	if ip != "" {
		rs.ARecords = &[]mgmtdns.ARecord{
			{
				Ipv4Address: &ip,
			},
		}
	}

	_, err = m.recordsets.CreateOrUpdate(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+prefix, mgmtdns.A, rs, ifMatch, ifNoneMatch)

	return err
}

func (m *manager) managedDomainPrefix(clusterDomain string) (string, error) {
	managedDomain, err := ManagedDomain(m.env, clusterDomain)
	if err != nil || managedDomain == "" {
		return "", err
	}

	return managedDomain[:strings.IndexByte(managedDomain, '.')], nil
}

// ManagedDomain returns the fully qualified domain of a cluster if we manage
// it.  If we don't, it returns the empty string.  We manage only domains of the
// form "foo.$LOCATION.aroapp.io" and "foo" (we consider this a short form of
// the former).
func ManagedDomain(env env.Interface, domain string) (string, error) {
	if domain == "" ||
		strings.HasPrefix(domain, ".") ||
		strings.HasSuffix(domain, ".") {
		// belt and braces: validation should already prevent this
		return "", fmt.Errorf("invalid domain %q", domain)
	}

	domain = strings.TrimSuffix(domain, "."+env.Domain())
	if strings.ContainsRune(domain, '.') {
		return "", nil
	}
	return domain + "." + env.Domain(), nil
}
