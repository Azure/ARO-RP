package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type secretConverter struct{}

func (c secretConverter) ToExternal(s *api.Secret) interface{} {
	out := new(Secret)
	out.proxyResource = true
	out.ID = s.ID
	out.Name = s.Name
	out.Type = s.Type
	return out
}

func (c secretConverter) ToInternal(_s interface{}, out *api.Secret) {
	ocm := _s.(*api.Secret)
	out.ID = ocm.ID
}

// ToExternalList returns a slice of external representations of the internal objects
func (c secretConverter) ToExternalList(s []*api.Secret) interface{} {
	l := &SecretList{
		Secrets: make([]*Secret, 0, len(s)),
	}

	for _, secrets := range s {
		c := c.ToExternal(secrets)
		l.Secrets = append(l.Secrets, c.(*Secret))
	}

	return l
}
