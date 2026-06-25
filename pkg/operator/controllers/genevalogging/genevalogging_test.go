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

	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"
	securityv1 "github.com/openshift/api/security/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
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
	full, err := selectOTelConfig(otelProfileMaxLogs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full, "processors: [memory_limiter, transform/log-parity, attributes/common, attributes/source-journald, batch]") {
		t.Fatal("full config missing expected processor chain")
	}
	if !strings.Contains(full, "logs/journald:") || !strings.Contains(full, "logs/containers:") || !strings.Contains(full, "logs/audit:") {
		t.Fatal("full config missing expected per-source pipelines")
	}
	if !strings.Contains(full, "key: EventName") || !strings.Contains(full, "key: source_name") {
		t.Fatal("full config missing source fields")
	}
	if !strings.Contains(full, "processors: [memory_limiter, transform/log-parity, attributes/common, attributes/source-containers, batch]") {
		t.Fatal("full config missing containers source processor")
	}
	if !strings.Contains(full, "set(log.attributes[\"node\"], \"${env:MONITORING_ROLE_INSTANCE}\")") {
		t.Fatal("full config missing node mapping")
	}
	if !strings.Contains(full, "set(log.attributes[\"namespace\"], resource.attributes[\"k8s.namespace.name\"]) where resource.attributes[\"k8s.namespace.name\"] != nil") {
		t.Fatal("full config missing lowercase namespace mapping")
	}
	if !strings.Contains(full, "set(log.attributes[\"pod\"], resource.attributes[\"k8s.pod.name\"]) where resource.attributes[\"k8s.pod.name\"] != nil") {
		t.Fatal("full config missing lowercase pod mapping")
	}
	if !strings.Contains(full, "set(log.attributes[\"container\"], resource.attributes[\"k8s.container.name\"]) where resource.attributes[\"k8s.container.name\"] != nil") {
		t.Fatal("full config missing lowercase container mapping")
	}
	if !strings.Contains(full, "set(log.attributes[\"raw_json_body\"], log.body)") {
		t.Fatal("full config missing raw_json_body mapping")
	}
	if !strings.Contains(full, "id: logrus_parse") {
		t.Fatal("full config missing logrus parser for container logs")
	}
	if !strings.Contains(full, "(?<fields>(?: [^= ]+=") {
		t.Fatal("full config missing logrus generic fields capture")
	}
	if strings.Contains(full, "(?!file=)") {
		t.Fatal("full config contains unsupported RE2 negative lookahead")
	}
	if !strings.Contains(full, "id: logrus_fields_parse") {
		t.Fatal("full config missing logrus structured fields parser for container logs")
	}
	if !strings.Contains(full, "id: logrus_file_split") {
		t.Fatal("full config missing logrus file split parser for container logs")
	}

	reduced, err := selectOTelConfig(otelProfileReducedLogs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reduced, "filter/drop-olm-noise:") {
		t.Fatal("reduced config missing noise filter")
	}
	if !strings.Contains(reduced, "filter/drop-journald-noise:") {
		t.Fatal("reduced config missing journald noise filter")
	}
	if !strings.Contains(reduced, "processors: [memory_limiter, filter/drop-journald-noise, transform/log-parity, attributes/common, attributes/source-journald, batch]") {
		t.Fatal("reduced config missing expected journald processor chain")
	}
	if !strings.Contains(reduced, "filter/keep-audit-write-verbs:") || !strings.Contains(reduced, "filter/drop-audit-review-noise:") || !strings.Contains(reduced, "filter/drop-audit-leases:") || !strings.Contains(reduced, "filter/drop-audit-non-platform-namespaces:") {
		t.Fatal("reduced config missing audit filters")
	}
	if !strings.Contains(reduced, "logs/audit:") || !strings.Contains(reduced, "processors: [memory_limiter, filter/keep-audit-write-verbs, filter/drop-audit-review-noise, filter/drop-audit-leases, filter/drop-audit-non-platform-namespaces, transform/log-parity, attributes/common, attributes/source-audit, batch]") {
		t.Fatal("reduced config missing filtered audit pipeline")
	}

	highSignal, err := selectOTelConfig(otelProfileMinimalLogs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(highSignal, "filter/keep-only-high-signal:") {
		t.Fatal("high-signal config missing keep-only-high-signal filter")
	}
	if !strings.Contains(highSignal, "filter/drop-journald-noise:") {
		t.Fatal("high-signal config missing journald noise filter")
	}
	if !strings.Contains(highSignal, "filter/keep-journald-high-signal:") {
		t.Fatal("high-signal config missing journald high-signal filter")
	}
	if !strings.Contains(highSignal, "processors: [memory_limiter, filter/drop-journald-noise, filter/keep-journald-high-signal, transform/log-parity, attributes/common, attributes/source-journald, batch]") {
		t.Fatal("high-signal config missing expected journald processor chain")
	}
	if !strings.Contains(highSignal, "filter/keep-audit-write-verbs:") {
		t.Fatal("high-signal config missing audit write-verb filter")
	}
	if !strings.Contains(highSignal, "filter/drop-audit-review-noise:") {
		t.Fatal("high-signal config missing audit review-noise filter")
	}
	if !strings.Contains(highSignal, "filter/drop-audit-leases:") {
		t.Fatal("high-signal config missing leases filter")
	}
	if !strings.Contains(highSignal, "filter/drop-audit-non-platform-namespaces:") {
		t.Fatal("high-signal config missing audit namespace filter")
	}
	if !strings.Contains(highSignal, "processors: [memory_limiter, filter/keep-audit-write-verbs, filter/drop-audit-review-noise, filter/drop-audit-leases, filter/drop-audit-non-platform-namespaces, transform/log-parity, attributes/common, attributes/source-audit, batch]") {
		t.Fatal("high-signal config missing expected audit processor chain")
	}
}

func TestSelectOTelConfigFailsIfPrimaryAndFallbackRenderFail(t *testing.T) {
	originalRender := renderOTelConfigFn
	renderOTelConfigFn = func(otelProfile) (string, error) {
		return "", errors.New("render failure")
	}
	defer func() {
		renderOTelConfigFn = originalRender
	}()

	_, err := selectOTelConfig(otelProfileMaxLogs)
	if err == nil {
		t.Fatal("expected selectOTelConfig to return an error")
	}
}

func TestSelectOTelConfigFallsBackToMinimalLogs(t *testing.T) {
	originalRender := renderOTelConfigFn
	var calledProfiles []otelProfile
	renderOTelConfigFn = func(profile otelProfile) (string, error) {
		calledProfiles = append(calledProfiles, profile)
		if profile == otelProfileMinimalLogs {
			return "minimal-config", nil
		}
		return "", errors.New("render failure")
	}
	defer func() {
		renderOTelConfigFn = originalRender
	}()

	cfg, err := selectOTelConfig(otelProfileMaxLogs)
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
			Client: ctrlfake.NewClientBuilder().WithObjects(&cv).Build(),
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
			Client: ctrlfake.NewClientBuilder().WithObjects(instance, &securityv1.SecurityContextConstraints{ObjectMeta: metav1.ObjectMeta{Name: "privileged"}}, &cv).Build(),
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
	target, ready, err := telemetryGatewayTarget(&arov1alpha1.Cluster{Spec: arov1alpha1.ClusterSpec{GatewayPrivateEndpointIP: "10.0.0.8", GatewayTelemetryDomain: "telemetry.eastus.aro.azure.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if !ready {
		t.Fatal("expected telemetry target to be ready")
	}
	if target.endpoint != "telemetry.eastus.aro.azure.com:4317" {
		t.Fatalf("got endpoint %q", target.endpoint)
	}
	if len(target.hostAliases) != 1 || target.hostAliases[0].IP != "10.0.0.8" {
		t.Fatalf("unexpected host aliases: %#v", target.hostAliases)
	}
}

func TestTelemetryGatewayTargetNotReadyWithoutEndpointIP(t *testing.T) {
	_, ready, err := telemetryGatewayTarget(&arov1alpha1.Cluster{})
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("expected telemetry target to be not ready when gateway endpoint IP is missing")
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
			Client: ctrlfake.NewClientBuilder().WithObjects(instance, &securityv1.SecurityContextConstraints{ObjectMeta: metav1.ObjectMeta{Name: "privileged"}}, &cv).Build(),
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
	renderOTelConfigFn = func(otelProfile) (string, error) {
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
			Client: ctrlfake.NewClientBuilder().WithObjects(instance, &securityv1.SecurityContextConstraints{ObjectMeta: metav1.ObjectMeta{Name: "privileged"}}, &cv).Build(),
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

	r := &Reconciler{AROController: base.AROController{Client: ctrlfake.NewClientBuilder().WithObjects(master, worker).Build()}}
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
