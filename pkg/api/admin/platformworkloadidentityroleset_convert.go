package admin

import "github.com/Azure/ARO-RP/pkg/api"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type platformWorkloadIdentityRoleSetConverter struct{}

// platformWorkloadIdentityRoleSetConverter.ToExternal returns a new external representation
// of the internal object, reading from the subset of the internal object's
// fields that appear in the external representation.  ToExternal does not
// modify its argument; there is no pointer aliasing between the passed and
// returned objects.
func (c platformWorkloadIdentityRoleSetConverter) ToExternal(s *api.PlatformWorkloadIdentityRoleSet) interface{} {
	out := &PlatformWorkloadIdentityRoleSet{
		Properties: PlatformWorkloadIdentityRoleSetProperties{
			OpenShiftVersion:              s.Properties.OpenShiftVersion,
			PlatformWorkloadIdentityRoles: make([]PlatformWorkloadIdentityRole, 0, len(s.Properties.PlatformWorkloadIdentityRoles)),
		},
		Name: s.Name,
		Type: s.Type,
	}

	for _, r := range s.Properties.PlatformWorkloadIdentityRoles {
		role := PlatformWorkloadIdentityRole{
			OperatorName:       r.OperatorName,
			RoleDefinitionName: r.RoleDefinitionName,
			RoleDefinitionID:   r.RoleDefinitionID,
			ServiceAccounts:    make([]string, 0, len(r.ServiceAccounts)),
			SecretLocation: SecretLocation{
				Namespace: r.SecretLocation.Namespace,
				Name:      r.SecretLocation.Name,
			},
		}

		role.ServiceAccounts = append(role.ServiceAccounts, r.ServiceAccounts...)

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

	out.Properties.OpenShiftVersion = new.Properties.OpenShiftVersion
	out.Properties.PlatformWorkloadIdentityRoles = make([]api.PlatformWorkloadIdentityRole, 0, len(new.Properties.PlatformWorkloadIdentityRoles))

	for _, r := range new.Properties.PlatformWorkloadIdentityRoles {
		role := api.PlatformWorkloadIdentityRole{
			OperatorName:       r.OperatorName,
			RoleDefinitionName: r.RoleDefinitionName,
			RoleDefinitionID:   r.RoleDefinitionID,
			ServiceAccounts:    make([]string, 0, len(r.ServiceAccounts)),
		}

		role.ServiceAccounts = append(role.ServiceAccounts, r.ServiceAccounts...)

		out.Properties.PlatformWorkloadIdentityRoles = append(out.Properties.PlatformWorkloadIdentityRoles, role)
	}
	out.Name = new.Properties.OpenShiftVersion
	out.Type = api.PlatformWorkloadIdentityRoleSetsType
}
