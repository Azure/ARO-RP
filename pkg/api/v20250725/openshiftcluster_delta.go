package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "github.com/Azure/ARO-RP/pkg/api/v20250725/generated"

type openShiftClusterDelta struct {
	ID         string                          `json:"id,omitempty" mutable:"case"`
	Name       string                          `json:"name,omitempty" mutable:"case"`
	Type       string                          `json:"type,omitempty" mutable:"case"`
	Location   string                          `json:"location,omitempty"`
	SystemData *generated.SystemData           `json:"systemData,omitempty" swagger:"readOnly"`
	Tags       map[string]*string              `json:"tags,omitempty" mutable:"true"`
	Properties openShiftClusterPropertiesDelta `json:"properties,omitempty"`
	Identity   *managedServiceIdentityDelta    `json:"identity,omitempty"`
}

type openShiftClusterPropertiesDelta struct {
	ProvisioningState               generated.ProvisioningState           `json:"provisioningState,omitempty"`
	ClusterProfile                  clusterProfileDelta                   `json:"clusterProfile,omitempty"`
	ConsoleProfile                  consoleProfileDelta                   `json:"consoleProfile,omitempty"`
	ServicePrincipalProfile         *servicePrincipalProfileDelta         `json:"servicePrincipalProfile,omitempty"`
	PlatformWorkloadIdentityProfile *platformWorkloadIdentityProfileDelta `json:"platformWorkloadIdentityProfile,omitempty"`
	NetworkProfile                  networkProfileDelta                   `json:"networkProfile,omitempty"`
	MasterProfile                   masterProfileDelta                    `json:"masterProfile,omitempty"`
	WorkerProfiles                  []workerProfileDelta                  `json:"workerProfiles,omitempty"`
	WorkerProfilesStatus            []*generated.WorkerProfile            `json:"workerProfilesStatus,omitempty" swagger:"readOnly"`
	APIServerProfile                apiServerProfileDelta                 `json:"apiserverProfile,omitempty"`
	IngressProfiles                 []ingressProfileDelta                 `json:"ingressProfiles,omitempty"`
}

type clusterProfileDelta struct {
	PullSecret           string                         `json:"pullSecret,omitempty"`
	Domain               string                         `json:"domain,omitempty"`
	Version              string                         `json:"version,omitempty"`
	ResourceGroupID      string                         `json:"resourceGroupId,omitempty"`
	FipsValidatedModules generated.FipsValidatedModules `json:"fipsValidatedModules,omitempty"`
	OIDCIssuer           *string                        `json:"oidcIssuer,omitempty" swagger:"readOnly"`
}

type consoleProfileDelta struct {
	URL string `json:"url,omitempty" swagger:"readOnly"`
}

type servicePrincipalProfileDelta struct {
	ClientID     string `json:"clientId,omitempty" mutable:"true"`
	ClientSecret string `json:"clientSecret,omitempty" mutable:"true"`
}

type platformWorkloadIdentityProfileDelta struct {
	UpgradeableTo              *string                                        `json:"upgradeableTo,omitempty" mutable:"true"`
	PlatformWorkloadIdentities map[string]*generated.PlatformWorkloadIdentity `json:"platformWorkloadIdentities,omitempty" mutable:"true"`
}

type networkProfileDelta struct {
	PodCIDR             string                     `json:"podCidr,omitempty"`
	ServiceCIDR         string                     `json:"serviceCidr,omitempty"`
	OutboundType        generated.OutboundType     `json:"outboundType,omitempty"`
	LoadBalancerProfile *loadBalancerProfileDelta  `json:"loadBalancerProfile,omitempty"`
	PreconfiguredNSG    generated.PreconfiguredNSG `json:"preconfiguredNSG,omitempty"`
}

type loadBalancerProfileDelta struct {
	ManagedOutboundIPs   *generated.ManagedOutboundIPs    `json:"managedOutboundIps,omitempty" mutable:"true"`
	EffectiveOutboundIPs []*generated.EffectiveOutboundIP `json:"effectiveOutboundIps,omitempty" swagger:"readOnly"`
}

type masterProfileDelta struct {
	VMSize              string                     `json:"vmSize,omitempty"`
	SubnetID            string                     `json:"subnetId,omitempty"`
	EncryptionAtHost    generated.EncryptionAtHost `json:"encryptionAtHost,omitempty"`
	DiskEncryptionSetID string                     `json:"diskEncryptionSetId,omitempty"`
}

type workerProfileDelta struct {
	Name                string                     `json:"name,omitempty"`
	VMSize              string                     `json:"vmSize,omitempty"`
	DiskSizeGB          int32                      `json:"diskSizeGB,omitempty"`
	SubnetID            string                     `json:"subnetId,omitempty"`
	Count               int32                      `json:"count,omitempty"`
	EncryptionAtHost    generated.EncryptionAtHost `json:"encryptionAtHost,omitempty"`
	DiskEncryptionSetID string                     `json:"diskEncryptionSetId,omitempty"`
}

type apiServerProfileDelta struct {
	Visibility generated.Visibility `json:"visibility,omitempty"`
	URL        string               `json:"url,omitempty" swagger:"readOnly"`
	IP         string               `json:"ip,omitempty" swagger:"readOnly"`
}

type ingressProfileDelta struct {
	Name       string               `json:"name,omitempty"`
	Visibility generated.Visibility `json:"visibility,omitempty"`
	IP         string               `json:"ip,omitempty" swagger:"readOnly"`
}

type managedServiceIdentityDelta struct {
	Type                   generated.ManagedServiceIdentityType       `json:"type,omitempty"`
	PrincipalID            string                                     `json:"principalId,omitempty" mutable:"true"`
	TenantID               string                                     `json:"tenantId,omitempty" mutable:"true"`
	UserAssignedIdentities map[string]*generated.UserAssignedIdentity `json:"userAssignedIdentities,omitempty" mutable:"true"`
}

func newOpenShiftClusterDelta(oc *OpenShiftCluster) *openShiftClusterDelta {
	delta := &openShiftClusterDelta{
		ID:       value(oc.ID),
		Name:     value(oc.Name),
		Type:     value(oc.Type),
		Location: value(oc.Location),
		Tags:     oc.Tags,
	}
	if oc.SystemData != nil {
		delta.SystemData = oc.SystemData
	}
	if oc.Identity != nil {
		delta.Identity = &managedServiceIdentityDelta{
			Type:                   value(oc.Identity.Type),
			PrincipalID:            value(oc.Identity.PrincipalID),
			TenantID:               value(oc.Identity.TenantID),
			UserAssignedIdentities: oc.Identity.UserAssignedIdentities,
		}
	}
	if oc.Properties == nil {
		return delta
	}

	p := oc.Properties
	delta.Properties.ProvisioningState = value(p.ProvisioningState)
	if p.ClusterProfile != nil {
		delta.Properties.ClusterProfile = clusterProfileDelta{
			PullSecret:           value(p.ClusterProfile.PullSecret),
			Domain:               value(p.ClusterProfile.Domain),
			Version:              value(p.ClusterProfile.Version),
			ResourceGroupID:      value(p.ClusterProfile.ResourceGroupID),
			FipsValidatedModules: value(p.ClusterProfile.FipsValidatedModules),
			OIDCIssuer:           p.ClusterProfile.OidcIssuer,
		}
	}
	if p.ConsoleProfile != nil {
		delta.Properties.ConsoleProfile.URL = value(p.ConsoleProfile.URL)
	}
	if p.ServicePrincipalProfile != nil {
		delta.Properties.ServicePrincipalProfile = &servicePrincipalProfileDelta{
			ClientID:     value(p.ServicePrincipalProfile.ClientID),
			ClientSecret: value(p.ServicePrincipalProfile.ClientSecret),
		}
	}
	if p.PlatformWorkloadIdentityProfile != nil {
		delta.Properties.PlatformWorkloadIdentityProfile = &platformWorkloadIdentityProfileDelta{
			UpgradeableTo:              p.PlatformWorkloadIdentityProfile.UpgradeableTo,
			PlatformWorkloadIdentities: p.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities,
		}
	}
	if p.NetworkProfile != nil {
		delta.Properties.NetworkProfile = networkProfileDelta{
			PodCIDR:          value(p.NetworkProfile.PodCidr),
			ServiceCIDR:      value(p.NetworkProfile.ServiceCidr),
			OutboundType:     value(p.NetworkProfile.OutboundType),
			PreconfiguredNSG: value(p.NetworkProfile.PreconfiguredNSG),
		}
		if p.NetworkProfile.LoadBalancerProfile != nil {
			delta.Properties.NetworkProfile.LoadBalancerProfile = &loadBalancerProfileDelta{
				ManagedOutboundIPs:   p.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs,
				EffectiveOutboundIPs: p.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs,
			}
		}
	}
	if p.MasterProfile != nil {
		delta.Properties.MasterProfile = masterProfileDelta{
			VMSize:              value(p.MasterProfile.VMSize),
			SubnetID:            value(p.MasterProfile.SubnetID),
			EncryptionAtHost:    value(p.MasterProfile.EncryptionAtHost),
			DiskEncryptionSetID: value(p.MasterProfile.DiskEncryptionSetID),
		}
	}
	for _, worker := range p.WorkerProfiles {
		if worker == nil {
			delta.Properties.WorkerProfiles = append(delta.Properties.WorkerProfiles, workerProfileDelta{})
			continue
		}
		delta.Properties.WorkerProfiles = append(delta.Properties.WorkerProfiles, workerProfileDelta{
			Name:                value(worker.Name),
			VMSize:              value(worker.VMSize),
			DiskSizeGB:          value(worker.DiskSizeGB),
			SubnetID:            value(worker.SubnetID),
			Count:               value(worker.Count),
			EncryptionAtHost:    value(worker.EncryptionAtHost),
			DiskEncryptionSetID: value(worker.DiskEncryptionSetID),
		})
	}
	delta.Properties.WorkerProfilesStatus = p.WorkerProfilesStatus
	if p.ApiserverProfile != nil {
		delta.Properties.APIServerProfile = apiServerProfileDelta{
			Visibility: value(p.ApiserverProfile.Visibility),
			URL:        value(p.ApiserverProfile.URL),
			IP:         value(p.ApiserverProfile.IP),
		}
	}
	for _, ingress := range p.IngressProfiles {
		if ingress == nil {
			delta.Properties.IngressProfiles = append(delta.Properties.IngressProfiles, ingressProfileDelta{})
			continue
		}
		delta.Properties.IngressProfiles = append(delta.Properties.IngressProfiles, ingressProfileDelta{
			Name:       value(ingress.Name),
			Visibility: value(ingress.Visibility),
			IP:         value(ingress.IP),
		})
	}
	return delta
}
