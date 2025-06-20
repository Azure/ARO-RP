package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	sdkdns "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armdns"
	azerrors "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/errors"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
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
	recordsets armdns.RecordSetsClient
}

func NewManager(env env.Interface, tokenCredential azcore.TokenCredential) Manager {
	options := arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: env.Environment().Cloud,
		},
	}
	return &manager{
		env:        env,
		recordsets: armdns.NewRecordSetsClient(env.SubscriptionID(), tokenCredential, &options),
	}
}

func (m *manager) Create(ctx context.Context, oc *api.OpenShiftCluster) error {
	prefix, err := m.managedDomainPrefix(oc.Properties.ClusterProfile.Domain)
	if err != nil || prefix == "" {
		return err
	}

	rs, err := m.recordsets.Get(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+prefix, sdkdns.RecordTypeA, nil)
	if err == nil {
		if rs.Properties.Metadata[resourceID] == nil || *rs.Properties.Metadata[resourceID] != oc.ID {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeDuplicateDomain, "", fmt.Sprintf("The provided domain '%s' is already in use by a cluster.", oc.Properties.ClusterProfile.Domain))
		}

		return nil
	}

	if azerrors.IsNotFoundError(err) {
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

	rs, err := m.recordsets.Get(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+prefix, sdkdns.RecordTypeA, nil)
	if err != nil {
		return err
	}

	if rs.Properties.Metadata[resourceID] == nil || *rs.Properties.Metadata[resourceID] != oc.ID {
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
	rs, err := m.recordsets.Get(ctx, m.env.ResourceGroup(), m.env.Domain(), "*.apps."+prefix, sdkdns.RecordTypeA, nil)
	if azerrors.IsNotFoundError(err) {
		isCreate = true
	}

	// If record exists and routerIP already match - skip CreateOrUpdate
	if err == nil && !isCreate {
		for _, a := range rs.Properties.ARecords {
			if *a.IPv4Address == routerIP {
				return nil
			}
		}
	}

	_, err = m.recordsets.CreateOrUpdate(ctx, m.env.ResourceGroup(), m.env.Domain(), "*.apps."+prefix, sdkdns.RecordTypeA, sdkdns.RecordSet{
		Properties: &sdkdns.RecordSetProperties{
			TTL: pointerutils.ToPtr(int64(300)),
			ARecords: []*sdkdns.ARecord{
				{
					IPv4Address: &routerIP,
				},
			},
		},
	}, &sdkdns.RecordSetsClientCreateOrUpdateOptions{
		IfMatch:     nil,
		IfNoneMatch: nil,
	})

	return err
}

func (m *manager) Delete(ctx context.Context, oc *api.OpenShiftCluster) error {
	prefix, err := m.managedDomainPrefix(oc.Properties.ClusterProfile.Domain)
	if err != nil || prefix == "" {
		return err
	}

	rs, err := m.recordsets.Get(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+prefix, sdkdns.RecordTypeA, nil)
	if azerrors.IsNotFoundError(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if rs.Properties.Metadata[resourceID] == nil || *rs.Properties.Metadata[resourceID] != oc.ID {
		return nil
	}

	_, err = m.recordsets.Delete(ctx, m.env.ResourceGroup(), m.env.Domain(), "*.apps."+prefix, sdkdns.RecordTypeA, &sdkdns.RecordSetsClientDeleteOptions{
		IfMatch: pointerutils.ToPtr(""),
	})
	if err != nil {
		return err
	}

	_, err = m.recordsets.Delete(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+prefix, sdkdns.RecordTypeA, &sdkdns.RecordSetsClientDeleteOptions{
		IfMatch: rs.Etag,
	})

	return err
}

func (m *manager) createOrUpdate(ctx context.Context, oc *api.OpenShiftCluster, ip, ifMatch, ifNoneMatch string) error {
	prefix, err := m.managedDomainPrefix(oc.Properties.ClusterProfile.Domain)
	if err != nil || prefix == "" {
		return err
	}

	rs := sdkdns.RecordSet{
		Properties: &sdkdns.RecordSetProperties{
			Metadata: map[string]*string{
				resourceID: &oc.ID,
			},
			TTL: pointerutils.ToPtr(int64(300)),
		},
	}

	if ip != "" {
		rs.Properties.ARecords = []*sdkdns.ARecord{
			{
				IPv4Address: &ip,
			},
		}
	}

	_, err = m.recordsets.CreateOrUpdate(ctx, m.env.ResourceGroup(), m.env.Domain(), "api."+prefix, sdkdns.RecordTypeA, rs,
		&sdkdns.RecordSetsClientCreateOrUpdateOptions{
			IfMatch:     &ifMatch,
			IfNoneMatch: &ifNoneMatch,
		})

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
