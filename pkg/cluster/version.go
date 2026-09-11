package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-semver/semver"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/feature"
)

// openShiftClusterDocumentVersioner is the interface that validates and obtains the version from an OpenShiftClusterDocument.
type openShiftClusterDocumentVersioner interface {
	// Get validates and obtains the OpenShift version of the OpenShiftClusterDocument doc using dbOpenShiftVersions, env and installViaHive parameters.
	Get(ctx context.Context, doc *api.OpenShiftClusterDocument, dbOpenShiftVersions database.OpenShiftVersions, env env.Interface, installViaHive bool) (*api.OpenShiftVersion, error)

	// GetWithSubscription validates and obtains the OpenShift version with subscription support for AFEC flags.
	GetWithSubscription(ctx context.Context, doc *api.OpenShiftClusterDocument, dbOpenShiftVersions database.OpenShiftVersions, env env.Interface, installViaHive bool, subscription *api.SubscriptionDocument) (*api.OpenShiftVersion, error)
}

// openShiftClusterDocumentVersionerService is the default implementation of the openShiftClusterDocumentVersioner interface.
type openShiftClusterDocumentVersionerService struct{}

func (service *openShiftClusterDocumentVersionerService) Get(ctx context.Context, doc *api.OpenShiftClusterDocument, dbOpenShiftVersions database.OpenShiftVersions, env env.Interface, installViaHive bool) (*api.OpenShiftVersion, error) {
	// For backward compatibility, call the enhanced version without subscription data
	return service.GetWithSubscription(ctx, doc, dbOpenShiftVersions, env, installViaHive, nil)
}

func (service *openShiftClusterDocumentVersionerService) GetWithSubscription(ctx context.Context, doc *api.OpenShiftClusterDocument, dbOpenShiftVersions database.OpenShiftVersions, env env.Interface, installViaHive bool, subscription *api.SubscriptionDocument) (*api.OpenShiftVersion, error) {
	requestedInstallVersion := doc.OpenShiftCluster.Properties.ClusterProfile.Version

	// TODO: Refactor to use changefeeds rather than querying the database every time
	// should also leverage shared changefeed or shared logic
	docs, err := dbOpenShiftVersions.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	activeOpenShiftVersions := make([]*api.OpenShiftVersion, 0)
	for _, versionDoc := range docs.OpenShiftVersionDocuments {
		if versionDoc.OpenShiftVersion.Properties.Enabled {
			activeOpenShiftVersions = append(activeOpenShiftVersions, versionDoc.OpenShiftVersion)
		}
	}

	// First, try to find the version in CosmosDB (existing behavior)
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

	// If not found in CosmosDB, check if arbitrary versions are enabled via AFEC flag or development environment
	allowArbitraryVersions := env.IsLocalDevelopmentMode() || 
		(subscription != nil && feature.IsRegisteredForFeature(subscription.Subscription.Properties, api.FeatureFlagArbitraryVersions))
	
	if allowArbitraryVersions {
		return service.generateACRVersionSpec(ctx, requestedInstallVersion, env, installViaHive)
	}

	errUnsupportedVersion := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.clusterProfile.version", fmt.Sprintf("The requested OpenShift version '%s' is not supported.", requestedInstallVersion))
	return nil, errUnsupportedVersion
}

// generateACRVersionSpec creates an OpenShiftVersion spec for arbitrary versions using ACR naming patterns
func (service *openShiftClusterDocumentVersionerService) generateACRVersionSpec(ctx context.Context, requestedVersion string, env env.Interface, installViaHive bool) (*api.OpenShiftVersion, error) {
	// Parse the version to extract major.minor for ACR image tagging
	parsedVersion, err := semver.NewVersion(requestedVersion)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.clusterProfile.version", fmt.Sprintf("The requested OpenShift version '%s' is not a valid semantic version.", requestedVersion))
	}

	// Generate ACR-based image specifications
	majorMinor := fmt.Sprintf("%d.%d", parsedVersion.Major, parsedVersion.Minor)
	acrDomain := env.ACRDomain()
	
	// Honor any pull spec override set
	installerPullspec := fmt.Sprintf("%s/aro-installer:%s", acrDomain, majorMinor)
	installerPullSpecOverride := env.LiveConfig().DefaultInstallerPullSpecOverride(ctx)
	if installerPullSpecOverride != "" {
		installerPullspec = installerPullSpecOverride
	}

	// For OpenShift pullspec, use either ACR (for Hive) or quay.io (for traditional installer)
	var openShiftPullspec string
	if installViaHive {
		// For Hive installations, use ACR domain
		// This is a best-effort approach - the exact image may not exist
		openShiftPullspec = fmt.Sprintf("%s/ocp-release:%s", acrDomain, requestedVersion)
	} else {
		// For traditional installations, use quay.io pattern
		openShiftPullspec = fmt.Sprintf("quay.io/openshift-release-dev/ocp-release:%s", requestedVersion)
	}

	return &api.OpenShiftVersion{
		Properties: api.OpenShiftVersionProperties{
			Version:           requestedVersion,
			OpenShiftPullspec: openShiftPullspec,
			InstallerPullspec: installerPullspec,
			Enabled:           true, // Enabled by virtue of being generated for arbitrary versions
		},
	}, nil
}

type FakeOpenShiftClusterDocumentVersionerService struct {
	expectedOpenShiftVersion *api.OpenShiftVersion
	expectedError            error
}

func (fake *FakeOpenShiftClusterDocumentVersionerService) Get(ctx context.Context, doc *api.OpenShiftClusterDocument, dbOpenShiftVersions database.OpenShiftVersions, env env.Interface, installViaHive bool) (*api.OpenShiftVersion, error) {
	return fake.expectedOpenShiftVersion, fake.expectedError
}

func (fake *FakeOpenShiftClusterDocumentVersionerService) GetWithSubscription(ctx context.Context, doc *api.OpenShiftClusterDocument, dbOpenShiftVersions database.OpenShiftVersions, env env.Interface, installViaHive bool, subscription *api.SubscriptionDocument) (*api.OpenShiftVersion, error) {
	return fake.expectedOpenShiftVersion, fake.expectedError
}
