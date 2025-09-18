package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	configv1 "github.com/openshift/api/config/v1"

	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (mon *Monitor) emitClusterVersions(ctx context.Context) error {
	aroMasterDeployment := &appsv1.Deployment{}
	err := mon.ocpclientset.Get(ctx, types.NamespacedName{Namespace: pkgoperator.Namespace, Name: "aro-operator-master"}, aroMasterDeployment)
	if err != nil {
		if kerrors.IsNotFound(err) {
			mon.log.Info("aro-operator-master deployment not found")
		} else {
			return errors.Join(errFetchAROOperatorMasterDeployment, err)
		}
	}
	operatorVersion := "unknown"
	if aroMasterDeployment.Labels != nil {
		if val, ok := aroMasterDeployment.Labels["version"]; ok {
			operatorVersion = val
		}
	}

	var availableRP, desiredVersion, actualVersion, actualMinorVersion string

	if version.GitCommit != mon.oc.Properties.ProvisionedBy {
		availableRP = version.GitCommit
	}

	if mon.clusterActualVersion != nil {
		actualVersion = mon.clusterActualVersion.String()
		actualMinorVersion = mon.clusterActualVersion.MinorVersion()
	}

	if mon.clusterDesiredVersion != nil {
		desiredVersion = mon.clusterDesiredVersion.String()
	}

	mon.emitGauge("cluster.versions", 1, map[string]string{
		"actualVersion":                        actualVersion,
		"desiredVersion":                       desiredVersion,
		"provisionedByResourceProviderVersion": mon.oc.Properties.ProvisionedBy, // last successful Put or Patch
		"resourceProviderVersion":              version.GitCommit,               // RP version currently running
		"operatorVersion":                      operatorVersion,                 // operator version in the cluster
		"availableRP":                          availableRP,                     // current RP version available for document update, empty when none
		"actualMinorVersion":                   actualMinorVersion,              // Minor version, empty if actual version is not in expected form
	})

	return nil
}

// Prefetch the cluster version for use in collectors that only need to run on
// certain OpenShift versions.
func (mon *Monitor) prefetchClusterVersion(ctx context.Context) error {
	cv := &configv1.ClusterVersion{}
	err := mon.ocpclientset.Get(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return errors.Join(errFetchClusterVersion, err)
	}

	av := actualVersion(cv)
	avobj, err := version.ParseVersion(av)
	if err != nil {
		mon.log.Errorf("failure parsing ClusterVersion: %s", err.Error())
	} else {
		mon.clusterActualVersion = avobj
	}

	dv := desiredVersion(cv)
	if dv != "" {
		dvobj, err := version.ParseVersion(dv)
		if err != nil {
			mon.log.Errorf("failure parsing desired ClusterVersion: %s", err.Error())
		} else {
			mon.clusterDesiredVersion = dvobj
		}
	}

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
