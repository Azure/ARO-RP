package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestValidatePlatformWorkloadIdentities(t *testing.T) {
	mockMiResourceId := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/not-a-real-group/providers/Microsoft.ManagedIdentity/userAssignedIdentities/not-a-real-mi"

	for _, tt := range []struct {
		test          string
		pwi           map[string]api.PlatformWorkloadIdentity
		version       string
		upgradeableTo *api.UpgradeableTo
		wantErr       string
	}{
		{
			test:    "Success - Valid platform workload identities provided",
			version: defaultVersion,
			pwi: map[string]api.PlatformWorkloadIdentity{
				"file-csi-driver":          {ResourceID: mockMiResourceId},
				"cloud-controller-manager": {ResourceID: mockMiResourceId},
				"ingress":                  {ResourceID: mockMiResourceId},
				"image-registry":           {ResourceID: mockMiResourceId},
				"machine-api":              {ResourceID: mockMiResourceId},
				"cloud-network-config":     {ResourceID: mockMiResourceId},
				"aro-operator":             {ResourceID: mockMiResourceId},
				"disk-csi-driver":          {ResourceID: mockMiResourceId},
			},
		},
		{
			test:    "Success - Valid platform workload identities provided with UpgradeableTo",
			version: defaultVersion,
			pwi: map[string]api.PlatformWorkloadIdentity{
				"file-csi-driver":          {ResourceID: mockMiResourceId},
				"cloud-controller-manager": {ResourceID: mockMiResourceId},
				"ingress":                  {ResourceID: mockMiResourceId},
				"image-registry":           {ResourceID: mockMiResourceId},
				"machine-api":              {ResourceID: mockMiResourceId},
				"cloud-network-config":     {ResourceID: mockMiResourceId},
				"aro-operator":             {ResourceID: mockMiResourceId},
				"disk-csi-driver":          {ResourceID: mockMiResourceId},
				"extra-new-operator":       {ResourceID: mockMiResourceId},
			},
			upgradeableTo: pointerutils.ToPtr(api.UpgradeableTo(getMIWIUpgradeableToVersion().String())),
		},
		{
			test:    "Success - Valid platform workload identities provided with UpgradeableTo smaller than current version",
			version: defaultVersion,
			pwi: map[string]api.PlatformWorkloadIdentity{
				"file-csi-driver":          {ResourceID: mockMiResourceId},
				"cloud-controller-manager": {ResourceID: mockMiResourceId},
				"ingress":                  {ResourceID: mockMiResourceId},
				"image-registry":           {ResourceID: mockMiResourceId},
				"machine-api":              {ResourceID: mockMiResourceId},
				"cloud-network-config":     {ResourceID: mockMiResourceId},
				"aro-operator":             {ResourceID: mockMiResourceId},
				"disk-csi-driver":          {ResourceID: mockMiResourceId},
			},
			upgradeableTo: pointerutils.ToPtr(api.UpgradeableTo("4.10.25")),
		},
		{
			test:    "Fail - Not a MIWI Cluster",
			version: defaultVersion,
			wantErr: "PopulatePlatformWorkloadIdentityRolesByVersionUsingRoleSets called for a Cluster Service Principal cluster",
		},
		{
			test:    "Fail - Invalid Cluster Version",
			version: "4.1052",
			pwi: map[string]api.PlatformWorkloadIdentity{
				"file-csi-driver":          {ResourceID: mockMiResourceId},
				"cloud-controller-manager": {ResourceID: mockMiResourceId},
				"ingress":                  {ResourceID: mockMiResourceId},
				"image-registry":           {ResourceID: mockMiResourceId},
				"machine-api":              {ResourceID: mockMiResourceId},
				"cloud-network-config":     {ResourceID: mockMiResourceId},
				"aro-operator":             {ResourceID: mockMiResourceId},
				"disk-csi-driver":          {ResourceID: mockMiResourceId},
			},
			wantErr: `could not parse version "4.1052"`,
		},
		{
			test:    "Fail - Invalid upgradeableTo version",
			version: defaultVersion,
			pwi: map[string]api.PlatformWorkloadIdentity{
				"file-csi-driver":          {ResourceID: mockMiResourceId},
				"cloud-controller-manager": {ResourceID: mockMiResourceId},
				"ingress":                  {ResourceID: mockMiResourceId},
				"image-registry":           {ResourceID: mockMiResourceId},
				"machine-api":              {ResourceID: mockMiResourceId},
				"cloud-network-config":     {ResourceID: mockMiResourceId},
				"aro-operator":             {ResourceID: mockMiResourceId},
				"disk-csi-driver":          {ResourceID: mockMiResourceId},
			},
			upgradeableTo: pointerutils.ToPtr(api.UpgradeableTo("4.1052")),
			wantErr:       `could not parse version "4.1052"`,
		},
		{
			test:    "Fail - No roleset exists for the older version",
			version: "4.10.14",
			pwi: map[string]api.PlatformWorkloadIdentity{
				"file-csi-driver":          {ResourceID: mockMiResourceId},
				"cloud-controller-manager": {ResourceID: mockMiResourceId},
				"ingress":                  {ResourceID: mockMiResourceId},
				"image-registry":           {ResourceID: mockMiResourceId},
				"machine-api":              {ResourceID: mockMiResourceId},
				"cloud-network-config":     {ResourceID: mockMiResourceId},
				"aro-operator":             {ResourceID: mockMiResourceId},
				"disk-csi-driver":          {ResourceID: mockMiResourceId},
			},
			wantErr: "400: InvalidParameter: : No PlatformWorkloadIdentityRoleSet found for the requested or upgradeable OpenShift minor version '4.10'. Please retry with different OpenShift version, and if the issue persists, raise an Azure support ticket",
		},
		{
			test:    "Fail - Unexpected platform workload identity provided",
			version: defaultVersion,
			pwi: map[string]api.PlatformWorkloadIdentity{
				"file-csi-driver":          {ResourceID: mockMiResourceId},
				"cloud-controller-manager": {ResourceID: mockMiResourceId},
				"ingress":                  {ResourceID: mockMiResourceId},
				"image-registry":           {ResourceID: mockMiResourceId},
				"machine-api":              {ResourceID: mockMiResourceId},
				"cloud-network-config":     {ResourceID: mockMiResourceId},
				"aro-operator":             {ResourceID: mockMiResourceId},
				"unexpected-identity":      {ResourceID: mockMiResourceId},
			},
			wantErr: unexpectedWorkloadIdentitiesError,
		},
		{
			test:    "Fail - Missing platform workload identity",
			version: defaultVersion,
			pwi: map[string]api.PlatformWorkloadIdentity{
				"file-csi-driver":          {ResourceID: mockMiResourceId},
				"cloud-controller-manager": {ResourceID: mockMiResourceId},
				"ingress":                  {ResourceID: mockMiResourceId},
				"image-registry":           {ResourceID: mockMiResourceId},
				"machine-api":              {ResourceID: mockMiResourceId},
				"cloud-network-config":     {ResourceID: mockMiResourceId},
				"aro-operator":             {ResourceID: mockMiResourceId},
			},
			wantErr: unexpectedWorkloadIdentitiesError,
		},
		{
			test:    "Fail - Extra platform workload identity provided",
			version: defaultVersion,
			pwi: map[string]api.PlatformWorkloadIdentity{
				"file-csi-driver":          {ResourceID: mockMiResourceId},
				"cloud-controller-manager": {ResourceID: mockMiResourceId},
				"ingress":                  {ResourceID: mockMiResourceId},
				"image-registry":           {ResourceID: mockMiResourceId},
				"machine-api":              {ResourceID: mockMiResourceId},
				"cloud-network-config":     {ResourceID: mockMiResourceId},
				"aro-operator":             {ResourceID: mockMiResourceId},
				"disk-csi-driver":          {ResourceID: mockMiResourceId},
				"extra-identity":           {ResourceID: mockMiResourceId},
			},
			wantErr: unexpectedWorkloadIdentitiesError,
		},
	} {
		t.Run(tt.test, func(t *testing.T) {
			f := frontend{
				availablePlatformWorkloadIdentityRoleSets: getPlatformWorkloadIdentityRolesChangeFeed(),
			}

			oc := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Version: tt.version,
					},
				},
			}

			if tt.pwi != nil {
				oc.Properties.PlatformWorkloadIdentityProfile = &api.PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: tt.pwi,
					UpgradeableTo:              tt.upgradeableTo,
				}
			}

			err := f.validatePlatformWorkloadIdentities(oc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
