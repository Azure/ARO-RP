package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type syncSetConverter struct{}

func (c syncSetConverter) ToExternal(ss *api.SyncSet) interface{} {
	out := new(SyncSet)
	out.proxyResource = true
	out.ID = ss.ID
	out.Name = ss.Name
	out.Type = ss.Type
	out.Properties.Resources = ss.Properties.Resources
	return out
}

func (c syncSetConverter) ToInternal(_ss interface{}, out *api.SyncSet) {
	ocm := _ss.(*api.SyncSet)
	out.ID = ocm.ID
}

// ToExternalList returns a slice of external representations of the internal objects
func (c syncSetConverter) ToExternalList(ss []*api.SyncSet) interface{} {
	l := &SyncSetList{
		SyncSets: make([]*SyncSet, 0, len(ss)),
	}

	for _, syncset := range ss {
		c := c.ToExternal(syncset)
		l.SyncSets = append(l.SyncSets, c.(*SyncSet))
	}

	return l
}
