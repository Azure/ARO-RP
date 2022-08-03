package v20220904

import (
	"encoding/json"

	"github.com/Azure/ARO-RP/pkg/api"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type clusterManagerConverter struct{}

func (c *clusterManagerConverter) ToExternal(ocm *api.ClusterManagerConfiguration) (interface{}, error) {
	out := new(ClusterManagerConfiguration)
	out.ID = ocm.ID
	out.Kind = ocm.Kind
	var data interface{}
	err := json.Unmarshal(ocm.Resources, &data)
	if err != nil {
		return nil, err
	}
	out.Resources = data
	return out, nil
}

func (c *clusterManagerConverter) ToInternal(_ocm interface{}, out *api.ClusterManagerConfiguration) error {
	ocm := _ocm.(*api.ClusterManagerConfiguration)
	out.ID = ocm.ID

	return nil
}

// ToExternalList returns a slice of external representations of the internal objects
func (c *clusterManagerConverter) ToExternalList(ocms []*api.ClusterManagerConfiguration, nextLink string) (interface{}, error) {
	l := &ClusterManagerConfigurationsList{
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
