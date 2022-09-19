package v20220904

import "github.com/Azure/ARO-RP/pkg/api"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type installVersionsConverter struct{}

func (i installVersionsConverter) ToExternal(installVersion *api.InstallVersion) interface{} {
	out := new(InstallVersion)

	// TODO - we set this value to true to use the bool so the linter does not get mad - it has no use
	// outside of the typerwalker looking for the existence of it, refactor swagger/typewalker to not need this
	out.proxyResource = true

	out.ID = installVersion.ID
	out.Name = installVersion.Name
	out.Properties = InstallVersionProperties(installVersion.Properties)

	return out
}

func (i installVersionsConverter) ToExternalList(installVersions []*api.InstallVersion, nextLink string) interface{} {
	l := &InstallVersionList{
		InstallVersions: make([]*InstallVersion, 0, len(installVersions)),
		NextLink:        nextLink,
	}

	for _, v := range installVersions {
		ver := i.ToExternal(v)
		l.InstallVersions = append(l.InstallVersions, ver.(*InstallVersion))
	}

	return l
}
