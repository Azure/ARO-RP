package v20220904

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type clusterManagerConfigurationConverter struct{}

func (c clusterManagerConfigurationConverter) ToExternal(ocm *api.ClusterManagerConfiguration) (interface{}, error) {
	out := new(ClusterManagerConfiguration)
	out.ID = ocm.ID
	out.Name = ocm.Name
	out.Properties.Resources = string(ocm.Properties.Resources)
	return out, nil
}

func (c clusterManagerConfigurationConverter) SyncSetToExternal(ocm *api.SyncSet) interface{} {
	out := new(SyncSet)
	out.proxyResource = true
	out.ID = ocm.ID
	out.Name = ocm.Name
	out.Type = ocm.Type
	out.Properties.Resources = ocm.Properties.Resources
	return out
}

func (c clusterManagerConfigurationConverter) MachinePoolToExternal(ocm *api.MachinePool) interface{} {
	out := new(MachinePool)
	out.proxyResource = true
	out.ID = ocm.ID
	out.Name = ocm.Name
	out.Type = ocm.Type
	out.Properties.Resources = ocm.Properties.Resources
	return out
}

func (c clusterManagerConfigurationConverter) SyncIdentityProviderToExternal(ocm *api.SyncIdentityProvider) interface{} {
	out := new(SyncIdentityProvider)
	out.proxyResource = true
	out.ID = ocm.ID
	out.Name = ocm.Name
	out.Type = ocm.Type
	out.Properties.Resources = ocm.Properties.Resources
	return out
}

func (c clusterManagerConfigurationConverter) SecretToExternal(ocm *api.Secret) interface{} {
	out := new(Secret)
	out.proxyResource = true
	out.ID = ocm.ID
	out.Name = ocm.Name
	out.Type = ocm.Type
	out.Properties.SecretResources = ""
	return out
}

func (c clusterManagerConfigurationConverter) ToInternal(_ocm interface{}, out *api.ClusterManagerConfiguration) error {
	ocm := _ocm.(*api.ClusterManagerConfiguration)
	out.ID = ocm.ID
	return nil
}

// ToExternalList returns a slice of external representations of the internal objects
func (c clusterManagerConfigurationConverter) ToExternalList(ocms []*api.ClusterManagerConfiguration, nextLink string) (interface{}, error) {
	l := &ClusterManagerConfigurationList{
		ClusterManagerConfigurations: make([]*ClusterManagerConfiguration, 0, len(ocms)),
		NextLink:                     nextLink,
	}

	for _, ocm := range ocms {
		c, err := c.ToExternal(ocm)
		if err != nil {
			return nil, err
		}
		l.ClusterManagerConfigurations = append(l.ClusterManagerConfigurations, c.(*ClusterManagerConfiguration))
	}

	return l, nil
}
