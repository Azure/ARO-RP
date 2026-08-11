package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

func TestOpenShiftClusterConverterToExternal(t *testing.T) {
	internal := converterInternalCluster()
	got := (openShiftClusterConverter{}).ToExternal(internal).(*OpenShiftCluster)
	want := converterExternalCluster()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ToExternal() mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestOpenShiftClusterConverterToExternalSparse(t *testing.T) {
	got := (openShiftClusterConverter{}).ToExternal(&api.OpenShiftCluster{}).(*OpenShiftCluster)

	if got.Properties == nil || got.Properties.ClusterProfile == nil || got.Properties.ConsoleProfile == nil ||
		got.Properties.NetworkProfile == nil || got.Properties.MasterProfile == nil || got.Properties.ApiserverProfile == nil {
		t.Fatal("value-based profiles must remain represented by non-nil generated model pointers")
	}
	if got.Properties.ServicePrincipalProfile != nil || got.Properties.PlatformWorkloadIdentityProfile != nil ||
		got.Properties.NetworkProfile.LoadBalancerProfile != nil || got.Identity != nil {
		t.Fatal("optional profiles must remain nil")
	}
	if got.Tags != nil || got.Properties.WorkerProfiles != nil || got.Properties.WorkerProfilesStatus != nil || got.Properties.IngressProfiles != nil {
		t.Fatal("nil collections must remain nil")
	}
	if got.Properties.ClusterProfile.PullSecret != nil {
		t.Fatal("empty pull secret must remain nil")
	}
}

func TestOpenShiftClusterConverterToExternalDoesNotAlias(t *testing.T) {
	internal := converterInternalCluster()
	external := (openShiftClusterConverter{}).ToExternal(internal).(*OpenShiftCluster)

	*external.ID = "changed"
	*external.Tags["tag"] = "changed"
	*external.Properties.WorkerProfiles[0].Name = "changed"
	*external.Identity.UserAssignedIdentities["identity"].ClientID = "changed"
	*external.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["operator"].ResourceID = "changed"

	if internal.ID != "resource-id" {
		t.Fatalf("ID aliased: %q", internal.ID)
	}
	if internal.Tags["tag"] != "value" {
		t.Fatalf("tags aliased: %q", internal.Tags["tag"])
	}
	if internal.Properties.WorkerProfiles[0].Name != "worker" {
		t.Fatalf("worker profile aliased: %q", internal.Properties.WorkerProfiles[0].Name)
	}
	if internal.Identity.UserAssignedIdentities["identity"].ClientID != "identity-client" {
		t.Fatalf("user-assigned identity aliased: %q", internal.Identity.UserAssignedIdentities["identity"].ClientID)
	}
	if internal.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["operator"].ResourceID != "operator-resource" {
		t.Fatalf("platform identity aliased: %q", internal.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["operator"].ResourceID)
	}
}

func TestOpenShiftClusterConverterToExternalList(t *testing.T) {
	first := converterInternalCluster()
	second := converterInternalCluster()
	second.Name = "second"

	got := (openShiftClusterConverter{}).ToExternalList([]*api.OpenShiftCluster{first, second}, "next").(*OpenShiftClusterList)
	if got.NextLink != "next" {
		t.Fatalf("NextLink = %q, want %q", got.NextLink, "next")
	}
	if len(got.OpenShiftClusters) != 2 || value(got.OpenShiftClusters[0].Name) != "cluster" || value(got.OpenShiftClusters[1].Name) != "second" {
		t.Fatalf("unexpected converted clusters: %#v", got.OpenShiftClusters)
	}

	empty := (openShiftClusterConverter{}).ToExternalList(nil, "").(*OpenShiftClusterList)
	if empty.OpenShiftClusters == nil || len(empty.OpenShiftClusters) != 0 {
		t.Fatalf("empty list = %#v, want non-nil empty slice", empty.OpenShiftClusters)
	}
}

func TestOpenShiftClusterConverterToInternal(t *testing.T) {
	external := converterExternalCluster()
	got := &api.OpenShiftCluster{}
	(openShiftClusterConverter{}).ToInternal(external, got)

	want := converterInternalCluster()
	want.Properties.ClusterProfile.OIDCIssuer = nil
	operator := want.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["operator"]
	operator.ClientID = ""
	operator.ObjectID = ""
	want.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["operator"] = operator

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ToInternal() mismatch\ngot:  %#v\nwant: %#v", got, want)
	}

	*external.ID = "changed"
	*external.Tags["tag"] = "changed"
	*external.Properties.WorkerProfiles[0].Name = "changed"
	if got.ID != "resource-id" || got.Tags["tag"] != "value" || got.Properties.WorkerProfiles[0].Name != "worker" {
		t.Fatal("ToInternal retained aliases to the external model")
	}
}

func TestOpenShiftClusterConverterToInternalPreservesExistingValues(t *testing.T) {
	upgradeableTo := api.UpgradeableTo("4.15.1")
	got := &api.OpenShiftCluster{
		Identity:   &api.ManagedServiceIdentity{PrincipalID: "existing-principal"},
		SystemData: api.SystemData{CreatedBy: "existing-creator"},
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile:          api.ClusterProfile{OIDCIssuer: pointerutils.ToPtr(api.OIDCIssuer("existing-issuer"))},
			ConsoleProfile:          api.ConsoleProfile{URL: "existing-console"},
			ServicePrincipalProfile: &api.ServicePrincipalProfile{ClientID: "existing-client"},
			PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
				UpgradeableTo: &upgradeableTo,
				PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
					"same":    {ResourceID: "same-resource", ClientID: "same-client", ObjectID: "same-object"},
					"changed": {ResourceID: "old-resource", ClientID: "old-client", ObjectID: "old-object"},
				},
			},
			NetworkProfile: api.NetworkProfile{LoadBalancerProfile: &api.LoadBalancerProfile{
				ManagedOutboundIPs:   &api.ManagedOutboundIPs{Count: 1},
				EffectiveOutboundIPs: []api.EffectiveOutboundIP{{ID: "existing-effective"}},
			}},
			APIServerProfile:     api.APIServerProfile{URL: "existing-api", IP: "existing-api-ip"},
			WorkerProfiles:       []api.WorkerProfile{{Name: "existing-worker"}},
			WorkerProfilesStatus: []api.WorkerProfile{{Name: "existing-status"}},
			IngressProfiles:      []api.IngressProfile{{Name: "existing-ingress", IP: "existing-ingress-ip"}},
		},
	}

	external := &OpenShiftCluster{OpenShiftCluster: generated.OpenShiftCluster{
		Properties: &generated.OpenShiftClusterProperties{
			ClusterProfile: &generated.ClusterProfile{},
			ConsoleProfile: &generated.ConsoleProfile{},
			PlatformWorkloadIdentityProfile: &generated.PlatformWorkloadIdentityProfile{
				PlatformWorkloadIdentities: map[string]*generated.PlatformWorkloadIdentity{
					"same":    {ResourceID: pointerutils.ToPtr("same-resource")},
					"changed": {ResourceID: pointerutils.ToPtr("new-resource")},
					"nil":     nil,
				},
			},
			NetworkProfile: &generated.NetworkProfile{LoadBalancerProfile: &generated.LoadBalancerProfile{
				ManagedOutboundIPs: &generated.ManagedOutboundIPs{Count: pointerutils.ToPtr(int32(2))},
			}},
			MasterProfile:    &generated.MasterProfile{},
			ApiserverProfile: &generated.APIServerProfile{},
		},
	}}

	(openShiftClusterConverter{}).ToInternal(external, got)

	if got.Identity.PrincipalID != "existing-principal" || got.SystemData.CreatedBy != "existing-creator" {
		t.Fatal("omitted identity or system data was not preserved")
	}
	if got.Properties.ConsoleProfile.URL != "existing-console" || got.Properties.APIServerProfile.URL != "existing-api" || got.Properties.APIServerProfile.IP != "existing-api-ip" {
		t.Fatal("empty read-only URLs or IPs were not preserved")
	}
	if got.Properties.ServicePrincipalProfile.ClientID != "existing-client" || value(got.Properties.ClusterProfile.OIDCIssuer) != "existing-issuer" {
		t.Fatal("omitted optional profile or OIDC issuer was not preserved")
	}
	if got.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count != 2 ||
		got.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs[0].ID != "existing-effective" {
		t.Fatal("load balancer update did not preserve effective IPs while updating managed IPs")
	}
	if got.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["same"].ClientID != "same-client" {
		t.Fatal("unchanged platform identity lost enriched IDs")
	}
	changed := got.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["changed"]
	if changed.ResourceID != "new-resource" || changed.ClientID != "" || changed.ObjectID != "" {
		t.Fatalf("changed platform identity = %#v", changed)
	}
	if _, ok := got.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["nil"]; ok {
		t.Fatal("nil platform identity should be ignored")
	}
	if len(got.Properties.WorkerProfiles) != 0 || len(got.Properties.WorkerProfilesStatus) != 0 || len(got.Properties.IngressProfiles) != 0 {
		t.Fatal("omitted worker and ingress collections should clear existing values")
	}
}

func TestOpenShiftClusterConverterToInternalHandlesNilCollectionEntries(t *testing.T) {
	external := converterExternalCluster()
	external.Identity.UserAssignedIdentities["nil"] = nil
	external.Properties.WorkerProfiles = append(external.Properties.WorkerProfiles, nil)
	external.Properties.WorkerProfilesStatus = append(external.Properties.WorkerProfilesStatus, nil)
	external.Properties.IngressProfiles = append(external.Properties.IngressProfiles, nil)
	external.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = append(
		external.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs, nil)

	got := &api.OpenShiftCluster{}
	(openShiftClusterConverter{}).ToInternal(external, got)

	if !reflect.DeepEqual(got.Identity.UserAssignedIdentities["nil"], api.UserAssignedIdentity{}) {
		t.Fatalf("nil identity entry = %#v", got.Identity.UserAssignedIdentities["nil"])
	}
	if !reflect.DeepEqual(got.Properties.WorkerProfiles[1], api.WorkerProfile{}) ||
		!reflect.DeepEqual(got.Properties.WorkerProfilesStatus[1], api.WorkerProfile{}) ||
		!reflect.DeepEqual(got.Properties.IngressProfiles[1], api.IngressProfile{}) ||
		!reflect.DeepEqual(got.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs[1], api.EffectiveOutboundIP{}) {
		t.Fatal("nil collection entries must convert to zero-valued internal entries")
	}
}

func TestOpenShiftClusterConverterExternalNoReadOnly(t *testing.T) {
	external := converterExternalCluster()
	(openShiftClusterConverter{}).ExternalNoReadOnly(external)

	if external.SystemData != nil || external.Properties.WorkerProfilesStatus != nil || external.Properties.ClusterProfile.OidcIssuer != nil {
		t.Fatal("top-level read-only fields were not cleared")
	}
	if external.Properties.ConsoleProfile.URL != nil || external.Properties.ApiserverProfile.URL != nil || external.Properties.ApiserverProfile.IP != nil ||
		external.Properties.IngressProfiles[0].IP != nil || external.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs != nil {
		t.Fatal("profile read-only fields were not cleared")
	}
	platformIdentity := external.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["operator"]
	if platformIdentity.ClientID != nil || platformIdentity.ObjectID != nil || value(platformIdentity.ResourceID) != "operator-resource" {
		t.Fatalf("platform identity scrubbed incorrectly: %#v", platformIdentity)
	}
	if external.Identity.PrincipalID != nil || external.Identity.TenantID != nil {
		t.Fatal("managed identity read-only fields were not cleared")
	}
	userIdentity := external.Identity.UserAssignedIdentities["identity"]
	if userIdentity.ClientID != nil || userIdentity.PrincipalID != nil {
		t.Fatalf("user identity was not scrubbed: %#v", userIdentity)
	}
	if value(external.Properties.ClusterProfile.Domain) != "domain.example" || value(external.Properties.ApiserverProfile.Visibility) != generated.VisibilityPrivate ||
		value(external.Properties.IngressProfiles[0].Visibility) != generated.VisibilityPublic || value(external.Identity.Type) != generated.ManagedServiceIdentityTypeUserAssigned {
		t.Fatal("writable neighboring fields were modified")
	}
}

func TestOpenShiftClusterConverterExternalNoReadOnlyHandlesNilEntries(t *testing.T) {
	external := converterExternalCluster()
	external.Properties.IngressProfiles = append(external.Properties.IngressProfiles, nil)
	external.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["nil"] = nil
	external.Identity.UserAssignedIdentities["nil"] = nil

	(openShiftClusterConverter{}).ExternalNoReadOnly(external)

	sparse := (openShiftClusterConverter{}).ToExternal(&api.OpenShiftCluster{}).(*OpenShiftCluster)
	(openShiftClusterConverter{}).ExternalNoReadOnly(sparse)
}

func TestOpenShiftClusterConverterJSONCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		internal *api.OpenShiftCluster
		wantJSON string
	}{
		{
			name:     "sparse response matches value-model omission",
			internal: &api.OpenShiftCluster{},
			wantJSON: `{"properties":{"apiserverProfile":{},"clusterProfile":{},"consoleProfile":{},"masterProfile":{},"networkProfile":{}},"systemData":{}}`,
		},
		{
			name: "zero-valued optional structures match omitempty",
			internal: &api.OpenShiftCluster{
				Tags:     map[string]string{},
				Identity: &api.ManagedServiceIdentity{UserAssignedIdentities: map[string]api.UserAssignedIdentity{}},
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{},
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
						PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{},
					},
					NetworkProfile: api.NetworkProfile{LoadBalancerProfile: &api.LoadBalancerProfile{
						ManagedOutboundIPs:   &api.ManagedOutboundIPs{},
						EffectiveOutboundIPs: []api.EffectiveOutboundIP{{}},
					}},
					WorkerProfiles:       []api.WorkerProfile{{}},
					WorkerProfilesStatus: []api.WorkerProfile{{}},
					IngressProfiles:      []api.IngressProfile{{}},
				},
			},
			wantJSON: `{
				"identity":{},
				"properties":{
					"apiserverProfile":{},"clusterProfile":{},"consoleProfile":{},"masterProfile":{},
					"networkProfile":{"loadBalancerProfile":{"managedOutboundIps":{},"effectiveOutboundIps":[{}]}},
					"servicePrincipalProfile":{},"platformWorkloadIdentityProfile":{},
					"workerProfiles":[{}],"workerProfilesStatus":[{}],"ingressProfiles":[{}]
				},
				"systemData":{}
			}`,
		},
		{
			name:     "fully populated response",
			internal: converterInternalCluster(),
			wantJSON: `{
				"id":"resource-id","name":"cluster","type":"Microsoft.RedHatOpenShift/openShiftClusters","location":"eastus",
				"tags":{"tag":"value"},
				"identity":{"type":"UserAssigned","principalId":"principal","tenantId":"tenant","userAssignedIdentities":{"identity":{"clientId":"identity-client","principalId":"identity-principal"}}},
				"properties":{
					"provisioningState":"Succeeded",
					"clusterProfile":{"pullSecret":"pull-secret","domain":"domain.example","version":"4.15.1","resourceGroupId":"cluster-rg","fipsValidatedModules":"Enabled","oidcIssuer":"https://issuer.example"},
					"consoleProfile":{"url":"https://console.example"},
					"servicePrincipalProfile":{"clientId":"sp-client","clientSecret":"sp-secret"},
					"platformWorkloadIdentityProfile":{"upgradeableTo":"4.16.0","platformWorkloadIdentities":{"operator":{"resourceId":"operator-resource","clientId":"operator-client","objectId":"operator-object"}}},
					"networkProfile":{"podCidr":"10.128.0.0/14","serviceCidr":"172.30.0.0/16","outboundType":"Loadbalancer","preconfiguredNSG":"Enabled","loadBalancerProfile":{"managedOutboundIps":{"count":2},"effectiveOutboundIps":[{"id":"effective-ip"}]}},
					"masterProfile":{"vmSize":"Standard_D8s_v3","subnetId":"master-subnet","encryptionAtHost":"Enabled","diskEncryptionSetId":"master-des"},
					"workerProfiles":[{"name":"worker","vmSize":"Standard_D4s_v3","diskSizeGB":128,"subnetId":"worker-subnet","count":3,"encryptionAtHost":"Disabled","diskEncryptionSetId":"worker-des"}],
					"workerProfilesStatus":[{"name":"status","vmSize":"Standard_D4s_v3","diskSizeGB":256,"subnetId":"status-subnet","count":4,"encryptionAtHost":"Enabled","diskEncryptionSetId":"status-des"}],
					"apiserverProfile":{"visibility":"Private","url":"https://api.example","ip":"1.2.3.4"},
					"ingressProfiles":[{"name":"default","visibility":"Public","ip":"5.6.7.8"}]
				},
				"systemData":{"createdBy":"creator","createdByType":"User","createdAt":"2024-01-02T03:04:05Z","lastModifiedBy":"modifier","lastModifiedByType":"Application","lastModifiedAt":"2024-02-03T04:05:06Z"}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			external := (openShiftClusterConverter{}).ToExternal(tt.internal)
			gotJSON, err := json.Marshal(external)
			if err != nil {
				t.Fatal(err)
			}
			assertJSONEqual(t, gotJSON, []byte(tt.wantJSON))
		})
	}
}

func TestOpenShiftClusterConverterRequestJSONCompatibility(t *testing.T) {
	request := []byte(`{
		"id":"resource-id","name":"cluster","type":"Microsoft.RedHatOpenShift/openShiftClusters","location":"eastus",
		"tags":{"tag":"value"},
		"properties":{
			"clusterProfile":{"domain":"domain.example","version":"4.15.1","resourceGroupId":"cluster-rg","fipsValidatedModules":"Enabled"},
			"consoleProfile":{},
			"networkProfile":{"podCidr":"10.128.0.0/14","serviceCidr":"172.30.0.0/16","outboundType":"Loadbalancer","preconfiguredNSG":"Enabled"},
			"masterProfile":{"vmSize":"Standard_D8s_v3","subnetId":"master-subnet","encryptionAtHost":"Enabled","diskEncryptionSetId":"master-des"},
			"workerProfiles":[{"name":"worker","vmSize":"Standard_D4s_v3","diskSizeGB":128,"subnetId":"worker-subnet","count":3,"encryptionAtHost":"Disabled","diskEncryptionSetId":"worker-des"}],
			"apiserverProfile":{"visibility":"Private"},"ingressProfiles":[{"name":"default","visibility":"Public"}]
		}
	}`)

	external := &OpenShiftCluster{}
	if err := json.Unmarshal(request, external); err != nil {
		t.Fatal(err)
	}
	got := &api.OpenShiftCluster{}
	(openShiftClusterConverter{}).ToInternal(external, got)

	if got.ID != "resource-id" || got.Properties.ClusterProfile.Domain != "domain.example" ||
		got.Properties.MasterProfile.VMSize != api.VMSizeStandardD8sV3 || got.Properties.WorkerProfiles[0].DiskSizeGB != 128 ||
		got.Properties.APIServerProfile.Visibility != api.VisibilityPrivate || got.Properties.IngressProfiles[0].Visibility != api.VisibilityPublic {
		t.Fatalf("request JSON converted incorrectly: %#v", got)
	}

	minimumExternal := (openShiftClusterConverter{}).ToExternal(&api.OpenShiftCluster{}).(*OpenShiftCluster)
	(openShiftClusterConverter{}).ExternalNoReadOnly(minimumExternal)
	if err := json.Unmarshal([]byte(`{}`), minimumExternal); err != nil {
		t.Fatal(err)
	}
	minimumInternal := &api.OpenShiftCluster{}
	(openShiftClusterConverter{}).ToInternal(minimumExternal, minimumInternal)
	if !reflect.DeepEqual(minimumInternal, &api.OpenShiftCluster{}) {
		t.Fatalf("minimum request converted to %#v, want zero-valued cluster", minimumInternal)
	}
}

func converterInternalCluster() *api.OpenShiftCluster {
	createdAt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	modifiedAt := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	oidcIssuer := api.OIDCIssuer("https://issuer.example")
	upgradeableTo := api.UpgradeableTo("4.16.0")

	return &api.OpenShiftCluster{
		ID:       "resource-id",
		Name:     "cluster",
		Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
		Location: "eastus",
		Tags:     map[string]string{"tag": "value"},
		SystemData: api.SystemData{
			CreatedBy:          "creator",
			CreatedByType:      api.CreatedByTypeUser,
			CreatedAt:          &createdAt,
			LastModifiedBy:     "modifier",
			LastModifiedByType: api.CreatedByTypeApplication,
			LastModifiedAt:     &modifiedAt,
		},
		Identity: &api.ManagedServiceIdentity{
			Type:        api.ManagedServiceIdentityUserAssigned,
			PrincipalID: "principal",
			TenantID:    "tenant",
			UserAssignedIdentities: map[string]api.UserAssignedIdentity{
				"identity": {ClientID: "identity-client", PrincipalID: "identity-principal"},
			},
		},
		Properties: api.OpenShiftClusterProperties{
			ProvisioningState: api.ProvisioningStateSucceeded,
			ClusterProfile: api.ClusterProfile{
				PullSecret:           api.SecureString("pull-secret"),
				Domain:               "domain.example",
				Version:              "4.15.1",
				ResourceGroupID:      "cluster-rg",
				FipsValidatedModules: api.FipsValidatedModulesEnabled,
				OIDCIssuer:           &oidcIssuer,
			},
			ConsoleProfile: api.ConsoleProfile{URL: "https://console.example"},
			ServicePrincipalProfile: &api.ServicePrincipalProfile{
				ClientID: "sp-client", ClientSecret: api.SecureString("sp-secret"),
			},
			PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
				UpgradeableTo: &upgradeableTo,
				PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
					"operator": {ResourceID: "operator-resource", ClientID: "operator-client", ObjectID: "operator-object"},
				},
			},
			NetworkProfile: api.NetworkProfile{
				PodCIDR:          "10.128.0.0/14",
				ServiceCIDR:      "172.30.0.0/16",
				OutboundType:     api.OutboundTypeLoadbalancer,
				PreconfiguredNSG: api.PreconfiguredNSGEnabled,
				LoadBalancerProfile: &api.LoadBalancerProfile{
					ManagedOutboundIPs:   &api.ManagedOutboundIPs{Count: 2},
					EffectiveOutboundIPs: []api.EffectiveOutboundIP{{ID: "effective-ip"}},
				},
			},
			MasterProfile: api.MasterProfile{
				VMSize: api.VMSizeStandardD8sV3, SubnetID: "master-subnet", EncryptionAtHost: api.EncryptionAtHostEnabled, DiskEncryptionSetID: "master-des",
			},
			WorkerProfiles: []api.WorkerProfile{{
				Name: "worker", VMSize: api.VMSizeStandardD4sV3, DiskSizeGB: 128, SubnetID: "worker-subnet", Count: 3,
				EncryptionAtHost: api.EncryptionAtHostDisabled, DiskEncryptionSetID: "worker-des",
			}},
			WorkerProfilesStatus: []api.WorkerProfile{{
				Name: "status", VMSize: api.VMSizeStandardD4sV3, DiskSizeGB: 256, SubnetID: "status-subnet", Count: 4,
				EncryptionAtHost: api.EncryptionAtHostEnabled, DiskEncryptionSetID: "status-des",
			}},
			APIServerProfile: api.APIServerProfile{Visibility: api.VisibilityPrivate, URL: "https://api.example", IP: "1.2.3.4"},
			IngressProfiles:  []api.IngressProfile{{Name: "default", Visibility: api.VisibilityPublic, IP: "5.6.7.8"}},
		},
	}
}

func converterExternalCluster() *OpenShiftCluster {
	createdAt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	modifiedAt := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	return &OpenShiftCluster{OpenShiftCluster: generated.OpenShiftCluster{
		ID: pointerutils.ToPtr("resource-id"), Name: pointerutils.ToPtr("cluster"), Type: pointerutils.ToPtr("Microsoft.RedHatOpenShift/openShiftClusters"),
		Location: pointerutils.ToPtr("eastus"), Tags: map[string]*string{"tag": pointerutils.ToPtr("value")},
		SystemData: &generated.SystemData{
			CreatedBy: pointerutils.ToPtr("creator"), CreatedByType: pointerutils.ToPtr(generated.CreatedByTypeUser), CreatedAt: &createdAt,
			LastModifiedBy: pointerutils.ToPtr("modifier"), LastModifiedByType: pointerutils.ToPtr(generated.CreatedByTypeApplication), LastModifiedAt: &modifiedAt,
		},
		Identity: &generated.ManagedServiceIdentity{
			Type: pointerutils.ToPtr(generated.ManagedServiceIdentityTypeUserAssigned), PrincipalID: pointerutils.ToPtr("principal"), TenantID: pointerutils.ToPtr("tenant"),
			UserAssignedIdentities: map[string]*generated.UserAssignedIdentity{
				"identity": {ClientID: pointerutils.ToPtr("identity-client"), PrincipalID: pointerutils.ToPtr("identity-principal")},
			},
		},
		Properties: &generated.OpenShiftClusterProperties{
			ProvisioningState: pointerutils.ToPtr(generated.ProvisioningStateSucceeded),
			ClusterProfile: &generated.ClusterProfile{
				PullSecret: pointerutils.ToPtr("pull-secret"), Domain: pointerutils.ToPtr("domain.example"), Version: pointerutils.ToPtr("4.15.1"),
				ResourceGroupID: pointerutils.ToPtr("cluster-rg"), FipsValidatedModules: pointerutils.ToPtr(generated.FipsValidatedModulesEnabled), OidcIssuer: pointerutils.ToPtr("https://issuer.example"),
			},
			ConsoleProfile:          &generated.ConsoleProfile{URL: pointerutils.ToPtr("https://console.example")},
			ServicePrincipalProfile: &generated.ServicePrincipalProfile{ClientID: pointerutils.ToPtr("sp-client"), ClientSecret: pointerutils.ToPtr("sp-secret")},
			PlatformWorkloadIdentityProfile: &generated.PlatformWorkloadIdentityProfile{
				UpgradeableTo: pointerutils.ToPtr("4.16.0"),
				PlatformWorkloadIdentities: map[string]*generated.PlatformWorkloadIdentity{
					"operator": {ResourceID: pointerutils.ToPtr("operator-resource"), ClientID: pointerutils.ToPtr("operator-client"), ObjectID: pointerutils.ToPtr("operator-object")},
				},
			},
			NetworkProfile: &generated.NetworkProfile{
				PodCidr: pointerutils.ToPtr("10.128.0.0/14"), ServiceCidr: pointerutils.ToPtr("172.30.0.0/16"), OutboundType: pointerutils.ToPtr(generated.OutboundTypeLoadbalancer),
				PreconfiguredNSG: pointerutils.ToPtr(generated.PreconfiguredNSGEnabled),
				LoadBalancerProfile: &generated.LoadBalancerProfile{
					ManagedOutboundIPs:   &generated.ManagedOutboundIPs{Count: pointerutils.ToPtr(int32(2))},
					EffectiveOutboundIPs: []*generated.EffectiveOutboundIP{{ID: pointerutils.ToPtr("effective-ip")}},
				},
			},
			MasterProfile: &generated.MasterProfile{
				VMSize: pointerutils.ToPtr("Standard_D8s_v3"), SubnetID: pointerutils.ToPtr("master-subnet"), EncryptionAtHost: pointerutils.ToPtr(generated.EncryptionAtHostEnabled), DiskEncryptionSetID: pointerutils.ToPtr("master-des"),
			},
			WorkerProfiles: []*generated.WorkerProfile{{
				Name: pointerutils.ToPtr("worker"), VMSize: pointerutils.ToPtr("Standard_D4s_v3"), DiskSizeGB: pointerutils.ToPtr(int32(128)), SubnetID: pointerutils.ToPtr("worker-subnet"), Count: pointerutils.ToPtr(int32(3)),
				EncryptionAtHost: pointerutils.ToPtr(generated.EncryptionAtHostDisabled), DiskEncryptionSetID: pointerutils.ToPtr("worker-des"),
			}},
			WorkerProfilesStatus: []*generated.WorkerProfile{{
				Name: pointerutils.ToPtr("status"), VMSize: pointerutils.ToPtr("Standard_D4s_v3"), DiskSizeGB: pointerutils.ToPtr(int32(256)), SubnetID: pointerutils.ToPtr("status-subnet"), Count: pointerutils.ToPtr(int32(4)),
				EncryptionAtHost: pointerutils.ToPtr(generated.EncryptionAtHostEnabled), DiskEncryptionSetID: pointerutils.ToPtr("status-des"),
			}},
			ApiserverProfile: &generated.APIServerProfile{Visibility: pointerutils.ToPtr(generated.VisibilityPrivate), URL: pointerutils.ToPtr("https://api.example"), IP: pointerutils.ToPtr("1.2.3.4")},
			IngressProfiles:  []*generated.IngressProfile{{Name: pointerutils.ToPtr("default"), Visibility: pointerutils.ToPtr(generated.VisibilityPublic), IP: pointerutils.ToPtr("5.6.7.8")}},
		},
	}}
}

func assertJSONEqual(t *testing.T, got, want []byte) {
	t.Helper()
	var gotValue, wantValue interface{}
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("unmarshal actual JSON: %v", err)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("unmarshal expected JSON: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("JSON mismatch\ngot:  %s\nwant: %s", got, want)
	}
}
