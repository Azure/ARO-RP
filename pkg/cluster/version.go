package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// openShiftClusterDocumentVersioner is the interface that validates and obtains the version from an OpenShiftClusterDocument.
type openShiftClusterDocumentVersioner interface {

	// Get validates and obtains the OpenShift version of the OpenShiftClusterDocument doc using dbOpenShiftVersions, env and installViaHive parameters.
	Get(ctx context.Context, doc *api.OpenShiftClusterDocument, dbOpenShiftVersions database.OpenShiftVersions, env env.Interface, installViaHive bool) (*api.OpenShiftVersion, error)
}

// openShiftClusterDocumentVersionerService is the default implementation of the openShiftClusterDocumentVersioner interface.
type openShiftClusterDocumentVersionerService struct{}

func (service *openShiftClusterDocumentVersionerService) Get(ctx context.Context, doc *api.OpenShiftClusterDocument, dbOpenShiftVersions database.OpenShiftVersions, env env.Interface, installViaHive bool) (*api.OpenShiftVersion, error) {
	requestedInstallVersion := doc.OpenShiftCluster.Properties.ClusterProfile.Version

	docs, err := dbOpenShiftVersions.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	activeOpenShiftVersions := make([]*api.OpenShiftVersion, 0)
	for _, doc := range docs.OpenShiftVersionDocuments {
		if doc.OpenShiftVersion.Properties.Enabled {
			activeOpenShiftVersions = append(activeOpenShiftVersions, doc.OpenShiftVersion)
		}
	}

	errUnsupportedVersion := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.clusterProfile.version", "The requested OpenShift version '%s' is not supported.", requestedInstallVersion)

	// when we have no OpenShiftVersion entries in CosmoDB, default to building one using the InstallStream
	if len(activeOpenShiftVersions) == 0 {
		if requestedInstallVersion != version.InstallStream.Version.String() {
			return nil, errUnsupportedVersion
		}

		installerPullSpec := env.LiveConfig().DefaultInstallerPullSpecOverride(ctx)
		if installerPullSpec == "" {
			// If no ENV var was set as an override, then use the default image name:tag format we build in the ARO-Installer build & push pipeline
			installerPullSpec = fmt.Sprintf("%s/aro-installer:release-%d.%d", env.ACRDomain(), version.InstallStream.Version.V[0], version.InstallStream.Version.V[1])
		}

		openshiftPullSpec := version.InstallStream.PullSpec
		if installViaHive {
			openshiftPullSpec = strings.Replace(openshiftPullSpec, "quay.io", env.ACRDomain(), 1)
		}

		return &api.OpenShiftVersion{
			Properties: api.OpenShiftVersionProperties{
				Version:           version.InstallStream.Version.String(),
				OpenShiftPullspec: openshiftPullSpec,
				InstallerPullspec: installerPullSpec,
				Enabled:           true,
			}}, nil
	}

	for _, active := range activeOpenShiftVersions {
		if requestedInstallVersion == active.Properties.Version {
			if installViaHive {
				active.Properties.OpenShiftPullspec = strings.Replace(active.Properties.OpenShiftPullspec, "quay.io", env.ACRDomain(), 1)
			}
			return active, nil
		}
	}

	return nil, errUnsupportedVersion
}

type FakeOpenShiftClusterDocumentVersionerService struct {
	expectedOpenShiftVersion *api.OpenShiftVersion
	expectedError            error
}

func (fake *FakeOpenShiftClusterDocumentVersionerService) Get(ctx context.Context, doc *api.OpenShiftClusterDocument, dbOpenShiftVersions database.OpenShiftVersions, env env.Interface, installViaHive bool) (*api.OpenShiftVersion, error) {
	return fake.expectedOpenShiftVersion, fake.expectedError
}
