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

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	mock_deployer "github.com/Azure/ARO-RP/pkg/util/mocks/deployer"
)

func TestGuardRailsReconciler(t *testing.T) {
	tests := []struct {
		name          string
		mocks         func(*mock_deployer.MockDeployer, *arov1alpha1.Cluster)
		flags         arov1alpha1.OperatorFlags
		cleanupNeeded bool
		// errors
		wantErr string
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
					Pullspec:                       "acrtest.example.com/gatekeeper:v3.15.1",
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
			wantErr: "GateKeeper deployment timed out on Ready: context deadline exceeded",
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
			deployer := mock_deployer.NewMockDeployer(controller)
			clientBuilder := ctrlfake.NewClientBuilder().WithObjects(cluster)

			if tt.mocks != nil {
				tt.mocks(deployer, cluster)
			}

			r := &Reconciler{
				log:               logrus.NewEntry(logrus.StandardLogger()),
				deployer:          deployer,
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
