package armredhatopenshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	armredhatopenshift "github.com/Azure/ARO-RP/pkg/client/sdk/resourcemanager/redhatopenshift/armredhatopenshift"
)

type OpenShiftClustersAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters armredhatopenshift.OpenShiftCluster) error
	UpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters armredhatopenshift.OpenShiftClusterUpdate) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, resourceName string) error
	List(ctx context.Context, options *armredhatopenshift.OpenShiftClustersClientListOptions) ([]*armredhatopenshift.OpenShiftCluster, error)
	ListByResourceGroup(ctx context.Context, resourceGroupName string, options *armredhatopenshift.OpenShiftClustersClientListByResourceGroupOptions) ([]*armredhatopenshift.OpenShiftCluster, error)
}

func (c *openShiftClustersClient) List(ctx context.Context, options *armredhatopenshift.OpenShiftClustersClientListOptions) (result []*armredhatopenshift.OpenShiftCluster, err error) {
	pager := c.NewListPager(options)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, page.Value...)
	}
	return result, nil
}

func (c *openShiftClustersClient) ListByResourceGroup(ctx context.Context, resourceGroupName string, options *armredhatopenshift.OpenShiftClustersClientListByResourceGroupOptions) (result []*armredhatopenshift.OpenShiftCluster, err error) {
	pager := c.NewListByResourceGroupPager(resourceGroupName, options)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, page.Value...)
	}
	return result, nil
}

func (c *openShiftClustersClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters armredhatopenshift.OpenShiftCluster) error {
	poller, err := c.BeginCreateOrUpdate(ctx, resourceGroupName, resourceName, stripReadOnlyFields(parameters), nil)
	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

// stripReadOnlyFields returns a copy of oc with all read-only fields removed. We have to make sure to do this before
// PUT requests because the TypeSpec-generated track 2 SDK doesn't automatically exclude read-only fields from PUT
// requests like the Autorest-generated track 1 SDK did.
func stripReadOnlyFields(oc armredhatopenshift.OpenShiftCluster) armredhatopenshift.OpenShiftCluster {
	result := armredhatopenshift.OpenShiftCluster{
		Location: oc.Location,
		Tags:     oc.Tags,
	}

	if oc.Identity != nil {
		result.Identity = &armredhatopenshift.ManagedServiceIdentity{
			Type:                   oc.Identity.Type,
			UserAssignedIdentities: make(map[string]*armredhatopenshift.UserAssignedIdentity),
		}
		for k, v := range oc.Identity.UserAssignedIdentities {
			if v != nil {
				result.Identity.UserAssignedIdentities[k] = &armredhatopenshift.UserAssignedIdentity{}
			}
		}
	}

	if p := oc.Properties; p != nil {
		result.Properties = &armredhatopenshift.OpenShiftClusterProperties{
			MasterProfile:           p.MasterProfile,
			ProvisioningState:       p.ProvisioningState,
			ServicePrincipalProfile: p.ServicePrincipalProfile,
			WorkerProfiles:          p.WorkerProfiles,
		}
		if p.ApiserverProfile != nil {
			result.Properties.ApiserverProfile = &armredhatopenshift.APIServerProfile{
				Visibility: p.ApiserverProfile.Visibility,
			}
		}
		if p.ClusterProfile != nil {
			result.Properties.ClusterProfile = &armredhatopenshift.ClusterProfile{
				Domain:               p.ClusterProfile.Domain,
				FipsValidatedModules: p.ClusterProfile.FipsValidatedModules,
				PullSecret:           p.ClusterProfile.PullSecret,
				ResourceGroupID:      p.ClusterProfile.ResourceGroupID,
				Version:              p.ClusterProfile.Version,
			}
		}
		if p.ConsoleProfile != nil {
			result.Properties.ConsoleProfile = &armredhatopenshift.ConsoleProfile{}
		}
		if p.IngressProfiles != nil {
			result.Properties.IngressProfiles = make([]*armredhatopenshift.IngressProfile, len(p.IngressProfiles))
			for i, ip := range p.IngressProfiles {
				if ip != nil {
					result.Properties.IngressProfiles[i] = &armredhatopenshift.IngressProfile{
						Name:       ip.Name,
						Visibility: ip.Visibility,
					}
				}
			}
		}
		if p.NetworkProfile != nil {
			result.Properties.NetworkProfile = &armredhatopenshift.NetworkProfile{
				OutboundType:     p.NetworkProfile.OutboundType,
				PodCidr:          p.NetworkProfile.PodCidr,
				PreconfiguredNSG: p.NetworkProfile.PreconfiguredNSG,
				ServiceCidr:      p.NetworkProfile.ServiceCidr,
			}
			if p.NetworkProfile.LoadBalancerProfile != nil {
				result.Properties.NetworkProfile.LoadBalancerProfile = &armredhatopenshift.LoadBalancerProfile{
					ManagedOutboundIPs: p.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs,
				}
			}
		}
		if p.PlatformWorkloadIdentityProfile != nil {
			result.Properties.PlatformWorkloadIdentityProfile = &armredhatopenshift.PlatformWorkloadIdentityProfile{
				PlatformWorkloadIdentities: make(map[string]*armredhatopenshift.PlatformWorkloadIdentity),
				UpgradeableTo:              p.PlatformWorkloadIdentityProfile.UpgradeableTo,
			}
			for k, v := range p.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
				if v != nil {
					result.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k] = &armredhatopenshift.PlatformWorkloadIdentity{
						ResourceID: v.ResourceID,
					}
				}
			}
		}
	}

	return result
}

func (c *openShiftClustersClient) UpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters armredhatopenshift.OpenShiftClusterUpdate) error {
	poller, err := c.BeginUpdate(ctx, resourceGroupName, resourceName, parameters, nil)
	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *openShiftClustersClient) DeleteAndWait(ctx context.Context, resourceGroupName string, resourceName string) error {
	poller, err := c.BeginDelete(ctx, resourceGroupName, resourceName, nil)
	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
