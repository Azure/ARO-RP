package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type openShiftVersionConverter struct{}

// openShiftVersionConverter.ToExternal returns a new external representation
// of the internal object, reading from the subset of the internal object's
// fields that appear in the external representation.  ToExternal does not
// modify its argument; there is no pointer aliasing between the passed and
// returned objects.
func (*openShiftVersionConverter) ToExternal(v *api.OpenShiftVersion) interface{} {
	out := &OpenShiftVersion{
		Version:           v.Version,
		OpenShiftPullspec: v.OpenShiftPullspec,
		InstallerPullspec: v.InstallerPullspec,
		Enabled:           v.Enabled,
	}

	return out
}

// ToExternalList returns a slice of external representations of the internal
// objects
func (c *openShiftVersionConverter) ToExternalList(vers []*api.OpenShiftVersion) interface{} {
	l := &OpenShiftVersionList{
		OpenShiftVersions: make([]*OpenShiftVersion, 0, len(vers)),
	}

	for _, ver := range vers {
		l.OpenShiftVersions = append(l.OpenShiftVersions, c.ToExternal(ver).(*OpenShiftVersion))
	}

	return l
}

// ToInternal overwrites in place a pre-existing internal object, setting (only)
// all mapped fields from the external representation. ToInternal modifies its
// argument; there is no pointer aliasing between the passed and returned
// objects
func (c *openShiftVersionConverter) ToInternal(_new interface{}, out *api.OpenShiftVersion) {
	new := _new.(*OpenShiftVersion)

	out.Enabled = new.Enabled
	out.InstallerPullspec = new.InstallerPullspec
	out.OpenShiftPullspec = new.OpenShiftPullspec
	out.Version = new.Version
}
