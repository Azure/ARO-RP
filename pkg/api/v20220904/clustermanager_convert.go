package v20220904

import (
	"encoding/json"

	"github.com/Azure/ARO-RP/pkg/api"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type clusterManagerConfigurationConverter struct{}

func (c clusterManagerConfigurationConverter) ToExternal(ocm *api.ClusterManagerConfiguration) (interface{}, error) {
	out := new(ClusterManagerConfiguration)
	out.ID = ocm.ID
	var data interface{}
	err := json.Unmarshal(ocm.Properties.Resources, &data)
	if err != nil {
		return nil, err
	}
	out.Properties.Resources = data
	return out, nil
}

func (c clusterManagerConfigurationConverter) SyncSetToExternal(ocm *api.SyncSet) (interface{}, error) {
	out := new(SyncSet)
	out.ID = ocm.ID
	out.Name = ocm.Name
	out.Type = ocm.Type
	out.Properties = SyncSetProperties{
		ClusterResourceId: ocm.Properties.ClusterResourceId,
		Resources:         ocm.Properties.Resources,
	}

	return out, nil
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
