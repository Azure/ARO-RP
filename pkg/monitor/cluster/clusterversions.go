package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (mon *Monitor) emitClusterVersions(ctx context.Context) error {
	cv, err := mon.getClusterVersion(ctx)
	if err != nil {
		return err
	}

	aroDeployments, err := mon.listARODeployments(ctx)
	if err != nil {
		return err
	}

	operatorVersion := "unknown" // TODO(mj): Once unknown is not present anymore, simplify this
	for _, d := range aroDeployments.Items {
		if d.Name == "aro-operator-master" {
			if d.Labels != nil {
				if val, ok := d.Labels["version"]; ok {
					operatorVersion = val
				}
			}
		}
	}

	availableRP := ""
	if version.GitCommit != mon.oc.Properties.ProvisionedBy {
		availableRP = version.GitCommit
	}

	actualVersion := actualVersion(cv)
	actualMinorVersion := ""
	if len(actualVersion) > 0 {
		parsedVersion, err := version.ParseVersion(actualVersion)
		if err != nil {
			return err
		}
		actualMinorVersion = parsedVersion.MinorVersion()
	}

	mon.emitGauge("cluster.versions", 1, map[string]string{
		"actualVersion":                        actualVersion,
		"desiredVersion":                       desiredVersion(cv),
		"provisionedByResourceProviderVersion": mon.oc.Properties.ProvisionedBy,                     // last successful Put or Patch
		"resourceProviderVersion":              version.GitCommit,                                   // RP version currently running
		"operatorVersion":                      operatorVersion,                                     // operator version in the cluster
		"availableVersion":                     availableVersion(cv, version.UpgradeStreams),        // current available version for upgrade from stream
		"availableRP":                          availableRP,                                         // current RP version available for document update, empty when none
		"latestGaMinorVersion":                 version.DefaultInstallStream.Version.MinorVersion(), // Latest GA in ARO Minor version
		"actualMinorVersion":                   actualMinorVersion,                                  // Minor version, empty if actual version is not in expected form
	})

	return nil
}

// actualVersion finds the actual current cluster state. The history is ordered by most
// recent first, so find the latest "Completed" status to get current
// cluster version
func actualVersion(cv *configv1.ClusterVersion) string {
	for _, history := range cv.Status.History {
		if history.State == configv1.CompletedUpdate {
			return history.Version
		}
	}
	return ""
}

func desiredVersion(cv *configv1.ClusterVersion) string {
	if cv.Spec.DesiredUpdate != nil &&
		cv.Spec.DesiredUpdate.Version != "" {
		return cv.Spec.DesiredUpdate.Version
	}

	return cv.Status.Desired.Version
}

// availableVersion checks the upgradeStreams for possible upgrade of the cluster
// when the upgrade is possible withing the current Y version, return true and closest upgrade stream version
func availableVersion(cv *configv1.ClusterVersion, streams []*version.Stream) string {
	av := actualVersion(cv)
	v, err := version.ParseVersion(av)
	if err != nil {
		return ""
	}

	uStream := version.GetUpgradeStream(streams, v, false)
	if uStream == nil {
		return ""
	}

	return uStream.Version.String()
}
