package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
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

func TestGetOTelProfiles(t *testing.T) {
	profiles, err := getOTelProfiles(arov1alpha1.OperatorFlags{})
	if err != nil {
		t.Fatal(err)
	}
	if profiles.master != otelProfileMinimalLogs || profiles.worker != otelProfileMinimalLogs {
		t.Fatalf("unexpected default profiles: %#v", profiles)
	}

	profiles, err = getOTelProfiles(arov1alpha1.OperatorFlags{
		operator.GenevaLoggingOTelMasterProfile: operator.GenevaLoggingOTelProfileHighLogLevel,
		operator.GenevaLoggingOTelWorkerProfile: operator.GenevaLoggingOTelProfileReducedLogs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if profiles.master != otelProfileHighLogLevel || profiles.worker != otelProfileReducedLogs {
		t.Fatalf("unexpected override profiles: %#v", profiles)
	}
}

func TestSelectOTelConfig(t *testing.T) {
	full := selectOTelConfig(otelProfileHighLogLevel)
	if !strings.Contains(full, "processors: [memory_limiter, transform/log-parity, batch]") {
		t.Fatal("full config missing expected processor chain")
	}

	reduced := selectOTelConfig(otelProfileReducedLogs)
	if !strings.Contains(reduced, "filter/drop-olm-noise:") {
		t.Fatal("reduced config missing noise filter")
	}

	highSignal := selectOTelConfig(otelProfileMinimalLogs)
	if !strings.Contains(highSignal, "filter/keep-only-high-signal:") {
		t.Fatal("high-signal config missing keep-only-high-signal filter")
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
	var foundConfig, foundGatewayCA bool
	for _, obj := range out {
		switch typed := obj.(type) {
		case *corev1.ConfigMap:
			if typed.Name == otelConfigMapName {
				foundConfig = true
			}
			if typed.Name == otelGatewayCACMName {
				foundGatewayCA = true
			}
		case *appsv1.DaemonSet:
			daemonsetNames = append(daemonsetNames, typed.Name)
		}
	}

	if !foundConfig || !foundGatewayCA {
		t.Fatalf("missing expected OTel configmaps: config=%t gatewayCA=%t", foundConfig, foundGatewayCA)
	}
	if !reflect.DeepEqual(daemonsetNames, []string{"otel-collector-master", "otel-collector-worker"}) {
		t.Fatalf("unexpected daemonsets: %v", daemonsetNames)
	}
}

func TestOTelDaemonSets(t *testing.T) {
	r := &Reconciler{}
	daemonsets, err := r.otelDaemonSets(&arov1alpha1.Cluster{
		Spec: arov1alpha1.ClusterSpec{
			ResourceID: testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
			ACRDomain:  "acrDomain",
		},
	}, "10.0.0.8:4317", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(daemonsets) != 2 {
		t.Fatalf("got %d daemonsets, want 2", len(daemonsets))
	}

	for _, ds := range daemonsets {
		collector, ok := getContainer(ds, "otel-collector")
		if !ok {
			t.Fatalf("missing otel-collector container in %s", ds.Name)
		}
		if collector.Image == "" {
			t.Fatalf("missing image for %s", ds.Name)
		}
		if ds.Spec.Template.Spec.PriorityClassName == "" {
			t.Fatalf("missing priority class for %s", ds.Name)
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

func TestGenevaLoggingResourcesCreateCABundleBeforeGatewayTargetReady(t *testing.T) {
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
	var foundGatewayCA bool
	for _, obj := range out {
		switch typed := obj.(type) {
		case *corev1.ConfigMap:
			if typed.Name == otelGatewayCACMName {
				foundGatewayCA = true
			}
		case *appsv1.DaemonSet:
			daemonsetCount++
		}
	}

	if !foundGatewayCA {
		t.Fatal("missing gateway CA configmap when gateway endpoint is not yet available")
	}
	if daemonsetCount != 0 {
		t.Fatalf("expected no daemonsets before gateway endpoint is ready, got %d", daemonsetCount)
	}
}

func TestClearOTelDaemonSetNodeSelectors(t *testing.T) {
	ctx := context.Background()
	master := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "otel-collector-master", Namespace: kubeNamespace},
		Spec:       appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{NodeSelector: map[string]string{"custom": "true"}}}},
	}
	worker := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "otel-collector-worker", Namespace: kubeNamespace},
		Spec:       appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{NodeSelector: map[string]string{"custom": "true"}}}},
	}

	r := &Reconciler{AROController: base.AROController{Client: ctrlfake.NewClientBuilder().WithObjects(master, worker).Build()}}
	if err := r.clearOTelDaemonSetNodeSelectors(ctx); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"otel-collector-master", "otel-collector-worker"} {
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
