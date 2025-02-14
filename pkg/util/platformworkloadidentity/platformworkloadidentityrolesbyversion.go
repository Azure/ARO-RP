package platformworkloadidentity

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// PlatformWorkloadIdentityRolesByVersion is the interface that validates and obtains the version from an PlatformWorkloadIdentityRoleSetDocument.
type PlatformWorkloadIdentityRolesByVersion interface {
	GetPlatformWorkloadIdentityRolesByRoleName() map[string]api.PlatformWorkloadIdentityRole
	PopulatePlatformWorkloadIdentityRolesByVersion(ctx context.Context, oc *api.OpenShiftCluster, dbPlatformWorkloadIdentityRoleSets database.PlatformWorkloadIdentityRoleSets) error
	PopulatePlatformWorkloadIdentityRolesByVersionUsingRoleSets(oc *api.OpenShiftCluster, platformWorkloadIdentityRoleSets []*api.PlatformWorkloadIdentityRoleSet) error
	MatchesPlatformWorkloadIdentityRoles(oc *api.OpenShiftCluster) bool
}

// platformWorkloadIdentityRolesByVersionService is the default implementation of the PlatformWorkloadIdentityRolesByVersion interface.
type PlatformWorkloadIdentityRolesByVersionService struct {
	platformWorkloadIdentityRoles []api.PlatformWorkloadIdentityRole
}

var _ PlatformWorkloadIdentityRolesByVersion = &PlatformWorkloadIdentityRolesByVersionService{}

func NewPlatformWorkloadIdentityRolesByVersionService() *PlatformWorkloadIdentityRolesByVersionService {
	return &PlatformWorkloadIdentityRolesByVersionService{
		platformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{},
	}
}

// PopulatePlatformWorkloadIdentityRolesByVersion aims to populate platformWorkloadIdentityRoles for current OpenShift minor version and also for UpgradeableTo minor version if provided and is greater than the current version
func (service *PlatformWorkloadIdentityRolesByVersionService) PopulatePlatformWorkloadIdentityRolesByVersion(ctx context.Context, oc *api.OpenShiftCluster, dbPlatformWorkloadIdentityRoleSets database.PlatformWorkloadIdentityRoleSets) error {
	if !oc.UsesWorkloadIdentity() {
		return fmt.Errorf("PopulatePlatformWorkloadIdentityRolesByVersion called for a Cluster Service Principal cluster")
	}

	docs, err := dbPlatformWorkloadIdentityRoleSets.ListAll(ctx)
	if err != nil {
		return err
	}

	platformWorkloadIdentityRoleSets := []*api.PlatformWorkloadIdentityRoleSet{}

	for _, doc := range docs.PlatformWorkloadIdentityRoleSetDocuments {
		platformWorkloadIdentityRoleSets = append(platformWorkloadIdentityRoleSets, doc.PlatformWorkloadIdentityRoleSet)
	}

	return service.PopulatePlatformWorkloadIdentityRolesByVersionUsingRoleSets(oc, platformWorkloadIdentityRoleSets)
}

// PopulatePlatformWorkloadIdentityRolesByVersionUsingRoleSets aims to populate platformWorkloadIdentityRoles for current OpenShift minor version and also for UpgradeableTo minor version if provided and is greater than the current version
// Rather than listing platformWorkloadIdentityRoleSets from db it takes platformWorkloadIdentityRoleSets as an argument
func (service *PlatformWorkloadIdentityRolesByVersionService) PopulatePlatformWorkloadIdentityRolesByVersionUsingRoleSets(oc *api.OpenShiftCluster, platformWorkloadIdentityRoleSets []*api.PlatformWorkloadIdentityRoleSet) error {
	if !oc.UsesWorkloadIdentity() {
		return fmt.Errorf("PopulatePlatformWorkloadIdentityRolesByVersionUsingRoleSets called for a Cluster Service Principal cluster")
	}

	currentOpenShiftVersion, err := version.ParseVersion(oc.Properties.ClusterProfile.Version)
	if err != nil {
		return err
	}
	currentMinorVersion := currentOpenShiftVersion.MinorVersion()
	requiredMinorVersions := map[string]bool{currentMinorVersion: false}

	if oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo != nil {
		upgradeableVersion, err := version.ParseVersion(string(*oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo))
		if err != nil {
			return err
		}
		upgradeableMinorVersion := upgradeableVersion.MinorVersion()
		if currentMinorVersion != upgradeableMinorVersion && currentOpenShiftVersion.Lt(upgradeableVersion) {
			requiredMinorVersions[upgradeableMinorVersion] = false
		}
	}

	for _, platformWorkloadIdentityRoleSet := range platformWorkloadIdentityRoleSets {
		for version := range requiredMinorVersions {
			if version == platformWorkloadIdentityRoleSet.Properties.OpenShiftVersion {
				service.platformWorkloadIdentityRoles = append(service.platformWorkloadIdentityRoles, platformWorkloadIdentityRoleSet.Properties.PlatformWorkloadIdentityRoles...)
				requiredMinorVersions[version] = true
			}
		}
	}

	for version, exists := range requiredMinorVersions {
		if !exists {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("No PlatformWorkloadIdentityRoleSet found for the requested or upgradeable OpenShift minor version '%s'. Please retry with different OpenShift version, and if the issue persists, raise an Azure support ticket", version))
		}
	}

	return nil
}

func (service *PlatformWorkloadIdentityRolesByVersionService) GetPlatformWorkloadIdentityRolesByRoleName() map[string]api.PlatformWorkloadIdentityRole {
	platformWorkloadIdentityRolesByRoleName := map[string]api.PlatformWorkloadIdentityRole{}
	for _, role := range service.platformWorkloadIdentityRoles {
		platformWorkloadIdentityRolesByRoleName[role.OperatorName] = role
	}
	return platformWorkloadIdentityRolesByRoleName
}

// Check if required platform identity are provided in cluster doc by assessing
// Condition 1: Platform Workload Identities and Platform Workload Identity Roles should be equal in length
// Condition 2: All the keys in Platform Workload Identities map should be present in Platform Workload Identity Roles
// These conditions also assures if Platform Workload Identities contains all the keys present in Platform Workload Identity Roles
func (service *PlatformWorkloadIdentityRolesByVersionService) MatchesPlatformWorkloadIdentityRoles(oc *api.OpenShiftCluster) bool {
	platformWorkloadIdentityRolesByRoleName := service.GetPlatformWorkloadIdentityRolesByRoleName()
	platformIdentities := oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities
	if len(platformIdentities) != len(platformWorkloadIdentityRolesByRoleName) {
		return false
	}

	for k := range platformIdentities {
		_, exists := platformWorkloadIdentityRolesByRoleName[k]
		if !exists {
			return false
		}
	}

	return true
}

func GetPlatformWorkloadIdentityMismatchError(oc *api.OpenShiftCluster, platformWorkloadIdentityRolesByRoleName map[string]api.PlatformWorkloadIdentityRole) error {
	if !oc.UsesWorkloadIdentity() {
		return fmt.Errorf("GetPlatformWorkloadIdentityMismatchError called for a Cluster Service Principal cluster")
	}
	requiredOperatorIdentities := []string{}
	for _, role := range platformWorkloadIdentityRolesByRoleName {
		requiredOperatorIdentities = append(requiredOperatorIdentities, role.OperatorName)
	}
	sort.Strings(requiredOperatorIdentities)
	currentOpenShiftVersion, err := version.ParseVersion(oc.Properties.ClusterProfile.Version)
	if err != nil {
		return err
	}
	currentMinorVersion := currentOpenShiftVersion.MinorVersion()
	v := currentMinorVersion
	if oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo != nil {
		upgradeableVersion, err := version.ParseVersion(string(*oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo))
		if err != nil {
			return err
		}
		upgradeableMinorVersion := upgradeableVersion.MinorVersion()
		if currentMinorVersion != upgradeableMinorVersion && currentOpenShiftVersion.Lt(upgradeableVersion) {
			v = fmt.Sprintf("%s or %s", v, upgradeableMinorVersion)
		}
	}
	return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePlatformWorkloadIdentityMismatch,
		"properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities", fmt.Sprintf("There's a mismatch between the required and expected set of platform workload identities for the requested OpenShift minor version '%s'. The required platform workload identities are '%v'", v, requiredOperatorIdentities))
}
