package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	configv1 "github.com/openshift/api/config/v1"
	securityv1 "github.com/openshift/api/security/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func clusterVersion(version string) configv1.ClusterVersion {
	return configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "version"},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{{State: configv1.CompletedUpdate, Version: version}},
		},
	}
}

func getContainer(d *appsv1.DaemonSet, name string) (corev1.Container, bool) {
	for _, c := range d.Spec.Template.Spec.Containers {
		if c.Name == name {
			return c, true
		}
	}
	return corev1.Container{}, false
}

func hasVolume(d *appsv1.DaemonSet, name string) bool {
	for _, v := range d.Spec.Template.Spec.Volumes {
		if v.Name == name {
			return true
		}
	}
	return false
}

func TestGetOTelProfiles(t *testing.T) {
	profiles, err := getOTelProfiles(arov1alpha1.OperatorFlags{})
	if err != nil {
		t.Fatal(err)
	}
	if profiles.master != otelProfileMinimalLogs || profiles.worker != otelProfileMinimalLogs {
		t.Fatalf("unexpected default profiles: %#v", profiles)
	}

	profiles, err = getOTelProfiles(arov1alpha1.OperatorFlags{
		operator.GenevaLoggingOTelMasterProfile: operator.GenevaLoggingOTelProfileMaxLogs,
		operator.GenevaLoggingOTelWorkerProfile: operator.GenevaLoggingOTelProfileReducedLogs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if profiles.master != otelProfileMaxLogs || profiles.worker != otelProfileReducedLogs {
		t.Fatalf("unexpected override profiles: %#v", profiles)
	}
}

func TestSelectOTelConfig(t *testing.T) {
	// All profiles render without error for both control plane and worker nodes.
	for _, profile := range []otelProfile{otelProfileMaxLogs, otelProfileReducedLogs, otelProfileMinimalLogs} {
		for _, isControlPlane := range []bool{true, false} {
			if _, err := renderOTelConfig(profile, isControlPlane); err != nil {
				t.Errorf("renderOTelConfig(%q, isControlPlane=%v): %v", profile, isControlPlane, err)
			}
		}
	}

	// Control plane gets the audit pipeline; workers do not.
	cp, _ := renderOTelConfig(otelProfileMaxLogs, true)
	worker, _ := renderOTelConfig(otelProfileMaxLogs, false)
	if !strings.Contains(cp, "logs/audit:") {
		t.Fatal("control plane config missing audit pipeline")
	}
	if strings.Contains(worker, "logs/audit:") {
		t.Fatal("worker config must not contain audit pipeline")
	}

	// SyncLoop is a control-plane-only pattern gated by the isControlPlane template conditional.
	cpMin, _ := renderOTelConfig(otelProfileMinimalLogs, true)
	workerMin, _ := renderOTelConfig(otelProfileMinimalLogs, false)
	if !strings.Contains(cpMin, "SyncLoop") {
		t.Fatal("control plane minimal-logs config missing SyncLoop pattern")
	}
	if strings.Contains(workerMin, "SyncLoop") {
		t.Fatal("worker minimal-logs config must not contain SyncLoop")
	}
}

func TestSelectOTelConfigFailsIfPrimaryAndFallbackRenderFail(t *testing.T) {
	originalRender := renderOTelConfigFn
	renderOTelConfigFn = func(otelProfile, bool) (string, error) {
		return "", errors.New("render failure")
	}
	defer func() {
		renderOTelConfigFn = originalRender
	}()

	_, err := selectOTelConfig(otelProfileMaxLogs, true)
	if err == nil {
		t.Fatal("expected selectOTelConfig to return an error")
	}
}

func TestSelectOTelConfigFallsBackToMinimalLogs(t *testing.T) {
	originalRender := renderOTelConfigFn
	var calledProfiles []otelProfile
	renderOTelConfigFn = func(profile otelProfile, _ bool) (string, error) {
		calledProfiles = append(calledProfiles, profile)
		if profile == otelProfileMinimalLogs {
			return "minimal-config", nil
		}
		return "", errors.New("render failure")
	}
	defer func() {
		renderOTelConfigFn = originalRender
	}()

	cfg, err := selectOTelConfig(otelProfileMaxLogs, true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg != "minimal-config" {
		t.Fatalf("expected minimal fallback config, got %q", cfg)
	}
	if !reflect.DeepEqual(calledProfiles, []otelProfile{otelProfileMaxLogs, otelProfileMinimalLogs}) {
		t.Fatalf("unexpected render order: %#v", calledProfiles)
	}
}

func TestGenevaLoggingNamespaceLabels(t *testing.T) {
	cv := clusterVersion("4.11.0")
	r := &Reconciler{
		AROController: base.AROController{
			Client: testclienthelper.NewAROFakeClientBuilder(&cv).Build(),
		},
	}

	labels, err := r.namespaceLabels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(labels, privilegedNamespaceLabels) {
		t.Fatalf("got labels %v, want %v", labels, privilegedNamespaceLabels)
	}
}

func TestGenevaLoggingResourcesOTel(t *testing.T) {
	instance := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec: arov1alpha1.ClusterSpec{
			ResourceID:               testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
			ACRDomain:                "acrDomain",
			GatewayPrivateEndpointIP: "10.0.0.8",
			GatewayTelemetryDomain:   "telemetry.eastus.aro.azure.com",
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
			},
		},
	}
	cv := clusterVersion("4.11.0")

	r := &Reconciler{
		AROController: base.AROController{
			Client: testclienthelper.NewAROFakeClientBuilder(instance, &securityv1.SecurityContextConstraints{ObjectMeta: metav1.ObjectMeta{Name: "privileged"}}, &cv).Build(),
		},
	}

	out, err := r.resources(context.Background(), instance)
	if err != nil {
		t.Fatal(err)
	}

	var daemonsetNames []string
	var foundConfig bool
	for _, obj := range out {
		switch typed := obj.(type) {
		case *corev1.ConfigMap:
			if typed.Name == otelConfigMapName {
				foundConfig = true
			}
		case *appsv1.DaemonSet:
			daemonsetNames = append(daemonsetNames, typed.Name)
		}
	}

	if !foundConfig {
		t.Fatal("missing expected OTel configmap")
	}
	if !reflect.DeepEqual(daemonsetNames, []string{MasterDaemonsetName, WorkerDaemonsetName}) {
		t.Fatalf("unexpected daemonsets: %v", daemonsetNames)
	}
}

func TestOTelDaemonSets(t *testing.T) {
	r := &Reconciler{}
	wantImage := version.TelemetryExporterImage("acrDomain")
	daemonsets, err := r.otelDaemonSets(&arov1alpha1.Cluster{
		Spec: arov1alpha1.ClusterSpec{
			ResourceID: testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
			ACRDomain:  "acrDomain",
		},
	}, "10.0.0.8:4317", nil, "master-hash", "worker-hash")
	if err != nil {
		t.Fatal(err)
	}
	if len(daemonsets) != 2 {
		t.Fatalf("got %d daemonsets, want 2", len(daemonsets))
	}

	for _, ds := range daemonsets {
		exporter, ok := getContainer(ds, "otel-exporter")
		if !ok {
			t.Fatalf("missing otel-exporter container in %s", ds.Name)
		}
		if exporter.Image != wantImage {
			t.Fatalf("unexpected image for %s: got %q want %q", ds.Name, exporter.Image, wantImage)
		}
		if !hasVolume(ds, "machine-id") {
			t.Fatalf("missing machine-id volume for %s", ds.Name)
		}
		if ds.Spec.Template.Spec.PriorityClassName == "" {
			t.Fatalf("missing priority class for %s", ds.Name)
		}
		wantHash := "worker-hash"
		if ds.Name == MasterDaemonsetName {
			wantHash = "master-hash"
		}
		if gotHash := ds.Spec.Template.Annotations["aro.openshift.io/otel-config-sha256"]; gotHash != wantHash {
			t.Fatalf("unexpected config hash annotation for %s: got %q want %q", ds.Name, gotHash, wantHash)
		}
		for _, env := range exporter.Env {
			if env.Name == "ENVIRONMENT" {
				t.Fatalf("unexpected ENVIRONMENT var for %s", ds.Name)
			}
		}
	}
}

func TestOTelDaemonSetsUseConfiguredPullSpec(t *testing.T) {
	r := &Reconciler{}
	const overridePullSpec = "example.invalid/otel/exporter:custom"
	daemonsets, err := r.otelDaemonSets(&arov1alpha1.Cluster{
		Spec: arov1alpha1.ClusterSpec{
			ResourceID: testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
			ACRDomain:  "acrDomain",
			OperatorFlags: arov1alpha1.OperatorFlags{
				controllerOTelPullSpec: overridePullSpec,
			},
		},
	}, "10.0.0.8:4317", nil, "master-hash", "worker-hash")
	if err != nil {
		t.Fatal(err)
	}

	for _, ds := range daemonsets {
		exporter, ok := getContainer(ds, "otel-exporter")
		if !ok {
			t.Fatalf("missing otel-exporter container in %s", ds.Name)
		}
		if exporter.Image != overridePullSpec {
			t.Fatalf("unexpected overridden image for %s: got %q want %q", ds.Name, exporter.Image, overridePullSpec)
		}
	}
}

func TestTelemetryGatewayTarget(t *testing.T) {
	for _, tt := range []struct {
		name            string
		cluster         *arov1alpha1.Cluster
		wantReady       bool
		wantEndpoint    string
		wantHostAliases []corev1.HostAlias
		wantErr         string
	}{
		{
			name:      "not ready when gateway endpoint IP is missing",
			cluster:   &arov1alpha1.Cluster{},
			wantReady: false,
		},
		{
			name: "error when gateway endpoint IP is not a valid IP address",
			cluster: &arov1alpha1.Cluster{Spec: arov1alpha1.ClusterSpec{
				GatewayPrivateEndpointIP: "not-an-ip",
				GatewayTelemetryDomain:   "telemetry.eastus.aro.azure.com",
			}},
			wantErr: `invalid cluster spec field "gatewayPrivateEndpointIP": "not-an-ip" is not a valid IP address`,
		},
		{
			name: "error when gateway telemetry domain is empty",
			cluster: &arov1alpha1.Cluster{Spec: arov1alpha1.ClusterSpec{
				GatewayPrivateEndpointIP: "10.0.0.8",
				GatewayTelemetryDomain:   "",
			}},
			wantErr: `invalid cluster spec field "gatewayTelemetryDomain": empty`,
		},
		{
			name: "ready with lowercased endpoint and host alias",
			cluster: &arov1alpha1.Cluster{Spec: arov1alpha1.ClusterSpec{
				GatewayPrivateEndpointIP: "10.0.0.8",
				GatewayTelemetryDomain:   "telemetry.EastUS.aro.azure.com",
			}},
			wantReady:    true,
			wantEndpoint: "telemetry.eastus.aro.azure.com:4317",
			wantHostAliases: []corev1.HostAlias{
				{
					IP:        "10.0.0.8",
					Hostnames: []string{"telemetry.eastus.aro.azure.com"},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			target, ready, err := telemetryGatewayTarget(tt.cluster)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if ready != tt.wantReady {
				t.Fatalf("got ready %v, want %v", ready, tt.wantReady)
			}
			if target.endpoint != tt.wantEndpoint {
				t.Fatalf("got endpoint %q, want %q", target.endpoint, tt.wantEndpoint)
			}
			if !reflect.DeepEqual(target.hostAliases, tt.wantHostAliases) {
				t.Fatalf("got host aliases %#v, want %#v", target.hostAliases, tt.wantHostAliases)
			}
		})
	}
}

func TestGenevaLoggingResourcesCreateConfigBeforeGatewayTargetReady(t *testing.T) {
	instance := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec: arov1alpha1.ClusterSpec{
			ResourceID: testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
			ACRDomain:  "acrDomain",
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
			},
		},
	}
	cv := clusterVersion("4.11.0")

	r := &Reconciler{
		AROController: base.AROController{
			Client: testclienthelper.NewAROFakeClientBuilder(instance, &securityv1.SecurityContextConstraints{ObjectMeta: metav1.ObjectMeta{Name: "privileged"}}, &cv).Build(),
		},
	}

	out, err := r.resources(context.Background(), instance)
	if err != nil {
		t.Fatal(err)
	}

	var daemonsetCount int
	var foundConfig bool
	for _, obj := range out {
		switch typed := obj.(type) {
		case *corev1.ConfigMap:
			if typed.Name == otelConfigMapName {
				foundConfig = true
			}
		case *appsv1.DaemonSet:
			daemonsetCount++
		}
	}

	if !foundConfig {
		t.Fatal("missing OTel configmap when gateway endpoint is not yet available")
	}
	if daemonsetCount != 0 {
		t.Fatalf("expected no daemonsets before gateway endpoint is ready, got %d", daemonsetCount)
	}
}

func TestGenevaLoggingResourcesReturnsErrorWhenOTelConfigRenderFails(t *testing.T) {
	originalRender := renderOTelConfigFn
	renderOTelConfigFn = func(otelProfile, bool) (string, error) {
		return "", errors.New("render failure")
	}
	defer func() {
		renderOTelConfigFn = originalRender
	}()

	instance := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec: arov1alpha1.ClusterSpec{
			ResourceID: testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
			ACRDomain:  "acrDomain",
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
			},
		},
	}
	cv := clusterVersion("4.11.0")

	r := &Reconciler{
		AROController: base.AROController{
			Client: testclienthelper.NewAROFakeClientBuilder(instance, &securityv1.SecurityContextConstraints{ObjectMeta: metav1.ObjectMeta{Name: "privileged"}}, &cv).Build(),
		},
	}

	_, err := r.resources(context.Background(), instance)
	if err == nil {
		t.Fatal("expected resources to return an error when OTel config rendering fails")
	}
}

func TestClearOTelDaemonSetNodeSelectors(t *testing.T) {
	ctx := context.Background()
	master := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: MasterDaemonsetName, Namespace: kubeNamespace},
		Spec:       appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{NodeSelector: map[string]string{"custom": "true"}}}},
	}
	worker := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: WorkerDaemonsetName, Namespace: kubeNamespace},
		Spec:       appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{NodeSelector: map[string]string{"custom": "true"}}}},
	}

	r := &Reconciler{AROController: base.AROController{Client: testclienthelper.NewAROFakeClientBuilder(master, worker).Build()}}
	if err := r.clearOTelDaemonSetNodeSelectors(ctx); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{MasterDaemonsetName, WorkerDaemonsetName} {
		ds := &appsv1.DaemonSet{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: name, Namespace: kubeNamespace}, ds); err != nil {
			t.Fatal(err)
		}
		if len(ds.Spec.Template.Spec.NodeSelector) != 0 {
			t.Fatalf("expected cleared node selector for %s, got %v", name, ds.Spec.Template.Spec.NodeSelector)
		}
	}
}

func TestCleanupStaleResources(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockDh := mock_dynamichelper.NewMockInterface(controller)
	r := &Reconciler{dh: mockDh}

	mockDh.EXPECT().EnsureDeleted(gomock.Any(), "DaemonSet.apps", kubeNamespace, "mdsd").Times(1)
	mockDh.EXPECT().EnsureDeleted(gomock.Any(), "ConfigMap", kubeNamespace, "fluent-config").Times(1)
	mockDh.EXPECT().EnsureDeleted(gomock.Any(), "Secret", kubeNamespace, "certificates").Times(1)
	mockDh.EXPECT().EnsureDeleted(gomock.Any(), "ConfigMap", kubeNamespace, legacyGatewayCACMName).Times(1)

	if err := r.cleanupStaleResources(context.Background()); err != nil {
		t.Fatal(err)
	}
}
