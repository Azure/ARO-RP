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

	// TODO: Refactor to use changefeeds rather than querying the database every time
	// should also leverage shared changefeed or shared logic
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

	errUnsupportedVersion := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.clusterProfile.version", fmt.Sprintf("The requested OpenShift version '%s' is not supported.", requestedInstallVersion))

	for _, active := range activeOpenShiftVersions {
		if requestedInstallVersion == active.Properties.Version {
			if installViaHive {
				active.Properties.OpenShiftPullspec = strings.Replace(active.Properties.OpenShiftPullspec, "quay.io", env.ACRDomain(), 1)
			}

			// Honor any pull spec override set
			installerPullSpecOverride := env.LiveConfig().DefaultInstallerPullSpecOverride(ctx)
			if installerPullSpecOverride != "" {
				active.Properties.InstallerPullspec = installerPullSpecOverride
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
