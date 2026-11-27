package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	// idmsName is the name used by the installer (installer-aro-wrapper) for
	// the ImageDigestMirrorSet that points release images at the ARO ACR.
	idmsName = "image-digest-mirror"

	idmsSourceRegistry = "quay.io/openshift-release-dev"
	idmsMirrorPath     = "openshift-release-dev"
)

// idmsRepositories are the repositories the installer mirrors to the ARO ACR.
var idmsRepositories = []string{
	"ocp-release",
	"ocp-release-nightly",
	"ocp-v4.0-art-dev",
}

// idmsWorkaround makes sure that the ARO ACR mirrors of the OpenShift release
// repositories are declared in an ImageDigestMirrorSet.
//
// Workers always get the ACR mirrors in /etc/containers/registries.conf from
// the static 90-aro-worker-registries MachineConfig. From 4.22 the signature
// policy also covers ocp-v4.0-art-dev, and the MCO only writes the sigstore
// lookup (registries.d) entries for a mirror when it learns about the mirror
// from an ImageDigestMirrorSet or ImageContentSourcePolicy. Without one, new
// machines are rejected pulling the first-boot images from the mirror ("A
// signature was required, but no signature exists") and never join.
type idmsWorkaround struct {
	log    *logrus.Entry
	client client.Client
}

var _ Workaround = &idmsWorkaround{}

func NewIDMSWorkaround(log *logrus.Entry, client client.Client) *idmsWorkaround {
	return &idmsWorkaround{log: log, client: client}
}

// Name implements [Workaround].
func (w *idmsWorkaround) Name() string {
	return "ImageDigestMirrorSet for the ARO ACR mirrors"
}

// IsRequired implements [Workaround].
//
// It is required on 4.21 and later only. On 4.20 any ImageDigestMirrorSet or
// ImageContentSourcePolicy raises the 4.20 -> 4.21 AdminAckRequired upgrade
// gate, so we must not create one there. It is also not applied while the
// cluster is upgrading, to avoid re-rendering MachineConfigs mid-upgrade.
func (w *idmsWorkaround) IsRequired(ctx context.Context, clusterVersion version.Version, cluster *arov1alpha1.Cluster) (bool, error) {
	if !idmsEnabled(cluster.Spec.OperatorFlags) {
		return false, nil
	}

	if clusterVersion.Lt(version.NewVersion(4, 21, 0)) {
		return false, nil
	}

	if cluster.Spec.ACRDomain == "" {
		w.log.Debug("ACR domain is not set, not managing the ImageDigestMirrorSet")
		return false, nil
	}

	cv := &configv1.ClusterVersion{}
	err := w.client.Get(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return false, fmt.Errorf("error getting the ClusterVersion: %w", err)
	}

	return !version.IsClusterUpgrading(cv), nil
}

// idmsEnabled reports whether the workaround is enabled. Unlike most operator
// flags, an absent flag means enabled: existing clusters keep their persisted
// operator flags and never receive new defaults, so treating a missing key as
// false would leave the repair off on exactly the clusters that need it. Only
// an explicit value can turn it off.
func idmsEnabled(flags arov1alpha1.OperatorFlags) bool {
	if _, ok := flags[operator.IDMSWorkaroundEnabled]; !ok {
		return true
	}

	return flags.GetSimpleBoolean(operator.IDMSWorkaroundEnabled)
}

// Ensure implements [Workaround]. It never modifies or removes anything that
// already declares a mirror: a source that is already mirrored by any
// ImageDigestMirrorSet or ImageContentSourcePolicy is left alone.
func (w *idmsWorkaround) Ensure(ctx context.Context) error {
	cluster := &arov1alpha1.Cluster{}
	err := w.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster)
	if err != nil {
		return err
	}
	acrDomain := cluster.Spec.ACRDomain
	if acrDomain == "" {
		return nil
	}

	mirrored, err := w.mirroredSources(ctx)
	if err != nil {
		return err
	}

	var missing []configv1.ImageDigestMirrors
	for _, repo := range idmsRepositories {
		source := idmsSourceRegistry + "/" + repo
		if mirrored[source] {
			continue
		}

		missing = append(missing, configv1.ImageDigestMirrors{
			Source:  source,
			Mirrors: []configv1.ImageMirror{configv1.ImageMirror(acrDomain + "/" + idmsMirrorPath + "/" + repo)},
		})
	}

	if len(missing) == 0 {
		return nil
	}

	idms := &configv1.ImageDigestMirrorSet{}
	err = w.client.Get(ctx, types.NamespacedName{Name: idmsName}, idms)
	switch {
	case kerrors.IsNotFound(err):
		w.log.Infof("creating ImageDigestMirrorSet %s", idmsName)
		return w.client.Create(ctx, &configv1.ImageDigestMirrorSet{
			ObjectMeta: metav1.ObjectMeta{Name: idmsName},
			Spec:       configv1.ImageDigestMirrorSetSpec{ImageDigestMirrors: missing},
		})
	case err != nil:
		return err
	}

	w.log.Infof("adding %d missing source(s) to ImageDigestMirrorSet %s", len(missing), idmsName)
	idms.Spec.ImageDigestMirrors = append(idms.Spec.ImageDigestMirrors, missing...)
	return w.client.Update(ctx, idms)
}

// Remove implements [Workaround]. The ImageDigestMirrorSet is intentionally
// never removed: it may have been created or edited by the customer, and
// removing it would put the cluster back into the failing configuration.
func (w *idmsWorkaround) Remove(ctx context.Context) error {
	return nil
}

// mirroredSources returns the set of sources which have at least one mirror
// declared in any ImageDigestMirrorSet or ImageContentSourcePolicy.
func (w *idmsWorkaround) mirroredSources(ctx context.Context) (map[string]bool, error) {
	mirrored := map[string]bool{}

	idmsList := &configv1.ImageDigestMirrorSetList{}
	err := w.client.List(ctx, idmsList)
	if err != nil && !meta.IsNoMatchError(err) {
		return nil, fmt.Errorf("error listing ImageDigestMirrorSets: %w", err)
	}
	for _, idms := range idmsList.Items {
		for _, m := range idms.Spec.ImageDigestMirrors {
			if len(m.Mirrors) > 0 {
				mirrored[m.Source] = true
			}
		}
	}

	// ImageContentSourcePolicy is deprecated, but clusters installed before
	// 4.13 are configured with it instead of an ImageDigestMirrorSet.
	icspList := &operatorv1alpha1.ImageContentSourcePolicyList{}
	err = w.client.List(ctx, icspList)
	if err != nil && !meta.IsNoMatchError(err) {
		return nil, fmt.Errorf("error listing ImageContentSourcePolicies: %w", err)
	}
	for _, icsp := range icspList.Items {
		for _, m := range icsp.Spec.RepositoryDigestMirrors {
			if len(m.Mirrors) > 0 {
				mirrored[m.Source] = true
			}
		}
	}

	return mirrored, nil
}
