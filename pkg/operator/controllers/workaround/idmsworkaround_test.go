package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"

	apiversion "github.com/Azure/ARO-RP/pkg/api/util/version"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const testACRDomain = "arosvc.azurecr.io"

func idmsTestCluster(flags map[string]string, acrDomain string) *v1alpha1.Cluster {
	return &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: v1alpha1.SingletonClusterName},
		Spec: v1alpha1.ClusterSpec{
			ACRDomain:     acrDomain,
			OperatorFlags: v1alpha1.OperatorFlags(flags),
		},
	}
}

func idmsTestClusterVersion(state configv1.UpdateState) *configv1.ClusterVersion {
	return &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "version"},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{{State: state, Version: "4.22.0"}},
		},
	}
}

func idmsEntry(repo, mirror string) configv1.ImageDigestMirrors {
	return configv1.ImageDigestMirrors{
		Source:  "quay.io/openshift-release-dev/" + repo,
		Mirrors: []configv1.ImageMirror{configv1.ImageMirror(mirror)},
	}
}

func allACRIDMSEntries() []configv1.ImageDigestMirrors {
	return []configv1.ImageDigestMirrors{
		idmsEntry("ocp-release", testACRDomain+"/openshift-release-dev/ocp-release"),
		idmsEntry("ocp-release-nightly", testACRDomain+"/openshift-release-dev/ocp-release-nightly"),
		idmsEntry("ocp-v4.0-art-dev", testACRDomain+"/openshift-release-dev/ocp-v4.0-art-dev"),
	}
}

func TestIDMSWorkaroundIsRequired(t *testing.T) {
	enabled := map[string]string{operator.IDMSWorkaroundEnabled: operator.FlagTrue}

	for _, tt := range []struct {
		name           string
		flags          map[string]string
		acrDomain      string
		clusterVersion apiversion.Version
		cvState        configv1.UpdateState
		noClusterVer   bool
		want           bool
		wantErr        bool
	}{
		{
			name:           "flag not set",
			flags:          map[string]string{},
			acrDomain:      testACRDomain,
			clusterVersion: apiversion.NewVersion(4, 22, 0),
			cvState:        configv1.CompletedUpdate,
		},
		{
			name:           "flag disabled",
			flags:          map[string]string{operator.IDMSWorkaroundEnabled: operator.FlagFalse},
			acrDomain:      testACRDomain,
			clusterVersion: apiversion.NewVersion(4, 22, 0),
			cvState:        configv1.CompletedUpdate,
		},
		{
			name:           "4.20 is not required, it would raise the 4.21 upgrade gate",
			flags:          enabled,
			acrDomain:      testACRDomain,
			clusterVersion: apiversion.NewVersion(4, 20, 45),
			cvState:        configv1.CompletedUpdate,
		},
		{
			name:           "4.19 is not required",
			flags:          enabled,
			acrDomain:      testACRDomain,
			clusterVersion: apiversion.NewVersion(4, 19, 24),
			cvState:        configv1.CompletedUpdate,
		},
		{
			name:           "4.21 is required",
			flags:          enabled,
			acrDomain:      testACRDomain,
			clusterVersion: apiversion.NewVersion(4, 21, 0),
			cvState:        configv1.CompletedUpdate,
			want:           true,
		},
		{
			name:           "4.22 is required",
			flags:          enabled,
			acrDomain:      testACRDomain,
			clusterVersion: apiversion.NewVersion(4, 22, 22),
			cvState:        configv1.CompletedUpdate,
			want:           true,
		},
		{
			name:           "not required while the cluster is upgrading",
			flags:          enabled,
			acrDomain:      testACRDomain,
			clusterVersion: apiversion.NewVersion(4, 21, 5),
			cvState:        configv1.PartialUpdate,
		},
		{
			name:           "not required without an ACR domain",
			flags:          enabled,
			acrDomain:      "",
			clusterVersion: apiversion.NewVersion(4, 22, 0),
			cvState:        configv1.CompletedUpdate,
		},
		{
			name:           "error when the ClusterVersion cannot be read",
			flags:          enabled,
			acrDomain:      testACRDomain,
			clusterVersion: apiversion.NewVersion(4, 22, 0),
			noClusterVer:   true,
			wantErr:        true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			_, log := testlog.LogForTesting(t)

			var objs []client.Object
			if !tt.noClusterVer {
				objs = append(objs, idmsTestClusterVersion(tt.cvState))
			}
			cl := testclienthelper.NewAROFakeClientBuilder(objs...).Build()

			w := NewIDMSWorkaround(log, cl)

			got, err := w.IsRequired(t.Context(), tt.clusterVersion, idmsTestCluster(tt.flags, tt.acrDomain))
			if tt.wantErr {
				r.Error(err)
				return
			}
			r.NoError(err)
			r.Equal(tt.want, got)
		})
	}
}

func TestIDMSWorkaroundEnsure(t *testing.T) {
	errFail := errors.New("failed client")

	for _, tt := range []struct {
		name     string
		cluster  *v1alpha1.Cluster
		objects  []client.Object
		addHooks func(*testclienthelper.HookingClient)

		// want is the expected spec of image-digest-mirror after Ensure. nil
		// means the object must not exist.
		want    []configv1.ImageDigestMirrors
		wantErr error
	}{
		{
			name:    "no mirror objects, creates the ImageDigestMirrorSet with all three sources",
			cluster: idmsTestCluster(nil, testACRDomain),
			want:    allACRIDMSEntries(),
		},
		{
			name:    "uses the cluster's ACR domain",
			cluster: idmsTestCluster(nil, "arointsvc.azurecr.io"),
			want: []configv1.ImageDigestMirrors{
				idmsEntry("ocp-release", "arointsvc.azurecr.io/openshift-release-dev/ocp-release"),
				idmsEntry("ocp-release-nightly", "arointsvc.azurecr.io/openshift-release-dev/ocp-release-nightly"),
				idmsEntry("ocp-v4.0-art-dev", "arointsvc.azurecr.io/openshift-release-dev/ocp-v4.0-art-dev"),
			},
		},
		{
			name:    "existing complete ImageDigestMirrorSet is left untouched",
			cluster: idmsTestCluster(nil, testACRDomain),
			objects: []client.Object{
				&configv1.ImageDigestMirrorSet{
					ObjectMeta: metav1.ObjectMeta{Name: idmsName},
					Spec:       configv1.ImageDigestMirrorSetSpec{ImageDigestMirrors: allACRIDMSEntries()},
				},
			},
			want: allACRIDMSEntries(),
		},
		{
			name:    "existing ImageDigestMirrorSet missing sources gets them appended, existing entries are kept",
			cluster: idmsTestCluster(nil, testACRDomain),
			objects: []client.Object{
				&configv1.ImageDigestMirrorSet{
					ObjectMeta: metav1.ObjectMeta{Name: idmsName},
					Spec: configv1.ImageDigestMirrorSetSpec{ImageDigestMirrors: []configv1.ImageDigestMirrors{
						idmsEntry("ocp-release", testACRDomain+"/openshift-release-dev/ocp-release"),
						{Source: "registry.example.com/custom", Mirrors: []configv1.ImageMirror{"mirror.example.com/custom"}},
					}},
				},
			},
			want: []configv1.ImageDigestMirrors{
				idmsEntry("ocp-release", testACRDomain+"/openshift-release-dev/ocp-release"),
				{Source: "registry.example.com/custom", Mirrors: []configv1.ImageMirror{"mirror.example.com/custom"}},
				idmsEntry("ocp-release-nightly", testACRDomain+"/openshift-release-dev/ocp-release-nightly"),
				idmsEntry("ocp-v4.0-art-dev", testACRDomain+"/openshift-release-dev/ocp-v4.0-art-dev"),
			},
		},
		{
			name:    "sources mirrored by a customer's own ImageDigestMirrorSet are not touched",
			cluster: idmsTestCluster(nil, testACRDomain),
			objects: []client.Object{
				&configv1.ImageDigestMirrorSet{
					ObjectMeta: metav1.ObjectMeta{Name: "customer-mirror"},
					Spec: configv1.ImageDigestMirrorSetSpec{ImageDigestMirrors: []configv1.ImageDigestMirrors{
						idmsEntry("ocp-release", "mirror.example.com/ocp-release"),
						idmsEntry("ocp-release-nightly", "mirror.example.com/ocp-release-nightly"),
						idmsEntry("ocp-v4.0-art-dev", "mirror.example.com/ocp-v4.0-art-dev"),
					}},
				},
			},
		},
		{
			name:    "an ImageContentSourcePolicy covering the sources means nothing is created",
			cluster: idmsTestCluster(nil, testACRDomain),
			objects: []client.Object{
				&operatorv1alpha1.ImageContentSourcePolicy{
					ObjectMeta: metav1.ObjectMeta{Name: "image-policy-0"},
					Spec: operatorv1alpha1.ImageContentSourcePolicySpec{RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
						{Source: "quay.io/openshift-release-dev/ocp-release", Mirrors: []string{testACRDomain + "/openshift-release-dev/ocp-release"}},
						{Source: "quay.io/openshift-release-dev/ocp-release-nightly", Mirrors: []string{testACRDomain + "/openshift-release-dev/ocp-release-nightly"}},
						{Source: "quay.io/openshift-release-dev/ocp-v4.0-art-dev", Mirrors: []string{testACRDomain + "/openshift-release-dev/ocp-v4.0-art-dev"}},
					}},
				},
			},
		},
		{
			name:    "a partial ImageContentSourcePolicy only gets the missing sources created",
			cluster: idmsTestCluster(nil, testACRDomain),
			objects: []client.Object{
				&operatorv1alpha1.ImageContentSourcePolicy{
					ObjectMeta: metav1.ObjectMeta{Name: "image-policy-0"},
					Spec: operatorv1alpha1.ImageContentSourcePolicySpec{RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
						{Source: "quay.io/openshift-release-dev/ocp-release", Mirrors: []string{testACRDomain + "/openshift-release-dev/ocp-release"}},
					}},
				},
			},
			want: []configv1.ImageDigestMirrors{
				idmsEntry("ocp-release-nightly", testACRDomain+"/openshift-release-dev/ocp-release-nightly"),
				idmsEntry("ocp-v4.0-art-dev", testACRDomain+"/openshift-release-dev/ocp-v4.0-art-dev"),
			},
		},
		{
			name:    "an entry with no mirrors does not count as mirrored",
			cluster: idmsTestCluster(nil, testACRDomain),
			objects: []client.Object{
				&operatorv1alpha1.ImageContentSourcePolicy{
					ObjectMeta: metav1.ObjectMeta{Name: "image-policy-0"},
					Spec: operatorv1alpha1.ImageContentSourcePolicySpec{RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
						{Source: "quay.io/openshift-release-dev/ocp-release"},
					}},
				},
			},
			want: allACRIDMSEntries(),
		},
		{
			name:    "ImageContentSourcePolicy API not served, still creates the ImageDigestMirrorSet",
			cluster: idmsTestCluster(nil, testACRDomain),
			addHooks: func(hc *testclienthelper.HookingClient) {
				hc.WithPreListHook(func(obj client.ObjectList, _ *client.ListOptions) error {
					if _, ok := obj.(*operatorv1alpha1.ImageContentSourcePolicyList); ok {
						return &meta.NoKindMatchError{GroupKind: schema.GroupKind{Group: "operator.openshift.io", Kind: "ImageContentSourcePolicy"}}
					}
					return nil
				})
			},
			want: allACRIDMSEntries(),
		},
		{
			name:    "list error is returned and nothing is created",
			cluster: idmsTestCluster(nil, testACRDomain),
			addHooks: func(hc *testclienthelper.HookingClient) {
				hc.WithPreListHook(func(obj client.ObjectList, _ *client.ListOptions) error {
					return errFail
				})
			},
			wantErr: errFail,
		},
		{
			name:    "create error is returned",
			cluster: idmsTestCluster(nil, testACRDomain),
			addHooks: func(hc *testclienthelper.HookingClient) {
				hc.WithPreCreateHook(func(obj client.Object) error { return errFail })
			},
			wantErr: errFail,
		},
		{
			name:    "no ACR domain, nothing is created",
			cluster: idmsTestCluster(nil, ""),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			_, log := testlog.LogForTesting(t)

			objs := append([]client.Object{tt.cluster}, tt.objects...)
			cl := testclienthelper.NewHookingClient(testclienthelper.NewAROFakeClientBuilder(objs...).Build())
			if tt.addHooks != nil {
				tt.addHooks(cl)
			}

			w := NewIDMSWorkaround(log, cl)

			err := w.Ensure(t.Context())
			if tt.wantErr != nil {
				r.ErrorIs(err, tt.wantErr)
			} else {
				r.NoError(err)
			}

			got := &configv1.ImageDigestMirrorSet{}
			err = cl.Get(t.Context(), types.NamespacedName{Name: idmsName}, got)
			if tt.want == nil {
				r.True(kerrors.IsNotFound(err), "expected %s to be absent, got err %v", idmsName, err)
				return
			}
			r.NoError(err)
			r.Equal(tt.want, got.Spec.ImageDigestMirrors)
		})
	}
}

func TestIDMSWorkaroundEnsureIsIdempotent(t *testing.T) {
	r := require.New(t)
	_, log := testlog.LogForTesting(t)

	cl := testclienthelper.NewAROFakeClientBuilder(idmsTestCluster(nil, testACRDomain)).Build()
	w := NewIDMSWorkaround(log, cl)

	r.NoError(w.Ensure(t.Context()))
	first := &configv1.ImageDigestMirrorSet{}
	r.NoError(cl.Get(t.Context(), types.NamespacedName{Name: idmsName}, first))

	r.NoError(w.Ensure(t.Context()))
	second := &configv1.ImageDigestMirrorSet{}
	r.NoError(cl.Get(t.Context(), types.NamespacedName{Name: idmsName}, second))

	r.Equal(first.ResourceVersion, second.ResourceVersion, "second Ensure must not update the object")
	r.Equal(allACRIDMSEntries(), second.Spec.ImageDigestMirrors)
}

// Remove must never delete the ImageDigestMirrorSet: it is called whenever
// IsRequired is false, including on every cluster older than 4.21.
func TestIDMSWorkaroundRemoveKeepsObject(t *testing.T) {
	r := require.New(t)
	_, log := testlog.LogForTesting(t)

	existing := &configv1.ImageDigestMirrorSet{
		ObjectMeta: metav1.ObjectMeta{Name: idmsName},
		Spec:       configv1.ImageDigestMirrorSetSpec{ImageDigestMirrors: allACRIDMSEntries()},
	}
	cl := testclienthelper.NewAROFakeClientBuilder(existing).Build()
	w := NewIDMSWorkaround(log, cl)

	r.NoError(w.Remove(t.Context()))

	got := &configv1.ImageDigestMirrorSet{}
	r.NoError(cl.Get(t.Context(), types.NamespacedName{Name: idmsName}, got))
	r.Equal(allACRIDMSEntries(), got.Spec.ImageDigestMirrors)
}
