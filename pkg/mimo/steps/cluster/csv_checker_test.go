package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func newCSVRestMapper() meta.RESTMapper {
	restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{
		{Group: "operators.coreos.com", Version: "v1alpha1"},
	})
	restMapper.Add(schema.GroupVersionKind{
		Group:   "operators.coreos.com",
		Version: "v1alpha1",
		Kind:    "ClusterServiceVersion",
	}, meta.RESTScopeNamespace)
	return restMapper
}

func newUnstructuredCSV(namespace, name string) *unstructured.Unstructured {
	csv := &unstructured.Unstructured{}
	csv.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operators.coreos.com",
		Version: "v1alpha1",
		Kind:    "ClusterServiceVersion",
	})
	csv.SetNamespace(namespace)
	csv.SetName(name)
	return csv
}

func TestParseMajorMinor(t *testing.T) {
	for _, tt := range []struct {
		name    string
		version string
		want    string
		wantErr bool
	}{
		{name: "standard version", version: "4.14.22", want: "4.14"},
		{name: "two-part version", version: "4.17", want: "4.17"},
		{name: "longer version", version: "4.15.3-rc.1", want: "4.15"},
		{name: "single part", version: "4", wantErr: true},
		{name: "empty", version: "", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMajorMinor(tt.version)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.want {
				t.Errorf("got %q, want %q", result, tt.want)
			}
		})
	}
}

func TestLoadConcerningCSVs(t *testing.T) {
	for _, tt := range []struct {
		name       string
		version    string
		wantErr    bool
		wantMinLen int
	}{
		{name: "4.12 data exists", version: "4.12", wantMinLen: 900},
		{name: "4.13 data exists", version: "4.13", wantMinLen: 1000},
		{name: "4.14 data exists", version: "4.14", wantMinLen: 800},
		{name: "4.15 data exists", version: "4.15", wantMinLen: 800},
		{name: "4.16 data exists", version: "4.16", wantMinLen: 800},
		{name: "4.17 data exists", version: "4.17", wantMinLen: 500},
		{name: "unsupported version", version: "4.11", wantErr: true},
		{name: "future version", version: "4.18", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := loadConcerningCSVs(tt.version)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) < tt.wantMinLen {
				t.Errorf("got %d entries, want at least %d", len(result), tt.wantMinLen)
			}
		})
	}
}

func TestDetectConcerningClusterServiceVersions(t *testing.T) {
	for _, tt := range []struct {
		name       string
		cv         *configv1.ClusterVersion
		csvs       []*unstructured.Unstructured
		wantErr    string
		wantResult string
	}{
		{
			name: "no concerning CSVs",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "version"},
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{State: configv1.CompletedUpdate, Version: "4.14.22"},
					},
				},
			},
			csvs: []*unstructured.Unstructured{
				newUnstructuredCSV("openshift-operators", "some-safe-operator.v1.0.0"),
				newUnstructuredCSV("openshift-operators", "another-safe-operator.v2.0.0"),
			},
			wantResult: "no inadvertently upgraded 4.18 ClusterServiceVersions detected on cluster running 4.14.22",
		},
		{
			name: "concerning CSVs detected",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "version"},
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{State: configv1.CompletedUpdate, Version: "4.14.22"},
					},
				},
			},
			csvs: []*unstructured.Unstructured{
				newUnstructuredCSV("openshift-operators", "some-safe-operator.v1.0.0"),
				newUnstructuredCSV("openshift-operators", "web-terminal.v1.13.0"),
			},
			wantErr: "TerminalError: inadvertently upgraded 4.18 ClusterServiceVersions detected on cluster running 4.14.22:\nweb-terminal.v1.13.0",
		},
		{
			name: "unrecognized version skips check",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "version"},
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{State: configv1.CompletedUpdate, Version: "4.18.0"},
					},
				},
			},
			csvs:       []*unstructured.Unstructured{},
			wantResult: "no concerning CSV data for version 4.18; skipping check",
		},
		{
			name: "empty version history",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "version"},
				Status:     configv1.ClusterVersionStatus{},
			},
			csvs:    []*unstructured.Unstructured{},
			wantErr: "TerminalError: ClusterVersion has no update history",
		},
		{
			name: "no CSVs on cluster",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "version"},
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{State: configv1.CompletedUpdate, Version: "4.15.10"},
					},
				},
			},
			csvs:       []*unstructured.Unstructured{},
			wantResult: "no inadvertently upgraded 4.18 ClusterServiceVersions detected on cluster running 4.15.10",
		},
		{
			name: "duplicate CSV names across namespaces",
			cv: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "version"},
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{State: configv1.CompletedUpdate, Version: "4.14.22"},
					},
				},
			},
			csvs: []*unstructured.Unstructured{
				newUnstructuredCSV("ns-a", "web-terminal.v1.13.0"),
				newUnstructuredCSV("ns-b", "web-terminal.v1.13.0"),
			},
			wantErr: "TerminalError: inadvertently upgraded 4.18 ClusterServiceVersions detected on cluster running 4.14.22:\nweb-terminal.v1.13.0",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			ctx := context.Background()
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_, log := testlog.New()

			builder := fake.NewClientBuilder().
				WithRESTMapper(newCSVRestMapper()).
				WithRuntimeObjects(tt.cv)
			for _, csv := range tt.csvs {
				builder = builder.WithRuntimeObjects(csv.DeepCopy())
			}

			ch := clienthelper.NewWithClient(log, testclienthelper.NewHookingClient(builder.Build()))
			tc := testtasks.NewFakeTestContext(
				ctx, _env, log, func() time.Time { return time.Unix(100, 0) },
				testtasks.WithClientHelper(ch),
			)

			err := DetectConcerningClusterServiceVersions(tc)

			if tt.wantErr != "" {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(Equal(tt.wantErr))
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				if tt.wantResult != "" {
					g.Expect(tc.GetResultMessage()).To(Equal(tt.wantResult))
				}
			}
		})
	}
}
