package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	mock_deployer "github.com/Azure/ARO-RP/pkg/util/mocks/deployer"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func clusterVersionForTest(version string) *configv1.ClusterVersion {
	return &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "version"},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{State: configv1.CompletedUpdate, Version: version},
			},
		},
	}
}

func TestGuardRailsReconcilerGatekeeper(t *testing.T) {
	cv := clusterVersionForTest("4.16.0")

	tests := []struct {
		name          string
		mocks         func(*mock_deployer.MockDeployer, *arov1alpha1.Cluster)
		flags         arov1alpha1.OperatorFlags
		cleanupNeeded bool
		wantErr       string
	}{
		{
			name: "disabled",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagFalse,
				operator.GuardrailsDeployManaged: operator.FlagFalse,
				controllerPullSpec:               "wonderfulPullspec",
			},
		},
		{
			name: "managed",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagTrue,
				controllerPullSpec:               "wonderfulPullspec",
				controllerNamespace:              "wonderful-namespace",
				controllerManagerRequestsCPU:     "10m",
				controllerManagerLimitCPU:        "100m",
				controllerManagerRequestsMem:     "512Mi",
				controllerManagerLimitMem:        "512Mi",
				controllerAuditRequestsCPU:       "10m",
				controllerAuditLimitCPU:          "100m",
				controllerAuditRequestsMem:       "512Mi",
				controllerAuditLimitMem:          "512Mi",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.GuardRailsDeploymentConfig{
					Pullspec:                       "wonderfulPullspec",
					Namespace:                      "wonderful-namespace",
					ManagerRequestsCPU:             "10m",
					ManagerLimitCPU:                "100m",
					ManagerRequestsMem:             "512Mi",
					ManagerLimitMem:                "512Mi",
					AuditRequestsCPU:               "10m",
					AuditLimitCPU:                  "100m",
					AuditRequestsMem:               "512Mi",
					AuditLimitMem:                  "512Mi",
					ValidatingWebhookTimeout:       "3",
					ValidatingWebhookFailurePolicy: "Ignore",
					MutatingWebhookTimeout:         "1",
					MutatingWebhookFailurePolicy:   "Ignore",
					RoleSCCResourceName:            "restricted-v2",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "wonderful-namespace", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "wonderful-namespace", "gatekeeper-controller-manager").Return(true, nil)
			},
		},
		{
			name: "managed, no pullspec & namespace (uses default)",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagTrue,
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.GuardRailsDeploymentConfig{
					Pullspec:                       "acrtest.example.com/gatekeeper:v3.19.2",
					Namespace:                      "openshift-azure-guardrails",
					ManagerRequestsCPU:             "100m",
					ManagerLimitCPU:                "1000m",
					ManagerRequestsMem:             "512Mi",
					ManagerLimitMem:                "512Mi",
					AuditRequestsCPU:               "100m",
					AuditLimitCPU:                  "1000m",
					AuditRequestsMem:               "512Mi",
					AuditLimitMem:                  "512Mi",
					ValidatingWebhookTimeout:       "3",
					ValidatingWebhookFailurePolicy: "Ignore",
					MutatingWebhookTimeout:         "1",
					MutatingWebhookFailurePolicy:   "Ignore",
					RoleSCCResourceName:            "restricted-v2",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-controller-manager").Return(true, nil)
			},
		},
		{
			name: "managed, GuardRails does not become ready",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagTrue,
				controllerPullSpec:               "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.GuardRailsDeploymentConfig{
					Pullspec:                       "wonderfulPullspec",
					Namespace:                      "openshift-azure-guardrails",
					ManagerRequestsCPU:             "100m",
					ManagerLimitCPU:                "1000m",
					ManagerRequestsMem:             "512Mi",
					ManagerLimitMem:                "512Mi",
					AuditRequestsCPU:               "100m",
					AuditLimitCPU:                  "1000m",
					AuditRequestsMem:               "512Mi",
					AuditLimitMem:                  "512Mi",
					ValidatingWebhookTimeout:       "3",
					ValidatingWebhookFailurePolicy: "Ignore",
					MutatingWebhookTimeout:         "1",
					MutatingWebhookFailurePolicy:   "Ignore",
					RoleSCCResourceName:            "restricted-v2",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantErr: "GateKeeper deployment timed out on Ready: timed out waiting for the condition",
		},
		{
			name: "managed, CreateOrUpdate() fails",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagTrue,
				controllerPullSpec:               "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.AssignableToTypeOf(&config.GuardRailsDeploymentConfig{})).Return(errors.New("failed ensure"))
			},
			wantErr: "failed ensure",
		},
		{
			name: "managed=false (removal)",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagFalse,
				controllerPullSpec:               "wonderfulPullspec",
			},
			cleanupNeeded: true,
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "managed=false (removal), Remove() fails",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagFalse,
				controllerPullSpec:               "wonderfulPullspec",
			},
			cleanupNeeded: true,
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(errors.New("failed delete"))
			},
			wantErr: "failed delete",
		},
		{
			name: "managed=blank (no action)",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: "",
				controllerPullSpec:               "wonderfulPullspec",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			cluster := &arov1alpha1.Cluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Cluster",
					APIVersion: "aro.openshift.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: tt.flags,
					ACRDomain:     "acrtest.example.com",
				},
			}
			dep := mock_deployer.NewMockDeployer(controller)
			clientBuilder := ctrlfake.NewClientBuilder().WithObjects(cluster, cv)

			if tt.mocks != nil {
				tt.mocks(dep, cluster)
			}

			r := &Reconciler{
				log:               logrus.NewEntry(logrus.StandardLogger()),
				deployer:          dep,
				client:            clientBuilder.Build(),
				readinessTimeout:  0 * time.Second,
				readinessPollTime: 1 * time.Second,
				cleanupNeeded:     tt.cleanupNeeded,
			}
			_, err := r.Reconcile(context.Background(), reconcile.Request{})
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("got error '%v', wanted error '%v'", err, tt.wantErr)
			}

			if err == nil && tt.wantErr != "" {
				t.Errorf("did not get an error, but wanted error '%v'", tt.wantErr)
			}
		})
	}
}

func TestReconcileVAP(t *testing.T) {
	cv := clusterVersionForTest("4.17.0")

	tests := []struct {
		name    string
		flags   arov1alpha1.OperatorFlags
		dhMocks func(*mock_dynamichelper.MockInterface)
		wantErr string
	}{
		{
			name: "VAP: disabled",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagFalse,
				operator.GuardrailsDeployManaged: operator.FlagTrue,
			},
		},
		{
			name: "VAP: managed=true, deploys VAP policies",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:                            operator.FlagTrue,
				operator.GuardrailsDeployManaged:                      operator.FlagTrue,
				operator.GuardrailsPolicyMachineDenyManaged:           operator.FlagTrue,
				operator.GuardrailsPolicyMachineDenyEnforcement:       operator.GuardrailsPolicyDeny,
				operator.GuardrailsPolicyMachineConfigDenyManaged:     operator.FlagTrue,
				operator.GuardrailsPolicyMachineConfigDenyEnforcement: operator.GuardrailsPolicyDryrun,
				operator.GuardrailsPolicyPrivNamespaceDenyManaged:     operator.FlagTrue,
				operator.GuardrailsPolicyPrivNamespaceDenyEnforcement: operator.GuardrailsPolicyWarn,
			},
			dhMocks: func(dh *mock_dynamichelper.MockInterface) {
				// 3 policies + 3 bindings = 6 Ensure calls
				dh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(6)
				// 3 bindings deleted before recreate
				dh.EXPECT().EnsureDeletedGVR(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(3)
			},
		},
		{
			name: "VAP: managed=false, removes all policies",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagFalse,
			},
			dhMocks: func(dh *mock_dynamichelper.MockInterface) {
				// 3 policies × (binding delete + policy delete) = 6 deletions
				dh.EXPECT().EnsureDeletedGVR(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(6)
			},
		},
		{
			name: "VAP: managed=blank, no action",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			cluster := &arov1alpha1.Cluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Cluster",
					APIVersion: "aro.openshift.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: tt.flags,
				},
			}

			dh := mock_dynamichelper.NewMockInterface(controller)
			if tt.dhMocks != nil {
				tt.dhMocks(dh)
			}

			// No gatekeeper is running (kubernetescli is nil)
			r := &Reconciler{
				log:      logrus.NewEntry(logrus.StandardLogger()),
				deployer: mock_deployer.NewMockDeployer(controller),
				client:   ctrlfake.NewClientBuilder().WithObjects(cluster, cv).Build(),
				dh:       dh,
			}
			_, err := r.Reconcile(context.Background(), reconcile.Request{})
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("got error '%v', wanted error '%v'", err, tt.wantErr)
			}

			if err == nil && tt.wantErr != "" {
				t.Errorf("did not get an error, but wanted error '%v'", tt.wantErr)
			}
		})
	}
}

func TestVapValidationAction(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"deny", "Deny"},
		{"Deny", "Deny"},
		{"warn", "Warn"},
		{"Warn", "Warn"},
		{"dryrun", "Audit"},
		{"DryRun", "Audit"},
		{"", "Deny"},
		{"unknown", "Deny"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := vapValidationAction(tt.input)
			if got != tt.want {
				t.Errorf("vapValidationAction(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
