package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type syncIdentityProviderConverter struct{}

func (c syncIdentityProviderConverter) ToExternal(sip *api.SyncIdentityProvider) interface{} {
	out := new(SyncIdentityProvider)
	out.proxyResource = true
	out.ID = sip.ID
	out.Name = sip.Name
	out.Type = sip.Type
	out.Properties.Resources = sip.Properties.Resources
	return out
}

func (c syncIdentityProviderConverter) ToInternal(_sip interface{}, out *api.SyncIdentityProvider) {
	ocm := _sip.(*api.SyncIdentityProvider)
	out.ID = ocm.ID
}

// ToExternalList returns a slice of external representations of the internal objects
func (c syncIdentityProviderConverter) ToExternalList(sip []*api.SyncIdentityProvider) interface{} {
	l := &SyncIdentityProviderList{
		SyncIdentityProviders: make([]*SyncIdentityProvider, 0, len(sip)),
	}

	for _, syncidentityproviders := range sip {
		c := c.ToExternal(syncidentityproviders)
		l.SyncIdentityProviders = append(l.SyncIdentityProviders, c.(*SyncIdentityProvider))
	}

	return l
}
