package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (m *manager) getOpenShiftVersionFromVersion(ctx context.Context) (*api.OpenShiftVersion, error) {
	requestedInstallVersion := &m.doc.OpenShiftCluster.Properties.ClusterProfile.Version

	docs, err := m.dbOpenShiftVersions.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	activeOpenShiftVersions := make([]*api.OpenShiftVersion, 0)
	for _, doc := range docs.OpenShiftVersionDocuments {
		if doc.OpenShiftVersion.Enabled {
			activeOpenShiftVersions = append(activeOpenShiftVersions, doc.OpenShiftVersion)
		}
	}

	// when we have no OpenShiftVersion entries in CosmoDB, default to building one using the InstallStream
	if len(activeOpenShiftVersions) == 0 {
		if *requestedInstallVersion != version.InstallStream.Version.String() {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.clusterProfile.version", "The provided OpenShift version '%s' is not supported.", *requestedInstallVersion)
		}

		installerPullSpec := m.env.LiveConfig().DefaultInstallerPullSpecOverride(ctx)
		if len(installerPullSpec) == 0 {
			// If no ENV var was set as an override, then use the default image name:tag format we build in the ARO-Installer build & push pipeline
			installerPullSpec = fmt.Sprintf("%s/aro-installer:release-%d.%d", m.env.ACRDomain(), version.InstallStream.Version.V[0], version.InstallStream.Version.V[1])
		}

		openshiftPullSpec := version.InstallStream.PullSpec
		if m.installViaHive {
			openshiftPullSpec = strings.Replace(openshiftPullSpec, "quay.io", m.env.ACRDomain(), 1)
		}

		return &api.OpenShiftVersion{
			Version:           version.InstallStream.Version.String(),
			OpenShiftPullspec: openshiftPullSpec,
			InstallerPullspec: installerPullSpec,
			Enabled:           true,
		}, nil
	}

	for _, active := range activeOpenShiftVersions {
		if *requestedInstallVersion == active.Version {
			return active, nil
		}
	}

	return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.clusterProfile.version", "The requested OpenShift version '%s' is not supported.", *requestedInstallVersion)
}
