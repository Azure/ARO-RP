package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	securityv1 "github.com/openshift/api/security/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func getContainer(d *appsv1.DaemonSet, containerName string) (corev1.Container, error) {
	for _, container := range d.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			return container, nil
		}
	}
	return corev1.Container{}, errors.New("not found")
}

func clusterVersion(version string) configv1.ClusterVersion {
	return configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Spec: configv1.ClusterVersionSpec{},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{
					State:   configv1.CompletedUpdate,
					Version: version,
				},
			},
		},
	}
}

func TestGetLoggingMode(t *testing.T) {
	tests := []struct {
		name         string
		flags        arov1alpha1.OperatorFlags
		wantMode     loggingMode
		wantErrMatch string
	}{
		{
			name:     "defaults to otel",
			flags:    arov1alpha1.OperatorFlags{},
			wantMode: loggingModeOTel,
		},
		{
			name: "explicit mdsd",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingMode: operator.GenevaLoggingModeMDSD,
			},
			wantMode: loggingModeMDSD,
		},
		{
			name: "explicit otel",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingMode: operator.GenevaLoggingModeOTel,
			},
			wantMode: loggingModeOTel,
		},
		{
			name: "reject multiple modes in a single flag value",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingMode: "mdsd,otel",
			},
			wantErrMatch: `unsupported geneva logging mode "mdsd,otel": expected "mdsd" or "otel"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMode, err := getLoggingMode(tt.flags)
			if tt.wantErrMatch != "" {
				utilerror.AssertErrorMessage(t, err, tt.wantErrMatch)
				return
			}

			utilerror.AssertErrorMessage(t, err, "")
			if gotMode != tt.wantMode {
				t.Fatalf("got mode %q, want %q", gotMode, tt.wantMode)
			}
		})
	}
}

func TestGetOTelProfile(t *testing.T) {
	tests := []struct {
		name         string
		flags        arov1alpha1.OperatorFlags
		wantProfile  otelProfile
		wantErrMatch string
	}{
		{
			name:        "defaults to minimal-logs profile",
			flags:       arov1alpha1.OperatorFlags{},
			wantProfile: otelProfileMinimalLogs,
		},
		{
			name: "explicit high-loglevel profile",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingOTelProfile: operator.GenevaLoggingOTelProfileHighLogLevel,
			},
			wantProfile: otelProfileFull,
		},
		{
			name: "explicit reduced-logs profile",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingOTelProfile: operator.GenevaLoggingOTelProfileReducedLogs,
			},
			wantProfile: otelProfileReduced,
		},
		{
			name: "explicit minimal-logs profile",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingOTelProfile: operator.GenevaLoggingOTelProfileMinimalLogs,
			},
			wantProfile: otelProfileHighSignal,
		},
		{
			name: "legacy full profile alias",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingOTelProfile: "full",
			},
			wantProfile: otelProfileFull,
		},
		{
			name: "legacy reduced-noise profile alias",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingOTelProfile: "reduced-noise",
			},
			wantProfile: otelProfileReduced,
		},
		{
			name: "legacy high-signal profile alias",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingOTelProfile: "high-signal",
			},
			wantProfile: otelProfileHighSignal,
		},
		{
			name: "invalid profile value",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingOTelProfile: "foo",
			},
			wantErrMatch: `master profile: unsupported geneva otel profile "foo": expected "high-loglevel", "reduced-logs", or "minimal-logs"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := getOTelProfile(tt.flags)
			if tt.wantErrMatch != "" {
				utilerror.AssertErrorMessage(t, err, tt.wantErrMatch)
				return
			}

			utilerror.AssertErrorMessage(t, err, "")
			if profile != tt.wantProfile {
				t.Fatalf("got profile %q, want %q", profile, tt.wantProfile)
			}
		})
	}
}

func TestGetOTelProfiles(t *testing.T) {
	tests := []struct {
		name         string
		flags        arov1alpha1.OperatorFlags
		wantMaster   otelProfile
		wantWorker   otelProfile
		wantErrMatch string
	}{
		{
			name:       "defaults both roles to minimal-logs",
			flags:      arov1alpha1.OperatorFlags{},
			wantMaster: otelProfileMinimalLogs,
			wantWorker: otelProfileMinimalLogs,
		},
		{
			name: "uses role-specific profile overrides",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingOTelMasterProfile: operator.GenevaLoggingOTelProfileHighLogLevel,
				operator.GenevaLoggingOTelWorkerProfile: operator.GenevaLoggingOTelProfileMinimalLogs,
			},
			wantMaster: otelProfileHighLogLevel,
			wantWorker: otelProfileMinimalLogs,
		},
		{
			name: "uses global profile when role-specific flags unset",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingOTelProfile: operator.GenevaLoggingOTelProfileMinimalLogs,
			},
			wantMaster: otelProfileMinimalLogs,
			wantWorker: otelProfileMinimalLogs,
		},
		{
			name: "invalid worker profile",
			flags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingOTelWorkerProfile: "bad",
			},
			wantErrMatch: `worker profile: unsupported geneva otel profile "bad": expected "high-loglevel", "reduced-logs", or "minimal-logs"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profiles, err := getOTelProfiles(tt.flags)
			if tt.wantErrMatch != "" {
				utilerror.AssertErrorMessage(t, err, tt.wantErrMatch)
				return
			}

			utilerror.AssertErrorMessage(t, err, "")
			if profiles.master != tt.wantMaster {
				t.Fatalf("got master profile %q, want %q", profiles.master, tt.wantMaster)
			}
			if profiles.worker != tt.wantWorker {
				t.Fatalf("got worker profile %q, want %q", profiles.worker, tt.wantWorker)
			}
		})
	}
}

func TestSelectOTelConfig(t *testing.T) {
	tests := []struct {
		name            string
		profile         otelProfile
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:    "full profile",
			profile: otelProfileFull,
			wantContains: []string{
				"processors: [memory_limiter, transform/log-parity, batch]",
			},
			wantNotContains: []string{
				"filter/drop-olm-noise:",
				"filter/keep-only-high-signal:",
			},
		},
		{
			name:    "reduced-noise profile",
			profile: otelProfileReduced,
			wantContains: []string{
				"filter/drop-olm-noise:",
				"processors: [memory_limiter, filter/drop-olm-noise, transform/log-parity, batch]",
			},
		},
		{
			name:    "high-signal profile",
			profile: otelProfileHighSignal,
			wantContains: []string{
				"filter/keep-only-high-signal:",
				"processors: [memory_limiter, filter/keep-only-high-signal, transform/log-parity, batch]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := selectOTelConfig(tt.profile)
			for _, token := range tt.wantContains {
				if !strings.Contains(cfg, token) {
					t.Fatalf("expected selected config to contain %q", token)
				}
			}
			for _, token := range tt.wantNotContains {
				if strings.Contains(cfg, token) {
					t.Fatalf("expected selected config to not contain %q", token)
				}
			}
		})
	}
}

func TestGenevaLoggingNamespaceLabels(t *testing.T) {
	tests := []struct {
		name       string
		cv         configv1.ClusterVersion
		wantLabels map[string]string
		wantErr    string
	}{
		{
			name:       "cluster < 4.11, no labels",
			cv:         clusterVersion("4.10.99"),
			wantLabels: map[string]string{},
		},
		{
			name:       "cluster >= 4.11, use pod security labels",
			cv:         clusterVersion("4.11.0"),
			wantLabels: privilegedNamespaceLabels,
		},
		{
			name:    "cluster version doesn't exist",
			cv:      configv1.ClusterVersion{},
			wantErr: `clusterversions.config.openshift.io "version" not found`,
		},
		{
			name:    "invalid version",
			cv:      clusterVersion("abcd"),
			wantErr: `could not parse version "abcd"`,
		},
	}
	for _, tt := range tests {
		ctx := context.Background()

		controller := gomock.NewController(t)
		defer controller.Finish()

		mockDh := mock_dynamichelper.NewMockInterface(controller)

		r := &Reconciler{
			AROController: base.AROController{
				Log:    logrus.NewEntry(logrus.StandardLogger()),
				Client: ctrlfake.NewClientBuilder().WithObjects(&tt.cv).Build(),
				Name:   ControllerName,
			},
			dh: mockDh,
		}

		labels, err := r.namespaceLabels(ctx)
		utilerror.AssertErrorMessage(t, err, tt.wantErr)

		if !reflect.DeepEqual(labels, tt.wantLabels) {
			t.Errorf("got: %v\nwanted:%v\n", labels, tt.wantLabels)
		}
	}
}

// validateEnvironmentVars validates that both fluentbit and mdsd containers have the correct environment variables
func validateEnvironmentVars(d *appsv1.DaemonSet, expectedValue string) []error {
	var errs []error

	// verify fluentbit has ENVIRONMENT env var
	fluentbit, err := getContainer(d, "fluentbit")
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to get fluentbit container: %w", err))
		return errs
	}
	foundFluentbitEnv := false
	for _, env := range fluentbit.Env {
		if env.Name == "ENVIRONMENT" {
			if env.Value != expectedValue {
				errs = append(errs, fmt.Errorf("fluentbit ENVIRONMENT env var has value '%s', expected '%s'", env.Value, expectedValue))
			}
			foundFluentbitEnv = true
			break
		}
	}
	if !foundFluentbitEnv {
		errs = append(errs, fmt.Errorf("fluentbit container missing ENVIRONMENT env var (expected value: '%s')", expectedValue))
	}

	// verify mdsd has MONITORING_ENVIRONMENT env var
	mdsd, err := getContainer(d, "mdsd")
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to get mdsd container: %w", err))
		return errs
	}
	foundMdsdEnv := false
	for _, env := range mdsd.Env {
		if env.Name == "MONITORING_ENVIRONMENT" {
			if env.Value != expectedValue {
				errs = append(errs, fmt.Errorf("mdsd MONITORING_ENVIRONMENT env var has value '%s', expected '%s'", env.Value, expectedValue))
			}
			foundMdsdEnv = true
			break
		}
	}
	if !foundMdsdEnv {
		errs = append(errs, fmt.Errorf("mdsd container missing MONITORING_ENVIRONMENT env var (expected value: '%s')", expectedValue))
	}

	return errs
}

func TestGenevaLoggingDaemonset(t *testing.T) {
	mdsdCleanupMocks := func(mockDh *mock_dynamichelper.MockInterface) {
		mockDh.EXPECT().EnsureDeleted(gomock.Any(), "DaemonSet.apps", kubeNamespace, "otel-collector-master").Return(nil)
		mockDh.EXPECT().EnsureDeleted(gomock.Any(), "DaemonSet.apps", kubeNamespace, "otel-collector-worker").Return(nil)
		mockDh.EXPECT().EnsureDeleted(gomock.Any(), "ConfigMap", kubeNamespace, otelConfigMapName).Return(nil)
		mockDh.EXPECT().EnsureDeleted(gomock.Any(), "ConfigMap", kubeNamespace, otelGatewayCACMName).Return(nil)
		mockDh.EXPECT().EnsureDeleted(gomock.Any(), "ConfigMap", kubeNamespace, legacyGatewayCACMName).Return(nil)
	}

	otelCleanupMocks := func(mockDh *mock_dynamichelper.MockInterface) {
		mockDh.EXPECT().EnsureDeleted(gomock.Any(), "DaemonSet.apps", kubeNamespace, "mdsd").Return(nil)
		mockDh.EXPECT().EnsureDeleted(gomock.Any(), "ConfigMap", kubeNamespace, "fluent-config").Return(nil)
		mockDh.EXPECT().EnsureDeleted(gomock.Any(), "Secret", kubeNamespace, certificatesSecretName).Return(nil)
		mockDh.EXPECT().EnsureDeleted(gomock.Any(), "ConfigMap", kubeNamespace, legacyGatewayCACMName).Return(nil)
	}
	nominalMocks := func(mockDh *mock_dynamichelper.MockInterface) {
		mdsdCleanupMocks(mockDh)
		mockDh.EXPECT().Ensure(
			gomock.Any(),
			gomock.AssignableToTypeOf(&securityv1.SecurityContextConstraints{}),
			gomock.AssignableToTypeOf(&corev1.Namespace{}),
			gomock.AssignableToTypeOf(&corev1.ConfigMap{}),
			gomock.AssignableToTypeOf(&corev1.Secret{}),
			gomock.AssignableToTypeOf(&corev1.ServiceAccount{}),
			gomock.AssignableToTypeOf(&appsv1.DaemonSet{}),
		).Times(1)
	}
	otelNominalMocks := func(mockDh *mock_dynamichelper.MockInterface) {
		otelCleanupMocks(mockDh)
		mockDh.EXPECT().Ensure(
			gomock.Any(),
			gomock.AssignableToTypeOf(&securityv1.SecurityContextConstraints{}),
			gomock.AssignableToTypeOf(&corev1.Namespace{}),
			gomock.AssignableToTypeOf(&corev1.ConfigMap{}),
			gomock.AssignableToTypeOf(&corev1.ConfigMap{}),
			gomock.AssignableToTypeOf(&corev1.ServiceAccount{}),
			gomock.AssignableToTypeOf(&appsv1.DaemonSet{}),
			gomock.AssignableToTypeOf(&appsv1.DaemonSet{}),
		).Times(1)
	}

	defaultConditions := []operatorv1.OperatorCondition{
		utilconditions.ControllerDefaultAvailable(ControllerName),
		utilconditions.ControllerDefaultProgressing(ControllerName),
		utilconditions.ControllerDefaultDegraded(ControllerName),
	}

	tests := []struct {
		name                     string
		request                  ctrl.Request
		operatorFlags            arov1alpha1.OperatorFlags
		gatewayPrivateEndpointIP string
		gatewayTelemetryDomain   string
		validateDaemonset        func(*appsv1.DaemonSet) []error
		mocks                    func(mockDh *mock_dynamichelper.MockInterface)
		wantErrMsg               string
		wantConditions           []operatorv1.OperatorCondition
	}{
		{
			name: "mdsd mode explicit",
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeMDSD,
			},
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				if len(d.Spec.Template.Spec.Containers) != 2 {
					errs = append(errs, fmt.Errorf("expected 2 containers, got %d", len(d.Spec.Template.Spec.Containers)))
				}

				// we want the default fluentbit image
				fluentbit, err := getContainer(d, "fluentbit")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(fluentbit.Image, version.FluentbitImage("acrDomain")) {
					errs = append(errs, errors.New(err))
				}

				// we want the default mdsd image
				mdsd, err := getContainer(d, "mdsd")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(mdsd.Image, version.MdsdImage("acrDomain")) {
					errs = append(errs, errors.New(err))
				}

				return
			},
			mocks:          nominalMocks,
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "fluentbit/mdsd specs provided as empty strings",
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeMDSD,
				controllerFluentbitPullSpec:   "",
				controllerMDSDPullSpec:        "",
			},
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				if len(d.Spec.Template.Spec.Containers) != 2 {
					errs = append(errs, fmt.Errorf("expected 2 containers, got %d", len(d.Spec.Template.Spec.Containers)))
				}

				// we want the default fluentbit image
				fluentbit, err := getContainer(d, "fluentbit")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(fluentbit.Image, version.FluentbitImage("acrDomain")) {
					errs = append(errs, errors.New(err))
				}

				// we want the default mdsd image
				mdsd, err := getContainer(d, "mdsd")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(mdsd.Image, version.MdsdImage("acrDomain")) {
					errs = append(errs, errors.New(err))
				}

				return
			},
			mocks:          nominalMocks,
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "fluentbit changed",
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeMDSD,
				controllerFluentbitPullSpec:   "otherurl/fluentbit",
			},
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				if len(d.Spec.Template.Spec.Containers) != 2 {
					errs = append(errs, fmt.Errorf("expected 2 containers, got %d", len(d.Spec.Template.Spec.Containers)))
				}

				// we want our fluentbit image
				fluentbit, err := getContainer(d, "fluentbit")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(fluentbit.Image, "otherurl/fluentbit") {
					errs = append(errs, errors.New(err))
				}

				// we want the default mdsd image
				mdsd, err := getContainer(d, "mdsd")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(mdsd.Image, version.MdsdImage("acrDomain")) {
					errs = append(errs, errors.New(err))
				}

				return
			},
			mocks:          nominalMocks,
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "mdsd changed",
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeMDSD,
				controllerMDSDPullSpec:        "otherurl/mdsd",
			},
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				if len(d.Spec.Template.Spec.Containers) != 2 {
					errs = append(errs, fmt.Errorf("expected 2 containers, got %d", len(d.Spec.Template.Spec.Containers)))
				}

				// we want the default fluentbit image
				fluentbit, err := getContainer(d, "fluentbit")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(fluentbit.Image, version.FluentbitImage("acrDomain")) {
					errs = append(errs, errors.New(err))
				}

				// we want the default mdsd image
				mdsd, err := getContainer(d, "mdsd")
				if err != nil {
					errs = append(errs, err)
					return
				}
				for _, err := range deep.Equal(mdsd.Image, "otherurl/mdsd") {
					errs = append(errs, errors.New(err))
				}

				return
			},
			mocks:      nominalMocks,
			wantErrMsg: "",
			wantConditions: []operatorv1.OperatorCondition{
				utilconditions.ControllerDefaultAvailable(ControllerName),
				utilconditions.ControllerDefaultProgressing(ControllerName),
				utilconditions.ControllerDefaultDegraded(ControllerName),
			},
		},
		{
			name: "ENVIRONMENT env var set to 'prod' from aro.environment operator flag",
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeMDSD,
				"aro.environment":             "prod",
			},
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				if len(d.Spec.Template.Spec.Containers) != 2 {
					errs = append(errs, fmt.Errorf("expected 2 containers, got %d", len(d.Spec.Template.Spec.Containers)))
				}
				errs = append(errs, validateEnvironmentVars(d, "prod")...)
				return
			},
			mocks:          nominalMocks,
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "ENVIRONMENT env var set to 'int' from aro.environment operator flag",
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeMDSD,
				"aro.environment":             "int",
			},
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				if len(d.Spec.Template.Spec.Containers) != 2 {
					errs = append(errs, fmt.Errorf("expected 2 containers, got %d", len(d.Spec.Template.Spec.Containers)))
				}
				errs = append(errs, validateEnvironmentVars(d, "int")...)
				return
			},
			mocks:          nominalMocks,
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "ENVIRONMENT env var empty when aro.environment flag not set",
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeMDSD,
			},
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				if len(d.Spec.Template.Spec.Containers) != 2 {
					errs = append(errs, fmt.Errorf("expected 2 containers, got %d", len(d.Spec.Template.Spec.Containers)))
				}
				errs = append(errs, validateEnvironmentVars(d, "")...)
				return
			},
			mocks:          nominalMocks,
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "otel mode enabled",
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeOTel,
			},
			gatewayPrivateEndpointIP: "10.0.0.8",
			gatewayTelemetryDomain:   "telemetry.eastus.aro.azure.com",
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				return nil
			},
			mocks:          otelNominalMocks,
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "otel mode requires gateway private endpoint ip",
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeOTel,
			},
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				return nil
			},
			mocks:      func(mockDh *mock_dynamichelper.MockInterface) { otelCleanupMocks(mockDh) },
			wantErrMsg: `geneva logging mode "otel" requires cluster spec field "gatewayPrivateEndpointIP"`,
			wantConditions: []operatorv1.OperatorCondition{
				utilconditions.ControllerDefaultAvailable(ControllerName),
				utilconditions.ControllerDefaultProgressing(ControllerName),
				func() operatorv1.OperatorCondition {
					c := utilconditions.ControllerDefaultDegraded(ControllerName)
					c.Status = operatorv1.ConditionTrue
					c.Message = `geneva logging mode "otel" requires cluster spec field "gatewayPrivateEndpointIP"`
					return c
				}(),
			},
		},
		{
			name: "otel mode requires valid gateway private endpoint ip",
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeOTel,
			},
			gatewayPrivateEndpointIP: "not-an-ip",
			gatewayTelemetryDomain:   "telemetry.eastus.aro.azure.com",
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				return nil
			},
			mocks:      func(mockDh *mock_dynamichelper.MockInterface) { otelCleanupMocks(mockDh) },
			wantErrMsg: `invalid cluster spec field "gatewayPrivateEndpointIP": "not-an-ip" is not a valid IP address`,
			wantConditions: []operatorv1.OperatorCondition{
				utilconditions.ControllerDefaultAvailable(ControllerName),
				utilconditions.ControllerDefaultProgressing(ControllerName),
				func() operatorv1.OperatorCondition {
					c := utilconditions.ControllerDefaultDegraded(ControllerName)
					c.Status = operatorv1.ConditionTrue
					c.Message = `invalid cluster spec field "gatewayPrivateEndpointIP": "not-an-ip" is not a valid IP address`
					return c
				}(),
			},
		},
		{
			name: "multiple modes in one flag are rejected",
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    "mdsd,otel",
			},
			validateDaemonset: func(d *appsv1.DaemonSet) (errs []error) {
				return nil
			},
			mocks:      func(mockDh *mock_dynamichelper.MockInterface) {},
			wantErrMsg: `unsupported geneva logging mode "mdsd,otel": expected "mdsd" or "otel"`,
			wantConditions: []operatorv1.OperatorCondition{
				utilconditions.ControllerDefaultAvailable(ControllerName),
				utilconditions.ControllerDefaultProgressing(ControllerName),
				func() operatorv1.OperatorCondition {
					c := utilconditions.ControllerDefaultDegraded(ControllerName)
					c.Status = operatorv1.ConditionTrue
					c.Message = `unsupported geneva logging mode "mdsd,otel": expected "mdsd" or "otel"`
					return c
				}(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Status:     arov1alpha1.ClusterStatus{Conditions: defaultConditions},
				Spec: arov1alpha1.ClusterSpec{
					ResourceID:               testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
					OperatorFlags:            tt.operatorFlags,
					ACRDomain:                "acrDomain",
					GatewayPrivateEndpointIP: tt.gatewayPrivateEndpointIP,
					GatewayTelemetryDomain:   tt.gatewayTelemetryDomain,
				},
			}

			cv := clusterVersion("4.11.0")
			resources := []client.Object{
				instance,
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: operator.Namespace,
						Name:      operator.SecretName,
					},
					Data: map[string][]byte{
						GenevaCertName: {},
						GenevaKeyName:  {},
					},
				},
				&securityv1.SecurityContextConstraints{
					ObjectMeta: metav1.ObjectMeta{
						Name: "privileged",
					},
				},
				&cv,
			}

			mockDh := mock_dynamichelper.NewMockInterface(controller)

			r := &Reconciler{
				AROController: base.AROController{
					Log:    logrus.NewEntry(logrus.StandardLogger()),
					Client: ctrlfake.NewClientBuilder().WithObjects(resources...).Build(),
					Name:   ControllerName,
				},
				dh: mockDh,
			}

			daemonset, err := r.daemonset(instance)
			if err != nil {
				t.Fatal(err)
			}

			errs := tt.validateDaemonset(daemonset)
			for _, err := range errs {
				t.Error(err)
			}

			tt.mocks(mockDh)
			ctx := context.Background()
			_, err = r.Reconcile(ctx, tt.request)

			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)
			utilconditions.AssertControllerConditions(t, ctx, r.Client, tt.wantConditions)
		})
	}
}

func TestModeTransitionCleanup(t *testing.T) {
	tests := []struct {
		name         string
		initialFlags arov1alpha1.OperatorFlags
		nextFlags    arov1alpha1.OperatorFlags
	}{
		{
			name: "mdsd to otel",
			initialFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeMDSD,
			},
			nextFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeOTel,
			},
		},
		{
			name: "otel to mdsd",
			initialFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeOTel,
			},
			nextFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
				operator.GenevaLoggingMode:    operator.GenevaLoggingModeMDSD,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Status:     arov1alpha1.ClusterStatus{Conditions: []operatorv1.OperatorCondition{}},
				Spec: arov1alpha1.ClusterSpec{
					ResourceID:               testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
					OperatorFlags:            tt.initialFlags,
					ACRDomain:                "acrDomain",
					GatewayPrivateEndpointIP: "10.0.0.8",
					GatewayTelemetryDomain:   "telemetry.eastus.aro.azure.com",
				},
			}

			cv := clusterVersion("4.11.0")
			resources := []client.Object{
				instance,
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: operator.Namespace,
						Name:      operator.SecretName,
					},
					Data: map[string][]byte{
						GenevaCertName: {},
						GenevaKeyName:  {},
					},
				},
				&securityv1.SecurityContextConstraints{
					ObjectMeta: metav1.ObjectMeta{
						Name: "privileged",
					},
				},
				&cv,
			}

			mockDh := mock_dynamichelper.NewMockInterface(controller)
			mockDh.EXPECT().EnsureDeleted(gomock.Any(), "DaemonSet.apps", kubeNamespace, "otel-collector-master").Return(nil).Times(1)
			mockDh.EXPECT().EnsureDeleted(gomock.Any(), "DaemonSet.apps", kubeNamespace, "otel-collector-worker").Return(nil).Times(1)
			mockDh.EXPECT().EnsureDeleted(gomock.Any(), "ConfigMap", kubeNamespace, otelConfigMapName).Return(nil).Times(1)
			mockDh.EXPECT().EnsureDeleted(gomock.Any(), "ConfigMap", kubeNamespace, otelGatewayCACMName).Return(nil).Times(1)
			mockDh.EXPECT().EnsureDeleted(gomock.Any(), "ConfigMap", kubeNamespace, legacyGatewayCACMName).Return(nil).Times(1)
			mockDh.EXPECT().EnsureDeleted(gomock.Any(), "DaemonSet.apps", kubeNamespace, "mdsd").Return(nil).Times(1)
			mockDh.EXPECT().EnsureDeleted(gomock.Any(), "ConfigMap", kubeNamespace, "fluent-config").Return(nil).Times(1)
			mockDh.EXPECT().EnsureDeleted(gomock.Any(), "Secret", kubeNamespace, certificatesSecretName).Return(nil).Times(1)
			mockDh.EXPECT().EnsureDeleted(gomock.Any(), "ConfigMap", kubeNamespace, legacyGatewayCACMName).Return(nil).Times(1)
			mockDh.EXPECT().Ensure(
				gomock.Any(),
				gomock.AssignableToTypeOf(&securityv1.SecurityContextConstraints{}),
				gomock.AssignableToTypeOf(&corev1.Namespace{}),
				gomock.AssignableToTypeOf(&corev1.ConfigMap{}),
				gomock.AssignableToTypeOf(&corev1.Secret{}),
				gomock.AssignableToTypeOf(&corev1.ServiceAccount{}),
				gomock.AssignableToTypeOf(&appsv1.DaemonSet{}),
			).Times(1)
			mockDh.EXPECT().Ensure(
				gomock.Any(),
				gomock.AssignableToTypeOf(&securityv1.SecurityContextConstraints{}),
				gomock.AssignableToTypeOf(&corev1.Namespace{}),
				gomock.AssignableToTypeOf(&corev1.ConfigMap{}),
				gomock.AssignableToTypeOf(&corev1.ConfigMap{}),
				gomock.AssignableToTypeOf(&corev1.ServiceAccount{}),
				gomock.AssignableToTypeOf(&appsv1.DaemonSet{}),
				gomock.AssignableToTypeOf(&appsv1.DaemonSet{}),
			).Times(1)

			r := &Reconciler{
				AROController: base.AROController{
					Log:    logrus.NewEntry(logrus.StandardLogger()),
					Client: ctrlfake.NewClientBuilder().WithObjects(resources...).Build(),
					Name:   ControllerName,
				},
				dh: mockDh,
			}

			ctx := context.Background()
			if _, err := r.Reconcile(ctx, ctrl.Request{}); err != nil {
				t.Fatalf("unexpected error on initial reconcile: %v", err)
			}

			updated := &arov1alpha1.Cluster{}
			if err := r.Client.Get(ctx, client.ObjectKey{Name: "cluster"}, updated); err != nil {
				t.Fatalf("unexpected error getting cluster: %v", err)
			}
			updated.Spec.OperatorFlags = tt.nextFlags
			if err := r.Client.Update(ctx, updated); err != nil {
				t.Fatalf("unexpected error updating cluster: %v", err)
			}

			if _, err := r.Reconcile(ctx, ctrl.Request{}); err != nil {
				t.Fatalf("unexpected error on transition reconcile: %v", err)
			}
		})
	}
}

func TestGenevaConfigMapResources(t *testing.T) {
	// Expected number of ENVIRONMENT filters in fluent.conf (one per log input type: journald, containers, audit)
	const expectedEnvironmentFilterCount = 3

	tests := []struct {
		name                     string
		mode                     loggingMode
		request                  ctrl.Request
		operatorFlags            arov1alpha1.OperatorFlags
		gatewayPrivateEndpointIP string
		gatewayTelemetryDomain   string
		validate                 func([]runtime.Object) []error
	}{
		{
			name: "mdsd mode configmaps",
			mode: loggingModeMDSD,
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled: operator.FlagTrue,
			},
			validate: func(r []runtime.Object) (errs []error) {
				maps := make(map[string]*corev1.ConfigMap)
				for _, i := range r {
					if d, ok := i.(*corev1.ConfigMap); ok {
						maps[d.Name] = d
					}
				}

				c, ok := maps["fluent-config"]
				if !ok {
					errs = append(errs, errors.New("missing fluent-config ConfigMap"))
				} else {
					fConf := c.Data["fluent.conf"]
					pConf := c.Data["parsers.conf"]

					if !strings.Contains(fConf, "[INPUT]") {
						errs = append(errs, errors.New("fluent.conf missing required [INPUT] section"))
					}

					if !strings.Contains(pConf, "[PARSER]") {
						errs = append(errs, errors.New("parsers.conf missing required [PARSER] section"))
					}

					// verify ENVIRONMENT filters are present for all log types
					if !strings.Contains(fConf, "Add Environment ${ENVIRONMENT}") {
						errs = append(errs, errors.New("fluent.conf missing ENVIRONMENT filter - logs will not include environment field"))
					}

					// count how many times the ENVIRONMENT filter appears (should match expectedEnvironmentFilterCount)
					environmentFilterCount := strings.Count(fConf, "Add Environment ${ENVIRONMENT}")
					if environmentFilterCount != expectedEnvironmentFilterCount {
						errs = append(errs, fmt.Errorf("expected %d ENVIRONMENT filters in fluent.conf (journald, containers, audit), got %d", expectedEnvironmentFilterCount, environmentFilterCount))
					}
				}

				return
			},
		},
		{
			name: "otel mode configmaps full profile",
			mode: loggingModeOTel,
			operatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled:     operator.FlagTrue,
				operator.GenevaLoggingMode:        operator.GenevaLoggingModeOTel,
				operator.GenevaLoggingOTelProfile: operator.GenevaLoggingOTelProfileFull,
			},
			gatewayPrivateEndpointIP: "10.0.0.8",
			gatewayTelemetryDomain:   "telemetry.eastus.aro.azure.com",
			validate: func(r []runtime.Object) (errs []error) {
				maps := make(map[string]*corev1.ConfigMap)
				secrets := make(map[string]*corev1.Secret)
				for _, i := range r {
					if d, ok := i.(*corev1.ConfigMap); ok {
						maps[d.Name] = d
					}
					if s, ok := i.(*corev1.Secret); ok {
						secrets[s.Name] = s
					}
				}

				otelConfigMap, ok := maps[otelConfigMapName]
				if !ok {
					errs = append(errs, errors.New("missing otel-config ConfigMap"))
				} else {
					cfg := otelConfigMap.Data["config.yaml"]
					if !strings.Contains(cfg, "memory_limiter:") {
						errs = append(errs, errors.New("otel config missing memory limiter processor"))
					}
					if !strings.Contains(cfg, "health_check") {
						errs = append(errs, errors.New("otel config missing health_check extension"))
					}
					if !strings.Contains(cfg, "file_storage:") || !strings.Contains(cfg, "storage: file_storage") {
						errs = append(errs, errors.New("otel config missing persistent file_storage tracking"))
					}
					if !strings.Contains(cfg, "journald:") || !strings.Contains(cfg, "file_log/containers:") || !strings.Contains(cfg, "file_log/audit:") {
						errs = append(errs, errors.New("otel config missing required journald/container/audit receivers"))
					}
					if !strings.Contains(cfg, "receivers: [journald, file_log/containers, file_log/audit]") {
						errs = append(errs, errors.New("otel config logs pipeline missing journald receiver"))
					}
					if !strings.Contains(cfg, "/var/log/containers/*_default_*.log") ||
						!strings.Contains(cfg, "/var/log/containers/*_kube-*_*.log") ||
						!strings.Contains(cfg, "/var/log/containers/*_openshift_*.log") ||
						!strings.Contains(cfg, "/var/log/containers/*_openshift-*_*.log") {
						errs = append(errs, errors.New("otel config missing namespace include parity for default/kube*/openshift*"))
					}
					if !strings.Contains(cfg, "/var/log/containers/*_openshift-logging_*.log") {
						errs = append(errs, errors.New("otel config missing namespace exclusion parity for openshift-logging"))
					}
					if !strings.Contains(cfg, "transform/log-parity:") {
						errs = append(errs, errors.New("otel config missing log parity transform processor"))
					}
					if !strings.Contains(cfg, `delete_matching_keys(log.attributes, "^_")`) ||
						!strings.Contains(cfg, `delete_key(log.attributes, "TIMESTAMP")`) {
						errs = append(errs, errors.New("otel config missing journald key pruning parity"))
					}
					if !strings.Contains(cfg, `set(log.body["user_username"], log.body["user"]["username"]) where IsMap(log.body) and IsMap(log.body["user"]) and log.body["user"]["username"] != nil`) ||
						!strings.Contains(cfg, `set(log.body["user_groups"], log.body["user"]["groups"]) where IsMap(log.body) and IsMap(log.body["user"]) and log.body["user"]["groups"] != nil`) ||
						!strings.Contains(cfg, `set(log.body["responseStatus_code"], log.body["responseStatus"]["code"]) where IsMap(log.body) and IsMap(log.body["responseStatus"]) and log.body["responseStatus"]["code"] != nil`) ||
						!strings.Contains(cfg, `set(log.body["responseStatus_message"], log.body["responseStatus"]["message"]) where IsMap(log.body) and IsMap(log.body["responseStatus"]) and log.body["responseStatus"]["message"] != nil`) ||
						!strings.Contains(cfg, `set(log.body["objectRef_resource"], log.body["objectRef"]["resource"]) where IsMap(log.body) and IsMap(log.body["objectRef"]) and log.body["objectRef"]["resource"] != nil`) {
						errs = append(errs, errors.New("otel config missing audit flattening parity"))
					}
					if !strings.Contains(cfg, `set(log.attributes["NODE"], "${env:MONITORING_ROLE_INSTANCE}")`) {
						errs = append(errs, errors.New("otel config missing NODE attribute sourced from MONITORING_ROLE_INSTANCE"))
					}
					if strings.Contains(cfg, `delete_key(body, "user")`) ||
						strings.Contains(cfg, `delete_key(body, "impersonatedUser")`) ||
						strings.Contains(cfg, `delete_key(body, "responseStatus")`) ||
						strings.Contains(cfg, `delete_key(body, "objectRef")`) {
						errs = append(errs, errors.New("otel config should preserve nested audit objects to avoid dropping unmapped keys"))
					}
					if !strings.Contains(cfg, "processors: [memory_limiter, transform/log-parity, batch]") {
						errs = append(errs, errors.New("otel config logs pipeline missing expected processors"))
					}
					if strings.Contains(cfg, "logs/degraded-info-sampled:") {
						errs = append(errs, errors.New("otel config should not include sampled degraded-info pipeline"))
					}
					if strings.Contains(cfg, "filter/drop-olm-noise:") || strings.Contains(cfg, "filter/keep-only-high-signal:") {
						errs = append(errs, errors.New("otel full profile config should not include reduced-noise or high-signal processors"))
					}
				}

				if _, ok := maps["fluent-config"]; ok {
					errs = append(errs, errors.New("fluent-config ConfigMap should not be present in otel mode"))
				}
				if _, ok := secrets[certificatesSecretName]; ok {
					errs = append(errs, errors.New("certificates secret should not be present in otel mode"))
				}
				gatewayCACM, ok := maps[otelGatewayCACMName]
				if !ok {
					errs = append(errs, fmt.Errorf("missing %s ConfigMap", otelGatewayCACMName))
				} else if gatewayCACM.Labels["config.openshift.io/inject-trusted-cabundle"] != "true" {
					errs = append(errs, fmt.Errorf("%s ConfigMap missing trusted CA injection label", otelGatewayCACMName))
				}

				return
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Status:     arov1alpha1.ClusterStatus{Conditions: []operatorv1.OperatorCondition{}},
				Spec: arov1alpha1.ClusterSpec{
					ResourceID:               testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
					OperatorFlags:            tt.operatorFlags,
					ACRDomain:                "acrDomain",
					GatewayPrivateEndpointIP: tt.gatewayPrivateEndpointIP,
					GatewayTelemetryDomain:   tt.gatewayTelemetryDomain,
				},
			}

			scc := &securityv1.SecurityContextConstraints{
				ObjectMeta: metav1.ObjectMeta{Name: "privileged"},
			}

			cv := clusterVersion("4.11.0")
			r := &Reconciler{
				AROController: base.AROController{
					Log:    logrus.NewEntry(logrus.StandardLogger()),
					Client: ctrlfake.NewClientBuilder().WithObjects(instance, scc, &cv).Build(),
					Name:   ControllerName,
				},
			}

			out, err := r.resources(context.Background(), instance, tt.mode, []byte{}, []byte{})
			if err != nil {
				t.Fatal(err)
			}

			errs := tt.validate(out)
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}

func TestGenevaOTelConfigMapUsesReducedNoiseProfile(t *testing.T) {
	instance := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     arov1alpha1.ClusterStatus{Conditions: []operatorv1.OperatorCondition{}},
		Spec: arov1alpha1.ClusterSpec{
			ResourceID:               testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
			OperatorFlags:            arov1alpha1.OperatorFlags{operator.GenevaLoggingEnabled: operator.FlagTrue, operator.GenevaLoggingMode: operator.GenevaLoggingModeOTel, operator.GenevaLoggingOTelProfile: operator.GenevaLoggingOTelProfileReduced},
			ACRDomain:                "acrDomain",
			GatewayPrivateEndpointIP: "10.0.0.8",
			GatewayTelemetryDomain:   "telemetry.eastus.aro.azure.com",
		},
	}

	scc := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{Name: "privileged"},
	}
	cv := clusterVersion("4.11.0")
	r := &Reconciler{
		AROController: base.AROController{
			Log:    logrus.NewEntry(logrus.StandardLogger()),
			Client: ctrlfake.NewClientBuilder().WithObjects(instance, scc, &cv).Build(),
			Name:   ControllerName,
		},
	}

	out, err := r.resources(context.Background(), instance, loggingModeOTel, []byte{}, []byte{})
	if err != nil {
		t.Fatal(err)
	}

	var otelCfg string
	for _, obj := range out {
		cm, ok := obj.(*corev1.ConfigMap)
		if !ok || cm.Name != otelConfigMapName {
			continue
		}
		otelCfg = cm.Data["config.yaml"]
	}
	if otelCfg == "" {
		t.Fatal("missing otel-config ConfigMap data")
	}

	if !strings.Contains(otelCfg, "filter/drop-olm-noise:") {
		t.Fatal("expected reduced-noise OTEL config profile with filter/drop-olm-noise")
	}
	if !strings.Contains(otelCfg, "processors: [memory_limiter, filter/drop-olm-noise, transform/log-parity, batch]") {
		t.Fatal("expected reduced OTEL config logs pipeline to include OLM noise drop filter")
	}
}

func TestGenevaOTelConfigMapSupportsRoleSpecificProfiles(t *testing.T) {
	instance := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     arov1alpha1.ClusterStatus{Conditions: []operatorv1.OperatorCondition{}},
		Spec: arov1alpha1.ClusterSpec{
			ResourceID: testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.GenevaLoggingEnabled:           operator.FlagTrue,
				operator.GenevaLoggingMode:              operator.GenevaLoggingModeOTel,
				operator.GenevaLoggingOTelMasterProfile: operator.GenevaLoggingOTelProfileHighLogLevel,
				operator.GenevaLoggingOTelWorkerProfile: operator.GenevaLoggingOTelProfileMinimalLogs,
			},
			ACRDomain:                "acrDomain",
			GatewayPrivateEndpointIP: "10.0.0.8",
			GatewayTelemetryDomain:   "telemetry.eastus.aro.azure.com",
		},
	}

	scc := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{Name: "privileged"},
	}
	cv := clusterVersion("4.11.0")
	r := &Reconciler{
		AROController: base.AROController{
			Log:    logrus.NewEntry(logrus.StandardLogger()),
			Client: ctrlfake.NewClientBuilder().WithObjects(instance, scc, &cv).Build(),
			Name:   ControllerName,
		},
	}

	out, err := r.resources(context.Background(), instance, loggingModeOTel, []byte{}, []byte{})
	if err != nil {
		t.Fatal(err)
	}

	var cm *corev1.ConfigMap
	for _, obj := range out {
		c, ok := obj.(*corev1.ConfigMap)
		if ok && c.Name == otelConfigMapName {
			cm = c
			break
		}
	}
	if cm == nil {
		t.Fatal("missing otel-config ConfigMap")
	}

	masterCfg := cm.Data[otelMasterConfigKey]
	workerCfg := cm.Data[otelWorkerConfigKey]
	if masterCfg == "" || workerCfg == "" {
		t.Fatalf("expected both %q and %q in otel configmap", otelMasterConfigKey, otelWorkerConfigKey)
	}
	if cm.Data["config.yaml"] != masterCfg {
		t.Fatal(`expected config.yaml to mirror master config for compatibility`)
	}
	if strings.Contains(masterCfg, "filter/keep-only-high-signal:") {
		t.Fatal("master high-loglevel profile should not include minimal-logs filter")
	}
	if !strings.Contains(workerCfg, "filter/keep-only-high-signal:") {
		t.Fatal("worker minimal-logs profile should include keep-only-high-signal filter")
	}
}

func TestOTelConfigYAMLStructure(t *testing.T) {
	var cfg map[string]interface{}
	if err := yaml.Unmarshal([]byte(otelConfigFull), &cfg); err != nil {
		t.Fatalf("failed to parse otel config yaml: %v", err)
	}

	getMap := func(parent map[string]interface{}, key string) map[string]interface{} {
		t.Helper()
		v, ok := parent[key]
		if !ok {
			t.Fatalf("missing key %q", key)
		}
		m, ok := v.(map[string]interface{})
		if !ok {
			t.Fatalf("key %q is not an object", key)
		}
		return m
	}

	getString := func(parent map[string]interface{}, key string) string {
		t.Helper()
		v, ok := parent[key]
		if !ok {
			t.Fatalf("missing key %q", key)
		}
		s, ok := v.(string)
		if !ok {
			t.Fatalf("key %q is not a string", key)
		}
		return s
	}

	getStringList := func(parent map[string]interface{}, key string) []string {
		t.Helper()
		v, ok := parent[key]
		if !ok {
			t.Fatalf("missing key %q", key)
		}
		raw, ok := v.([]interface{})
		if !ok {
			t.Fatalf("key %q is not a list", key)
		}
		out := make([]string, 0, len(raw))
		for i, item := range raw {
			s, ok := item.(string)
			if !ok {
				t.Fatalf("key %q has non-string entry at index %d", key, i)
			}
			out = append(out, s)
		}
		return out
	}

	getBool := func(parent map[string]interface{}, key string) bool {
		t.Helper()
		v, ok := parent[key]
		if !ok {
			t.Fatalf("missing key %q", key)
		}
		b, ok := v.(bool)
		if !ok {
			t.Fatalf("key %q is not a bool", key)
		}
		return b
	}

	getObjectList := func(parent map[string]interface{}, key string) []map[string]interface{} {
		t.Helper()
		v, ok := parent[key]
		if !ok {
			t.Fatalf("missing key %q", key)
		}
		raw, ok := v.([]interface{})
		if !ok {
			t.Fatalf("key %q is not a list", key)
		}
		out := make([]map[string]interface{}, 0, len(raw))
		for i, item := range raw {
			obj, ok := item.(map[string]interface{})
			if !ok {
				t.Fatalf("key %q has non-object entry at index %d", key, i)
			}
			out = append(out, obj)
		}
		return out
	}

	receivers := getMap(cfg, "receivers")
	journald := getMap(receivers, "journald")
	if len(journald) == 0 {
		t.Fatal("missing journald receiver")
	}
	if got := getString(journald, "directory"); got != "/var/log/journal" {
		t.Fatalf("unexpected journald directory %q", got)
	}
	if got := getString(journald, "journalctl_path"); got != "/usr/bin/journalctl" {
		t.Fatalf("unexpected journald journalctl_path %q", got)
	}
	if got := getString(journald, "start_at"); got != "beginning" {
		t.Fatalf("unexpected journald start_at %q", got)
	}
	if got := getString(journald, "storage"); got != "file_storage" {
		t.Fatalf("unexpected journald storage %q", got)
	}
	if _, ok := receivers["file_log/containers"]; !ok {
		t.Fatal("missing file_log/containers receiver")
	}
	audit := getMap(receivers, "file_log/audit")
	auditIncludes := getStringList(audit, "include")
	if !reflect.DeepEqual(auditIncludes, []string{"/var/log/kube-apiserver/audit.log"}) {
		t.Fatalf("unexpected file_log/audit include list: got %v", auditIncludes)
	}
	if got := getString(audit, "start_at"); got != "end" {
		t.Fatalf("unexpected file_log/audit start_at %q", got)
	}
	if got := getString(audit, "storage"); got != "file_storage" {
		t.Fatalf("unexpected file_log/audit storage %q", got)
	}
	if got := getString(audit, "on_truncate"); got != "read_whole_file" {
		t.Fatalf("unexpected file_log/audit on_truncate %q", got)
	}
	if got := getBool(audit, "include_file_path"); !got {
		t.Fatal("expected file_log/audit include_file_path=true")
	}
	auditOperators := getObjectList(audit, "operators")
	if len(auditOperators) != 1 {
		t.Fatalf("expected one file_log/audit operator, got %d", len(auditOperators))
	}
	jsonParser := auditOperators[0]
	if got := getString(jsonParser, "type"); got != "json_parser" {
		t.Fatalf("unexpected file_log/audit operator type %q", got)
	}
	if got := getString(jsonParser, "parse_from"); got != "body" {
		t.Fatalf("unexpected file_log/audit parse_from %q", got)
	}
	if got := getString(jsonParser, "parse_to"); got != "body" {
		t.Fatalf("unexpected file_log/audit parse_to %q", got)
	}
	timestamp := getMap(jsonParser, "timestamp")
	if got := getString(timestamp, "parse_from"); got != "body.stageTimestamp" {
		t.Fatalf("unexpected file_log/audit timestamp parse_from %q", got)
	}
	if got := getString(timestamp, "layout_type"); got != "strptime" {
		t.Fatalf("unexpected file_log/audit timestamp layout_type %q", got)
	}
	if got := getString(timestamp, "layout"); got != "%Y-%m-%dT%H:%M:%S.%LZ" {
		t.Fatalf("unexpected file_log/audit timestamp layout %q", got)
	}

	processors := getMap(cfg, "processors")
	for _, key := range []string{
		"memory_limiter",
		"transform/log-parity",
		"batch",
	} {
		if _, ok := processors[key]; !ok {
			t.Fatalf("missing processor %q", key)
		}
	}
	memoryLimiter := getMap(processors, "memory_limiter")
	if got := memoryLimiter["limit_mib"]; got != float64(1000) {
		t.Fatalf("unexpected memory_limiter limit_mib: got %v, want %v", got, 1000)
	}
	if got := memoryLimiter["spike_limit_mib"]; got != float64(150) {
		t.Fatalf("unexpected memory_limiter spike_limit_mib: got %v, want %v", got, 150)
	}

	exporters := getMap(cfg, "exporters")
	gatewayExporter := getMap(exporters, "otlp_grpc/gateway")
	if got := getString(gatewayExporter, "endpoint"); got != "${env:GENEVA_GATEWAY_ENDPOINT}" {
		t.Fatalf("unexpected gateway exporter endpoint %q", got)
	}

	extensions := getMap(cfg, "extensions")
	if _, ok := extensions["health_check"]; !ok {
		t.Fatal("missing health_check extension")
	}
	if _, ok := extensions["file_storage"]; !ok {
		t.Fatal("missing file_storage extension")
	}

	service := getMap(cfg, "service")
	pipelines := getMap(service, "pipelines")
	logsPipeline := getMap(pipelines, "logs")

	wantReceivers := []string{"journald", "file_log/containers", "file_log/audit"}
	if got := getStringList(logsPipeline, "receivers"); !reflect.DeepEqual(got, wantReceivers) {
		t.Fatalf("unexpected logs pipeline receivers: got %v, want %v", got, wantReceivers)
	}

	wantProcessors := []string{"memory_limiter", "transform/log-parity", "batch"}
	if got := getStringList(logsPipeline, "processors"); !reflect.DeepEqual(got, wantProcessors) {
		t.Fatalf("unexpected logs pipeline processors: got %v, want %v", got, wantProcessors)
	}

	wantExporters := []string{"otlp_grpc/gateway"}
	if got := getStringList(logsPipeline, "exporters"); !reflect.DeepEqual(got, wantExporters) {
		t.Fatalf("unexpected logs pipeline exporters: got %v, want %v", got, wantExporters)
	}

	containerReceiver := getMap(receivers, "file_log/containers")
	if got := getString(containerReceiver, "start_at"); got != "beginning" {
		t.Fatalf("unexpected file_log/containers start_at %q", got)
	}
	if got := getString(containerReceiver, "storage"); got != "file_storage" {
		t.Fatalf("unexpected file_log/containers storage %q", got)
	}
	exclude := getStringList(containerReceiver, "exclude")
	if len(exclude) != 10 {
		t.Fatalf("expected 10 excluded namespace patterns, got %d", len(exclude))
	}
	foundKEDAExclude := false
	for _, p := range exclude {
		if p == "/var/log/containers/*_openshift-keda_*.log" {
			foundKEDAExclude = true
			break
		}
	}
	if !foundKEDAExclude {
		t.Fatal("expected file_log/containers exclude list to include openshift-keda")
	}
	if !getBool(containerReceiver, "include_file_path") {
		t.Fatal("expected file_log/containers include_file_path to be true")
	}
	if !getBool(containerReceiver, "include_file_path_resolved") {
		t.Fatal("expected file_log/containers include_file_path_resolved to be true")
	}
	containerOperators := getObjectList(containerReceiver, "operators")
	if len(containerOperators) != 5 {
		t.Fatalf("unexpected file_log/containers operators: %+v", containerOperators)
	}
	resolvedPathMove := containerOperators[0]
	if got := getString(resolvedPathMove, "type"); got != "move" {
		t.Fatalf("unexpected resolved path operator type %q", got)
	}
	if got := getString(resolvedPathMove, "from"); got != `attributes["log.file.path_resolved"]` {
		t.Fatalf("unexpected resolved path operator from %q", got)
	}
	if got := getString(resolvedPathMove, "to"); got != `attributes["log.file.path"]` {
		t.Fatalf("unexpected resolved path operator to %q", got)
	}
	if got := getString(resolvedPathMove, "if"); got != `attributes["log.file.path_resolved"] != nil` {
		t.Fatalf("unexpected resolved path operator if %q", got)
	}
	if got := getString(containerOperators[1], "type"); got != "container" {
		t.Fatalf("unexpected file_log/containers second operator type %q", got)
	}
	klogParser := containerOperators[2]
	if got := getString(klogParser, "id"); got != "klog_parse" {
		t.Fatalf("unexpected klog parser id %q", got)
	}
	if got := getString(klogParser, "type"); got != "regex_parser" {
		t.Fatalf("unexpected klog parser type %q", got)
	}
	if got := getString(klogParser, "parse_from"); got != "body" {
		t.Fatalf("unexpected klog parser parse_from %q", got)
	}
	if got := getString(klogParser, "parse_to"); got != "body" {
		t.Fatalf("unexpected klog parser parse_to %q", got)
	}
	if got := getString(klogParser, "on_error"); got != "send_quiet" {
		t.Fatalf("unexpected klog parser on_error %q", got)
	}
	severity := getMap(klogParser, "severity")
	if got := getString(severity, "parse_from"); got != "body.severity" {
		t.Fatalf("unexpected klog parser severity parse_from %q", got)
	}
	severityMapping := getMap(severity, "mapping")
	if got := getString(severityMapping, "fatal"); got != "F" {
		t.Fatalf("unexpected klog parser fatal severity mapping %q", got)
	}
	klogValueParser := containerOperators[3]
	if got := getString(klogValueParser, "id"); got != "klog_value_parse" {
		t.Fatalf("unexpected klog key-value parser id %q", got)
	}
	if got := getString(klogValueParser, "type"); got != "key_value_parser" {
		t.Fatalf("unexpected klog key-value parser type %q", got)
	}
	if got := getString(klogValueParser, "if"); got != `type(body) == "map"` {
		t.Fatalf("unexpected klog key-value parser if %q", got)
	}
	if got := getString(klogValueParser, "parse_from"); got != "body.message" {
		t.Fatalf("unexpected klog key-value parser parse_from %q", got)
	}
	if got := getString(klogValueParser, "parse_to"); got != "body.fields" {
		t.Fatalf("unexpected klog key-value parser parse_to %q", got)
	}
	if got := getString(klogValueParser, "on_error"); got != "send_quiet" {
		t.Fatalf("unexpected klog key-value parser on_error %q", got)
	}
	flatten := containerOperators[4]
	if got := getString(flatten, "type"); got != "flatten" {
		t.Fatalf("unexpected klog flatten type %q", got)
	}
	if got := getString(flatten, "field"); got != "body" {
		t.Fatalf("unexpected klog flatten field %q", got)
	}
	if got := getString(flatten, "if"); got != `type(body) == "map"` {
		t.Fatalf("unexpected klog flatten if %q", got)
	}

	journaldReceiver := getMap(receivers, "journald")
	journaldOperators := getObjectList(journaldReceiver, "operators")
	if len(journaldOperators) != 1 {
		t.Fatalf("unexpected journald operators length: got %d, want 1", len(journaldOperators))
	}
	if got := getString(journaldOperators[0], "id"); got != "kubenswrapper-multiline" {
		t.Fatalf("unexpected journald operator id: got %q, want %q", got, "kubenswrapper-multiline")
	}
	if got := getString(journaldOperators[0], "type"); got != "recombine" {
		t.Fatalf("unexpected journald operator type: got %q, want %q", got, "recombine")
	}
	if got := getString(journaldOperators[0], "if"); got != "attributes[\"SYSLOG_IDENTIFIER\"] == \"kubenswrapper\"" {
		t.Fatalf("unexpected journald operator if: got %q", got)
	}
	if got := getString(journaldOperators[0], "combine_field"); got != "body" {
		t.Fatalf("unexpected journald operator combine_field: got %q, want %q", got, "body")
	}
	if got := getString(journaldOperators[0], "is_first_entry"); got != "body matches \"^[^\\\\s]\"" {
		t.Fatalf("unexpected journald operator is_first_entry: got %q", got)
	}
}

func TestOTelConfigAuditReceiverParity(t *testing.T) {
	tests := []struct {
		name string
		cfg  string
	}{
		{
			name: "full",
			cfg:  otelConfigFull,
		},
		{
			name: "reduced-noise",
			cfg:  otelConfigReducedNoise,
		},
		{
			name: "high-signal",
			cfg:  otelConfigHighSignal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, expected := range []string{
				"file_log/audit:",
				"- /var/log/kube-apiserver/audit.log",
				"on_truncate: read_whole_file",
				"include_file_path: true",
				"- type: json_parser",
				"parse_from: body",
				"parse_to: body",
				"parse_from: body.stageTimestamp",
				`layout: "%Y-%m-%dT%H:%M:%S.%LZ"`,
				"receivers: [journald, file_log/containers, file_log/audit]",
			} {
				if !strings.Contains(tt.cfg, expected) {
					t.Fatalf("audit receiver config missing %q", expected)
				}
			}
		})
	}
}

func TestOTelConfigJournaldReceiverParity(t *testing.T) {
	tests := []struct {
		name string
		cfg  string
	}{
		{
			name: "full",
			cfg:  otelConfigFull,
		},
		{
			name: "reduced-noise",
			cfg:  otelConfigReducedNoise,
		},
		{
			name: "high-signal",
			cfg:  otelConfigHighSignal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, expected := range []string{
				"journald:",
				"directory: /var/log/journal",
				"journalctl_path: /usr/bin/journalctl",
				"start_at: beginning",
				"storage: file_storage",
			} {
				if !strings.Contains(tt.cfg, expected) {
					t.Fatalf("journald receiver config missing %q", expected)
				}
			}
		})
	}
}

func TestOTelConfigContainerReceiverParity(t *testing.T) {
	tests := []struct {
		name string
		cfg  string
	}{
		{
			name: "full",
			cfg:  otelConfigFull,
		},
		{
			name: "reduced-noise",
			cfg:  otelConfigReducedNoise,
		},
		{
			name: "high-signal",
			cfg:  otelConfigHighSignal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, expected := range []string{
				"file_log/containers:",
				"start_at: beginning",
				"storage: file_storage",
				"include_file_path: true",
				"include_file_path_resolved: true",
				"- type: move",
				`from: attributes["log.file.path_resolved"]`,
				`to: attributes["log.file.path"]`,
				`if: 'attributes["log.file.path_resolved"] != nil'`,
				"- type: container",
				"id: klog_parse",
				"type: regex_parser",
				"parse_from: body",
				"parse_to: body",
				"on_error: send_quiet",
				"id: klog_value_parse",
				"type: key_value_parser",
				"parse_from: body.message",
				"parse_to: body.fields",
				"type: flatten",
				"field: body",
				`if: 'type(body) == "map"'`,
				`set(log.body["user_username"], log.body["user"]["username"]) where IsMap(log.body) and IsMap(log.body["user"]) and log.body["user"]["username"] != nil`,
				`set(log.attributes["CONTAINER"], resource.attributes["k8s.container.name"]) where resource.attributes["k8s.container.name"] != nil`,
				`set(log.attributes["POD"], resource.attributes["k8s.pod.name"]) where resource.attributes["k8s.pod.name"] != nil`,
				`set(log.attributes["NAMESPACE"], resource.attributes["k8s.namespace.name"]) where resource.attributes["k8s.namespace.name"] != nil`,
				"receivers: [journald, file_log/containers, file_log/audit]",
			} {
				if !strings.Contains(tt.cfg, expected) {
					t.Fatalf("container receiver config missing %q", expected)
				}
			}
		})
	}
}

func TestOTelConfigRHKeysMissingYAMLStructure(t *testing.T) {
	var cfg map[string]interface{}
	if err := yaml.Unmarshal([]byte(otelConfigReducedNoise), &cfg); err != nil {
		t.Fatalf("failed to parse otel RH-keys-missing config yaml: %v", err)
	}

	processors, ok := cfg["processors"].(map[string]interface{})
	if !ok {
		t.Fatal("missing processors section")
	}
	if _, ok := processors["filter/drop-olm-noise"]; !ok {
		t.Fatal("missing filter/drop-olm-noise processor")
	}

	service, ok := cfg["service"].(map[string]interface{})
	if !ok {
		t.Fatal("missing service section")
	}
	pipelines, ok := service["pipelines"].(map[string]interface{})
	if !ok {
		t.Fatal("missing service.pipelines section")
	}

	logsPipeline, ok := pipelines["logs"].(map[string]interface{})
	if !ok {
		t.Fatal("missing logs pipeline")
	}
	logsProcessors, ok := logsPipeline["processors"].([]interface{})
	if !ok {
		t.Fatal("logs processors should be a list")
	}

	foundNoiseFilter := false
	for _, p := range logsProcessors {
		if s, ok := p.(string); ok && s == "filter/drop-olm-noise" {
			foundNoiseFilter = true
		}
	}
	if !foundNoiseFilter {
		t.Fatal("logs pipeline should include filter/drop-olm-noise")
	}

	receivers, ok := cfg["receivers"].(map[string]interface{})
	if !ok {
		t.Fatal("missing receivers section")
	}
	containerReceiver, ok := receivers["file_log/containers"].(map[string]interface{})
	if !ok {
		t.Fatal("missing file_log/containers receiver")
	}
	include, ok := containerReceiver["include"].([]interface{})
	if !ok {
		t.Fatal("file_log/containers include should be a list")
	}
	includeSet := map[string]bool{}
	for _, entry := range include {
		if s, ok := entry.(string); ok {
			includeSet[s] = true
		}
	}

	for _, expected := range []string{
		"/var/log/pods/openshift-azure-*_*/*/*.log",
		"/var/log/pods/openshift-machine-api_*/*/*.log",
		"/var/log/pods/openshift-monitoring_*/*/*.log",
	} {
		if !includeSet[expected] {
			t.Fatalf("expected file_log/containers include list to include %q", expected)
		}
	}

	for _, denied := range []string{
		"/var/log/pods/openshift-marketplace_*/*/*.log",
		"/var/log/pods/openshift-operator-lifecycle-manager_*/*/*.log",
		"/var/log/pods/openshift-openstack-infra_*/*/*.log",
		"/var/log/pods/openshift-insights_*/*/*.log",
		"/var/log/pods/openshift-keda_*/*/*.log",
		"/var/log/pods/openshift-cluster-storage-operator_*/*/*.log",
	} {
		if includeSet[denied] {
			t.Fatalf("did not expect file_log/containers include list to include denied namespace pattern %q", denied)
		}
	}
}

func TestOTelConfigHighSignalYAMLStructure(t *testing.T) {
	var cfg map[string]interface{}
	if err := yaml.Unmarshal([]byte(otelConfigHighSignal), &cfg); err != nil {
		t.Fatalf("failed to parse otel high-signal config yaml: %v", err)
	}

	processors, ok := cfg["processors"].(map[string]interface{})
	if !ok {
		t.Fatal("missing processors section")
	}
	if _, ok := processors["filter/keep-only-high-signal"]; !ok {
		t.Fatal("missing filter/keep-only-high-signal processor")
	}

	service, ok := cfg["service"].(map[string]interface{})
	if !ok {
		t.Fatal("missing service section")
	}
	pipelines, ok := service["pipelines"].(map[string]interface{})
	if !ok {
		t.Fatal("missing service.pipelines section")
	}
	logsPipeline, ok := pipelines["logs"].(map[string]interface{})
	if !ok {
		t.Fatal("missing logs pipeline")
	}
	logsProcessors, ok := logsPipeline["processors"].([]interface{})
	if !ok {
		t.Fatal("logs processors should be a list")
	}

	foundHighSignalFilter := false
	for _, p := range logsProcessors {
		if s, ok := p.(string); ok && s == "filter/keep-only-high-signal" {
			foundHighSignalFilter = true
		}
	}
	if !foundHighSignalFilter {
		t.Fatal("logs pipeline should include filter/keep-only-high-signal")
	}

	receivers, ok := cfg["receivers"].(map[string]interface{})
	if !ok {
		t.Fatal("missing receivers section")
	}
	containerReceiver, ok := receivers["file_log/containers"].(map[string]interface{})
	if !ok {
		t.Fatal("missing file_log/containers receiver")
	}
	include, ok := containerReceiver["include"].([]interface{})
	if !ok {
		t.Fatal("file_log/containers include should be a list")
	}
	includeSet := map[string]bool{}
	for _, entry := range include {
		if s, ok := entry.(string); ok {
			includeSet[s] = true
		}
	}
	for _, expected := range []string{
		"/var/log/containers/*_openshift-azure-*_*.log",
		"/var/log/containers/*_openshift-machine-api_*.log",
		"/var/log/containers/*_openshift-kube-apiserver_*.log",
	} {
		if !includeSet[expected] {
			t.Fatalf("expected file_log/containers include list to include %q", expected)
		}
	}

	for _, denied := range []string{
		"/var/log/containers/*_openshift-marketplace_*.log",
		"/var/log/containers/*_openshift-operator-lifecycle-manager_*.log",
		"/var/log/containers/*_openshift-openstack-infra_*.log",
		"/var/log/containers/*_openshift-insights_*.log",
		"/var/log/containers/*_openshift-keda_*.log",
		"/var/log/containers/*_openshift-cluster-storage-operator_*.log",
	} {
		if includeSet[denied] {
			t.Fatalf("did not expect file_log/containers include list to include denied namespace pattern %q", denied)
		}
	}
}

func TestOTelDaemonSets(t *testing.T) {
	tests := []struct {
		name                string
		flags               arov1alpha1.OperatorFlags
		gatewayEndpoint     string
		wantImage           string
		wantGatewayEndpoint string
	}{
		{
			name:                "default image",
			flags:               arov1alpha1.OperatorFlags{},
			gatewayEndpoint:     "10.0.0.8:4317",
			wantImage:           version.OTelImage("acrDomain"),
			wantGatewayEndpoint: "10.0.0.8:4317",
		},
		{
			name: "custom image",
			flags: arov1alpha1.OperatorFlags{
				controllerOTelPullSpec: "test.local/otel:custom",
			},
			gatewayEndpoint:     "10.0.0.9:4317",
			wantImage:           "test.local/otel:custom",
			wantGatewayEndpoint: "10.0.0.9:4317",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{}
			daemonsets, err := r.otelDaemonSets(&arov1alpha1.Cluster{
				Spec: arov1alpha1.ClusterSpec{
					ResourceID:    testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"),
					ACRDomain:     "acrDomain",
					OperatorFlags: tt.flags,
				},
			}, tt.gatewayEndpoint, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(daemonsets) != 2 {
				t.Fatalf("expected 2 daemonsets, got %d", len(daemonsets))
			}

			byName := map[string]*appsv1.DaemonSet{}
			for _, ds := range daemonsets {
				byName[ds.Name] = ds
			}
			masterDS := byName["otel-collector-master"]
			workerDS := byName["otel-collector-worker"]
			if masterDS == nil || workerDS == nil {
				t.Fatalf("expected both master and worker daemonsets, got keys: %v", reflect.ValueOf(byName).MapKeys())
			}

			assertCommon := func(ds *appsv1.DaemonSet, wantCPULimit string) {
				collector, err := getContainer(ds, "otel-collector")
				if err != nil {
					t.Fatal(err)
				}
				if collector.Image != tt.wantImage {
					t.Fatalf("got image %q, want %q", collector.Image, tt.wantImage)
				}
				if got := collector.Resources.Limits.Cpu().String(); got != wantCPULimit {
					t.Fatalf("got cpu limit %q, want %q", got, wantCPULimit)
				}
				foundGatewayEndpoint := false
				foundMonitoringRoleInstance := false
				for _, env := range collector.Env {
					if env.Name == "GENEVA_GATEWAY_ENDPOINT" {
						foundGatewayEndpoint = true
						if env.Value != tt.wantGatewayEndpoint {
							t.Fatalf("got GENEVA_GATEWAY_ENDPOINT %q, want %q", env.Value, tt.wantGatewayEndpoint)
						}
					}
					if env.Name == "MONITORING_ROLE_INSTANCE" {
						foundMonitoringRoleInstance = true
						if env.ValueFrom == nil || env.ValueFrom.FieldRef == nil {
							t.Fatal("expected MONITORING_ROLE_INSTANCE env var to use fieldRef")
						}
						if env.ValueFrom.FieldRef.APIVersion != "v1" {
							t.Fatalf("got MONITORING_ROLE_INSTANCE fieldRef apiVersion %q, want %q", env.ValueFrom.FieldRef.APIVersion, "v1")
						}
						if env.ValueFrom.FieldRef.FieldPath != "spec.nodeName" {
							t.Fatalf("got MONITORING_ROLE_INSTANCE fieldRef path %q, want %q", env.ValueFrom.FieldRef.FieldPath, "spec.nodeName")
						}
					}
				}
				if !foundGatewayEndpoint {
					t.Fatal("expected GENEVA_GATEWAY_ENDPOINT env var to be set")
				}
				if !foundMonitoringRoleInstance {
					t.Fatal("expected MONITORING_ROLE_INSTANCE env var to be set")
				}
				expectedConfigPath := "/etc/otel/" + otelWorkerConfigKey
				if strings.Contains(ds.Name, "master") {
					expectedConfigPath = "/etc/otel/" + otelMasterConfigKey
				}
				if len(collector.Args) != 2 || collector.Args[0] != "--config" || collector.Args[1] != expectedConfigPath {
					t.Fatalf("expected args [--config %s], got %v", expectedConfigPath, collector.Args)
				}
				if ds.Spec.Template.Spec.PriorityClassName != "system-cluster-critical" {
					t.Fatalf("expected PriorityClassName system-cluster-critical, got %q", ds.Spec.Template.Spec.PriorityClassName)
				}
				if _, ok := ds.Spec.Template.Annotations["scheduler.alpha.kubernetes.io/critical-pod"]; ok {
					t.Fatal("expected deprecated critical-pod annotation to be removed")
				}
				if ds.Spec.Template.Spec.DeprecatedServiceAccount != "" {
					t.Fatalf("expected DeprecatedServiceAccount to be empty, got %q", ds.Spec.Template.Spec.DeprecatedServiceAccount)
				}
				if ds.Spec.Template.Spec.AutomountServiceAccountToken == nil || *ds.Spec.Template.Spec.AutomountServiceAccountToken {
					t.Fatal("expected automount service account token to be disabled")
				}
				if collector.SecurityContext == nil {
					t.Fatal("expected collector security context to be set")
				}
				if collector.SecurityContext.Privileged != nil && *collector.SecurityContext.Privileged {
					t.Fatal("expected collector to run without privileged mode")
				}
				if collector.SecurityContext.AllowPrivilegeEscalation == nil || *collector.SecurityContext.AllowPrivilegeEscalation {
					t.Fatal("expected allowPrivilegeEscalation=false")
				}
				if collector.SecurityContext.SeccompProfile == nil || collector.SecurityContext.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
					t.Fatal("expected RuntimeDefault seccomp profile")
				}
				if collector.SecurityContext.SELinuxOptions == nil || collector.SecurityContext.SELinuxOptions.Type != "spc_t" {
					t.Fatal("expected spc_t SELinux context for host journal access")
				}
				if collector.SecurityContext.Capabilities == nil || len(collector.SecurityContext.Capabilities.Drop) != 1 || collector.SecurityContext.Capabilities.Drop[0] != "ALL" {
					t.Fatal("expected all Linux capabilities to be dropped")
				}

				foundGatewayCAConfigMap := false
				foundFileStorageVolume := false
				foundLogVolume := false
				for _, volume := range ds.Spec.Template.Spec.Volumes {
					if volume.Name == "log" && volume.HostPath != nil && volume.HostPath.Path == "/var/log" {
						foundLogVolume = true
					}
					if volume.Name == "gateway-ca-otel-export" && volume.Projected != nil {
						for _, source := range volume.Projected.Sources {
							if source.ConfigMap == nil || source.ConfigMap.Name != otelGatewayCACMName {
								continue
							}
							for _, item := range source.ConfigMap.Items {
								if item.Key == otelGatewayCAKey && item.Path == "ca.crt" {
									foundGatewayCAConfigMap = true
								}
							}
						}
					}
					if volume.Name == "otel-file-storage" && volume.HostPath != nil && volume.HostPath.Path == "/var/lib/otelcol/file_storage" {
						foundFileStorageVolume = true
					}
				}
				if !foundGatewayCAConfigMap {
					t.Fatalf("expected gateway-ca-otel-export volume to project %q key %q to ca.crt", otelGatewayCACMName, otelGatewayCAKey)
				}
				if !foundFileStorageVolume {
					t.Fatal("expected otel-file-storage hostPath volume to be configured")
				}
				if !foundLogVolume {
					t.Fatal("expected /var/log hostPath volume for container, audit, and journald logs")
				}
				foundFileStorageMount := false
				foundLogMount := false
				for _, vm := range collector.VolumeMounts {
					if vm.Name == "otel-file-storage" && vm.MountPath == "/var/lib/otelcol/file_storage" && !vm.ReadOnly {
						foundFileStorageMount = true
					}
					if vm.Name == "log" && vm.MountPath == "/var/log" && vm.ReadOnly {
						foundLogMount = true
					}
				}
				if !foundFileStorageMount {
					t.Fatal("expected otel-file-storage volume mount to be writable and mounted")
				}
				if !foundLogMount {
					t.Fatal("expected read-only /var/log mount for container, audit, and journald logs")
				}
				if ds.Spec.Template.Spec.Affinity == nil || ds.Spec.Template.Spec.Affinity.NodeAffinity == nil || ds.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
					t.Fatal("expected required node affinity for daemonset")
				}
				terms := ds.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
				if strings.Contains(ds.Name, "master") {
					hasMasterExists := false
					hasControlPlaneExists := false
					for _, term := range terms {
						if len(term.MatchExpressions) != 1 {
							continue
						}
						expr := term.MatchExpressions[0]
						if expr.Key == masterRoleLabel && expr.Operator == corev1.NodeSelectorOpExists {
							hasMasterExists = true
						}
						if expr.Key == controlPlaneRoleLabel && expr.Operator == corev1.NodeSelectorOpExists {
							hasControlPlaneExists = true
						}
					}
					if !hasMasterExists || !hasControlPlaneExists {
						t.Fatalf("expected master daemonset affinity to support both %q and %q labels", masterRoleLabel, controlPlaneRoleLabel)
					}
				} else {
					if len(terms) != 1 {
						t.Fatalf("expected single worker affinity term, got %d", len(terms))
					}
					hasMasterDoesNotExist := false
					hasControlPlaneDoesNotExist := false
					for _, expr := range terms[0].MatchExpressions {
						if expr.Key == masterRoleLabel && expr.Operator == corev1.NodeSelectorOpDoesNotExist {
							hasMasterDoesNotExist = true
						}
						if expr.Key == controlPlaneRoleLabel && expr.Operator == corev1.NodeSelectorOpDoesNotExist {
							hasControlPlaneDoesNotExist = true
						}
					}
					if !hasMasterDoesNotExist || !hasControlPlaneDoesNotExist {
						t.Fatalf("expected worker daemonset affinity to exclude both %q and %q labels", masterRoleLabel, controlPlaneRoleLabel)
					}
				}
			}

			assertCommon(masterDS, "300m")
			assertCommon(workerDS, "200m")
		})
	}
}

func TestTelemetryGatewayTarget(t *testing.T) {
	tests := []struct {
		name        string
		cluster     *arov1alpha1.Cluster
		want        string
		wantAliases []corev1.HostAlias
		wantErrText string
	}{
		{
			name: "uses gateway telemetry domain",
			cluster: &arov1alpha1.Cluster{
				Spec: arov1alpha1.ClusterSpec{
					Location:                 "eastus",
					GatewayPrivateEndpointIP: "10.0.0.8",
					GatewayTelemetryDomain:   "telemetry.eastus.aro.azure.com",
				},
			},
			want: "telemetry.eastus.aro.azure.com:4317",
			wantAliases: []corev1.HostAlias{
				{
					IP:        "10.0.0.8",
					Hostnames: []string{"telemetry.eastus.aro.azure.com"},
				},
			},
		},
		{
			name: "uses configured gateway telemetry domain regardless of location",
			cluster: &arov1alpha1.Cluster{
				Spec: arov1alpha1.ClusterSpec{
					Location:                 "eastus",
					GatewayPrivateEndpointIP: "10.0.0.8",
					GatewayTelemetryDomain:   "telemetry.westus.aro.azure.com",
				},
			},
			want: "telemetry.westus.aro.azure.com:4317",
			wantAliases: []corev1.HostAlias{
				{
					IP:        "10.0.0.8",
					Hostnames: []string{"telemetry.westus.aro.azure.com"},
				},
			},
		},
		{
			name: "falls back to private endpoint IP when no gateway telemetry domain available",
			cluster: &arov1alpha1.Cluster{
				Spec: arov1alpha1.ClusterSpec{
					Location:                 "eastus",
					GatewayPrivateEndpointIP: "10.0.0.8",
				},
			},
			want: "10.0.0.8:4317",
		},
		{
			name: "requires gateway private endpoint ip",
			cluster: &arov1alpha1.Cluster{
				Spec: arov1alpha1.ClusterSpec{
					Location: "eastus",
				},
			},
			wantErrText: `geneva logging mode "otel" requires cluster spec field "gatewayPrivateEndpointIP"`,
		},
		{
			name: "validates gateway private endpoint ip",
			cluster: &arov1alpha1.Cluster{
				Spec: arov1alpha1.ClusterSpec{
					Location:                 "eastus",
					GatewayPrivateEndpointIP: "not-an-ip",
					GatewayTelemetryDomain:   "telemetry.eastus.aro.azure.com",
				},
			},
			wantErrText: `invalid cluster spec field "gatewayPrivateEndpointIP": "not-an-ip" is not a valid IP address`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := telemetryGatewayTarget(tt.cluster)
			if tt.wantErrText != "" {
				utilerror.AssertErrorMessage(t, err, tt.wantErrText)
				return
			}
			utilerror.AssertErrorMessage(t, err, "")
			if got.endpoint != tt.want {
				t.Fatalf("got endpoint %q, want %q", got.endpoint, tt.want)
			}
			if !reflect.DeepEqual(got.hostAliases, tt.wantAliases) {
				t.Fatalf("got host aliases %v, want %v", got.hostAliases, tt.wantAliases)
			}
		})
	}
}

func TestClearOTelDaemonSetNodeSelectors(t *testing.T) {
	ctx := context.Background()
	master := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-collector-master",
			Namespace: kubeNamespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"aro.openshift.io/otel-disabled": "true",
					},
				},
			},
		},
	}
	worker := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-collector-worker",
			Namespace: kubeNamespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"some-custom-selector": "value",
					},
				},
			},
		},
	}
	r := &Reconciler{
		AROController: base.AROController{
			Log:    logrus.NewEntry(logrus.StandardLogger()),
			Client: ctrlfake.NewClientBuilder().WithObjects(master, worker).Build(),
			Name:   ControllerName,
		},
	}

	if err := r.clearOTelDaemonSetNodeSelectors(ctx); err != nil {
		t.Fatal(err)
	}

	gotMaster := &appsv1.DaemonSet{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: kubeNamespace, Name: "otel-collector-master"}, gotMaster); err != nil {
		t.Fatal(err)
	}
	if len(gotMaster.Spec.Template.Spec.NodeSelector) != 0 {
		t.Fatalf("expected otel-collector-master nodeSelector to be cleared, got %v", gotMaster.Spec.Template.Spec.NodeSelector)
	}

	gotWorker := &appsv1.DaemonSet{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: kubeNamespace, Name: "otel-collector-worker"}, gotWorker); err != nil {
		t.Fatal(err)
	}
	if len(gotWorker.Spec.Template.Spec.NodeSelector) != 0 {
		t.Fatalf("expected otel-collector-worker nodeSelector to be cleared, got %v", gotWorker.Spec.Template.Spec.NodeSelector)
	}
}
