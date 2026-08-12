package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

type platformWorkloadIdentityRoleSetConverter struct{}

// platformWorkloadIdentityRoleSetConverter.ToExternal returns a new external representation
// of the internal object, reading from the subset of the internal object's
// fields that appear in the external representation.  ToExternal does not
// modify its argument; there is no pointer aliasing between the passed and
// returned objects.
func (c platformWorkloadIdentityRoleSetConverter) ToExternal(s *api.PlatformWorkloadIdentityRoleSet) interface{} {
	out := &PlatformWorkloadIdentityRoleSet{
		PlatformWorkloadIdentityRoleSet: generated.PlatformWorkloadIdentityRoleSet{
			Properties: &generated.PlatformWorkloadIdentityRoleSetProperties{},
		},
	}
	if s.ID != "" {
		out.ID = pointerutils.ToPtr(s.ID)
	}
	if s.Name != "" {
		out.Name = pointerutils.ToPtr(s.Name)
	}
	if s.Type != "" {
		out.Type = pointerutils.ToPtr(s.Type)
	}
	if s.Properties.OpenShiftVersion != "" {
		out.Properties.OpenShiftVersion = pointerutils.ToPtr(s.Properties.OpenShiftVersion)
	}
	if len(s.Properties.PlatformWorkloadIdentityRoles) > 0 {
		out.Properties.PlatformWorkloadIdentityRoles = make([]*generated.PlatformWorkloadIdentityRole, 0, len(s.Properties.PlatformWorkloadIdentityRoles))
	}

	for _, r := range s.Properties.PlatformWorkloadIdentityRoles {
		role := &generated.PlatformWorkloadIdentityRole{}
		if r.OperatorName != "" {
			role.OperatorName = pointerutils.ToPtr(r.OperatorName)
		}
		if r.RoleDefinitionName != "" {
			role.RoleDefinitionName = pointerutils.ToPtr(r.RoleDefinitionName)
		}
		if r.RoleDefinitionID != "" {
			role.RoleDefinitionID = pointerutils.ToPtr(r.RoleDefinitionID)
		}
		out.Properties.PlatformWorkloadIdentityRoles = append(out.Properties.PlatformWorkloadIdentityRoles, role)
	}

	return out
}

// ToExternalList returns a slice of external representations of the internal
// objects
func (c platformWorkloadIdentityRoleSetConverter) ToExternalList(sets []*api.PlatformWorkloadIdentityRoleSet) interface{} {
	l := &PlatformWorkloadIdentityRoleSetList{
		PlatformWorkloadIdentityRoleSets: make([]*PlatformWorkloadIdentityRoleSet, 0, len(sets)),
	}

	for _, set := range sets {
		l.PlatformWorkloadIdentityRoleSets = append(l.PlatformWorkloadIdentityRoleSets, c.ToExternal(set).(*PlatformWorkloadIdentityRoleSet))
	}

	return l
}

// ToInternal overwrites in place a pre-existing internal object, setting (only)
// all mapped fields from the external representation. ToInternal modifies its
// argument; there is no pointer aliasing between the passed and returned
// objects
func (c platformWorkloadIdentityRoleSetConverter) ToInternal(_new interface{}, out *api.PlatformWorkloadIdentityRoleSet) {
	new := _new.(*PlatformWorkloadIdentityRoleSet)

	out.Properties.OpenShiftVersion = value(new.Properties.OpenShiftVersion)
	out.Properties.PlatformWorkloadIdentityRoles = make([]api.PlatformWorkloadIdentityRole, 0, len(new.Properties.PlatformWorkloadIdentityRoles))

	for _, r := range new.Properties.PlatformWorkloadIdentityRoles {
		if r == nil {
			continue
		}
		role := api.PlatformWorkloadIdentityRole{
			OperatorName:       value(r.OperatorName),
			RoleDefinitionName: value(r.RoleDefinitionName),
			RoleDefinitionID:   value(r.RoleDefinitionID),
		}
		out.Properties.PlatformWorkloadIdentityRoles = append(out.Properties.PlatformWorkloadIdentityRoles, role)
	}
	out.Name = value(new.Properties.OpenShiftVersion)
	out.Type = api.PlatformWorkloadIdentityRoleSetsType
}
